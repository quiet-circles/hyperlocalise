package cmd

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/quiet-circles/hyperlocalise/internal/config"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/localstore"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/storage"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/syncsvc"
	"github.com/spf13/cobra"
)

type statusOptions struct {
	configPath string
	locales    []string
	output     string
	group      string
	bucket     string
}

func defaultStatusOptions() statusOptions {
	return statusOptions{
		output: "csv",
	}
}

func newStatusCmd() *cobra.Command {
	o := defaultStatusOptions()

	cmd := &cobra.Command{
		Use:   "status",
		Short: "show translation status by locale",
		Long: `Shows translation status for each locale as CSV.
Status values:
  - translated: has a non-empty translation value
  - needs_review: LLM-generated translation not yet curated
  - untranslated: empty translation value`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(o.configPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			locales := o.locales
			if len(locales) == 0 {
				locales = cfg.Locales.Targets
			}

			if o.group != "" {
				g, ok := cfg.Groups[o.group]
				if !ok {
					return fmt.Errorf("unknown group %q", o.group)
				}
				if len(locales) == 0 || o.group != "" {
					locales = g.Targets
				}
			}

			store, err := localstore.NewJSONStore(cfg)
			if err != nil {
				return fmt.Errorf("init local store: %w", err)
			}

			snapshot, err := store.ReadSnapshot(context.Background(), syncsvc.LocalReadRequest{
				Locales: locales,
			})
			if err != nil {
				return fmt.Errorf("read local translations: %w", err)
			}

			slices.Sort(locales)
			snapshot.Entries = filterByLocaleAndBucket(snapshot.Entries, locales, o.bucket, cfg)

			switch strings.ToLower(o.output) {
			case "csv":
				return writeStatusCSV(cmd.OutOrStdout(), snapshot.Entries, cfg.Locales.Source)
			default:
				return fmt.Errorf("unsupported output format %q", o.output)
			}
		},
	}

	cmd.Flags().StringVar(&o.configPath, "config", "", "path to i18n config")
	cmd.Flags().StringSliceVar(&o.locales, "locale", nil, "target locale(s) to report")
	cmd.Flags().StringVar(&o.output, "output", o.output, "output format: csv")
	cmd.Flags().StringVar(&o.group, "group", "", "filter by group name")
	cmd.Flags().StringVar(&o.bucket, "bucket", "", "filter by bucket name")

	return cmd
}

func filterByLocaleAndBucket(entries []storage.Entry, locales []string, bucket string, cfg *config.I18NConfig) []storage.Entry {
	var filtered []storage.Entry

	bucketFiles := make(map[string]bool)
	if bucket != "" {
		if b, ok := cfg.Buckets[bucket]; ok {
			for _, f := range b.Files {
				bucketFiles[f.From] = true
			}
		}
	}

	for _, e := range entries {
		if !slices.Contains(locales, e.Locale) {
			continue
		}
		if bucket != "" && e.Namespace != "" && !bucketFiles[e.Namespace] {
			continue
		}
		filtered = append(filtered, e)
	}

	return filtered
}

func computeStatus(entry storage.Entry) string {
	if strings.TrimSpace(entry.Value) == "" {
		return "untranslated"
	}
	if entry.Provenance.Origin == storage.OriginLLM && entry.Provenance.State == storage.StateDraft {
		return "needs_review"
	}
	return "translated"
}

func writeStatusCSV(w io.Writer, entries []storage.Entry, sourceLocale string) error {
	records := [][]string{
		{"key", "namespace", "locale", "status", "origin", "state"},
	}

	byKeyLocale := make(map[string]map[string]storage.Entry)
	for _, e := range entries {
		if byKeyLocale[e.Key] == nil {
			byKeyLocale[e.Key] = make(map[string]storage.Entry)
		}
		byKeyLocale[e.Key][e.Locale] = e
	}

	keys := make([]string, 0, len(byKeyLocale))
	for k := range byKeyLocale {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	for _, key := range keys {
		byLocale := byKeyLocale[key]
		for _, entry := range byLocale {
			records = append(records, []string{
				entry.Key,
				entry.Namespace,
				entry.Locale,
				computeStatus(entry),
				entry.Provenance.Origin,
				entry.Provenance.State,
			})
		}
	}

	csvWriter := csv.NewWriter(w)
	if err := csvWriter.WriteAll(records); err != nil {
		return fmt.Errorf("write csv: %w", err)
	}

	return nil
}
