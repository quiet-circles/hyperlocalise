package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/quiet-circles/hyperlocalise/internal/config"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/localstore"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/storage"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/storage/bootstrap"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/storageregistry"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/syncsvc"
	"github.com/spf13/cobra"
)

type syncCommonOptions struct {
	configPath            string
	locales               []string
	dryRun                bool
	output                string
	failOnConflict        bool
	applyCuratedOverDraft bool
}

func defaultSyncCommonOptions() syncCommonOptions {
	return syncCommonOptions{
		dryRun:                true,
		output:                "text",
		failOnConflict:        true,
		applyCuratedOverDraft: true,
	}
}

func newSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "synchronize translations with remote storage adapters",
	}

	cmd.AddCommand(newSyncPullCmd())
	cmd.AddCommand(newSyncPushCmd())

	return cmd
}

func addSyncCommonFlags(cmd *cobra.Command, o *syncCommonOptions) {
	cmd.Flags().StringVar(&o.configPath, "config", "", "path to i18n config")
	cmd.Flags().StringSliceVar(&o.locales, "locale", nil, "target locale(s) to sync")
	cmd.Flags().BoolVar(&o.dryRun, "dry-run", o.dryRun, "preview changes without applying")
	cmd.Flags().StringVar(&o.output, "output", o.output, "output format: text, json, or markdown")
	cmd.Flags().BoolVar(&o.failOnConflict, "fail-on-conflict", o.failOnConflict, "return error if conflicts are detected")
	cmd.Flags().BoolVar(&o.applyCuratedOverDraft, "apply-curated-over-draft", o.applyCuratedOverDraft, "allow pull to update local draft entries with curated remote values")
}

type syncRuntime struct {
	cfg    *config.I18NConfig
	local  *localstore.JSONStore
	remote storage.StorageAdapter
	svc    *syncsvc.Service
}

func newSyncRuntime(configPath string) (*syncRuntime, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}
	if cfg.Storage == nil {
		return nil, fmt.Errorf("storage config is required: add top-level \"storage\" with adapter and config")
	}

	reg := storageregistry.New()
	if err := bootstrap.RegisterBuiltins(reg); err != nil {
		return nil, err
	}

	adapter, err := reg.New(cfg.Storage.Adapter, cfg.Storage.Config)
	if err != nil {
		return nil, err
	}

	local, err := localstore.NewJSONStore(cfg)
	if err != nil {
		return nil, err
	}

	return &syncRuntime{
		cfg:    cfg,
		local:  local,
		remote: adapter,
		svc:    syncsvc.New(),
	}, nil
}

func writeSyncReport(cmd *cobra.Command, report syncsvc.Report, output string) error {
	switch strings.ToLower(strings.TrimSpace(output)) {
	case "", "text":
		_, err := fmt.Fprintf(
			cmd.OutOrStdout(),
			"action=%s creates=%d updates=%d unchanged=%d conflicts=%d risky=%d applied=%d skipped=%d warnings=%d\n",
			report.Action,
			len(report.Creates),
			len(report.Updates),
			len(report.Unchanged),
			len(report.Conflicts),
			len(report.Risky),
			len(report.Applied),
			len(report.Skipped),
			len(report.Warnings),
		)
		return err
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	case "md", "markdown":
		return writeSyncMarkdown(cmd.OutOrStdout(), report)
	default:
		return fmt.Errorf("unsupported output format %q", output)
	}
}

type reviewChange struct {
	Type      string
	Locale    string
	Key       string
	Context   string
	Namespace string
	Category  string
	Detail    string
}

func writeSyncMarkdown(w io.Writer, report syncsvc.Report) error {
	var b strings.Builder
	b.WriteString("## Translation Diff Summary\n\n")
	fmt.Fprintf(&b, "- Action: `%s`\n", report.Action)
	fmt.Fprintf(&b, "- Creates: `%d`\n", len(report.Creates))
	fmt.Fprintf(&b, "- Updates: `%d`\n", len(report.Updates))
	fmt.Fprintf(&b, "- Conflicts: `%d`\n", len(report.Conflicts))
	fmt.Fprintf(&b, "- Risky changes: `%d`\n", len(report.Risky))
	fmt.Fprintf(&b, "- Warnings: `%d`\n\n", len(report.Warnings))

	if len(report.Risky) > 0 {
		b.WriteString("### Risk Highlights\n\n")
		b.WriteString("| Locale | Key | Risk | Detail |\n")
		b.WriteString("| --- | --- | --- | --- |\n")
		for _, risk := range report.Risky {
			detail := risk.Message
			if risk.Code == syncsvc.RiskCodeLengthSpike {
				detail = fmt.Sprintf("%s (%d -> %d, x%.2f)", risk.Message, risk.BaselineLength, risk.CandidateLength, risk.Ratio)
			}
			fmt.Fprintf(&b, "| `%s` | `%s` | `%s` | %s |\n", risk.ID.Locale, formatKeyWithContext(risk.ID.Key, risk.ID.Context), risk.Code, detail)
		}
		b.WriteString("\n")
	}

	changes := buildReviewChanges(report)
	if len(changes) == 0 {
		b.WriteString("### Diffs by Locale and Key Category\n\n")
		b.WriteString("No creates, updates, or conflicts.\n")
		_, err := io.WriteString(w, b.String())
		return err
	}

	riskByID := mapRiskCodesByEntryID(report.Risky)
	b.WriteString("### Diffs by Locale and Key Category\n\n")
	grouped := groupChangesByLocaleAndCategory(changes)
	locales := make([]string, 0, len(grouped))
	for locale := range grouped {
		locales = append(locales, locale)
	}
	slices.Sort(locales)

	for _, locale := range locales {
		fmt.Fprintf(&b, "#### `%s`\n\n", locale)
		cats := grouped[locale]
		categories := make([]string, 0, len(cats))
		for category := range cats {
			categories = append(categories, category)
		}
		slices.Sort(categories)
		for _, category := range categories {
			fmt.Fprintf(&b, "- **%s**\n", category)
			for _, change := range cats[category] {
				riskLabel := ""
				if codes := riskByID[entryIDLabel(change.Locale, change.Key, change.Context)]; len(codes) > 0 {
					riskLabel = " [RISK: " + strings.Join(codes, ", ") + "]"
				}
				fmt.Fprintf(
					&b,
					"  - `%s` `%s`%s%s\n",
					change.Type,
					formatKeyWithContext(change.Key, change.Context),
					change.Detail,
					riskLabel,
				)
			}
		}
		b.WriteString("\n")
	}

	_, err := io.WriteString(w, b.String())
	return err
}

func buildReviewChanges(report syncsvc.Report) []reviewChange {
	changes := make([]reviewChange, 0, len(report.Creates)+len(report.Updates)+len(report.Conflicts))
	for _, e := range report.Creates {
		changes = append(changes, reviewChange{
			Type:      "create",
			Locale:    e.Locale,
			Key:       e.Key,
			Context:   e.Context,
			Namespace: e.Namespace,
			Category:  keyCategory(e.Namespace, e.Key),
		})
	}
	for _, e := range report.Updates {
		changes = append(changes, reviewChange{
			Type:      "update",
			Locale:    e.Locale,
			Key:       e.Key,
			Context:   e.Context,
			Namespace: e.Namespace,
			Category:  keyCategory(e.Namespace, e.Key),
		})
	}
	for _, c := range report.Conflicts {
		changes = append(changes, reviewChange{
			Type:     "conflict",
			Locale:   c.ID.Locale,
			Key:      c.ID.Key,
			Context:  c.ID.Context,
			Category: keyCategory("", c.ID.Key),
			Detail:   " (reason: " + c.Reason + ")",
		})
	}

	slices.SortFunc(changes, func(a, b reviewChange) int {
		if c := strings.Compare(a.Locale, b.Locale); c != 0 {
			return c
		}
		if c := strings.Compare(a.Category, b.Category); c != 0 {
			return c
		}
		if c := strings.Compare(a.Type, b.Type); c != 0 {
			return c
		}
		if c := strings.Compare(a.Key, b.Key); c != 0 {
			return c
		}
		return strings.Compare(a.Context, b.Context)
	})

	return changes
}

func groupChangesByLocaleAndCategory(changes []reviewChange) map[string]map[string][]reviewChange {
	grouped := map[string]map[string][]reviewChange{}
	for _, c := range changes {
		if grouped[c.Locale] == nil {
			grouped[c.Locale] = map[string][]reviewChange{}
		}
		grouped[c.Locale][c.Category] = append(grouped[c.Locale][c.Category], c)
	}
	return grouped
}

func keyCategory(namespace, key string) string {
	prefix := keyPrefix(key)
	if namespace == "" {
		if prefix == "" {
			return "uncategorized"
		}
		return prefix
	}
	if prefix == "" {
		return namespace
	}
	return namespace + "/" + prefix
}

func keyPrefix(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	parts := strings.FieldsFunc(key, func(r rune) bool {
		switch r {
		case '.', '/', ':', '_':
			return true
		default:
			return false
		}
	})
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func mapRiskCodesByEntryID(risks []syncsvc.RiskChange) map[string][]string {
	byID := make(map[string][]string, len(risks))
	for _, risk := range risks {
		id := entryIDLabel(risk.ID.Locale, risk.ID.Key, risk.ID.Context)
		if slices.Contains(byID[id], risk.Code) {
			continue
		}
		byID[id] = append(byID[id], risk.Code)
		slices.Sort(byID[id])
	}
	return byID
}

func entryIDLabel(locale, key, context string) string {
	return locale + "\x00" + key + "\x00" + context
}

func formatKeyWithContext(key, context string) string {
	if strings.TrimSpace(context) == "" {
		return key
	}
	return key + " [" + context + "]"
}

func backgroundContext() context.Context {
	return context.Background()
}
