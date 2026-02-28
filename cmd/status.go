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

			locales, err := resolveStatusLocales(cfg, o.locales, o.group)
			if err != nil {
				return err
			}
			if len(locales) == 0 {
				return fmt.Errorf("no locales selected")
			}

			bucketFiles, err := statusBucketFiles(cfg, o.bucket)
			if err != nil {
				return err
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
			snapshot.Entries = filterByLocaleAndBucket(snapshot.Entries, locales, o.bucket, bucketFiles)

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

func resolveStatusLocales(cfg *config.I18NConfig, requestedLocales []string, group string) ([]string, error) {
	locales := append([]string(nil), requestedLocales...)
	if len(locales) == 0 {
		locales = append([]string(nil), cfg.Locales.Targets...)
	}
	if group == "" {
		return locales, nil
	}

	g, ok := cfg.Groups[group]
	if !ok {
		return nil, fmt.Errorf("unknown group %q", group)
	}
	if len(g.Targets) == 0 {
		return locales, nil
	}
	if len(requestedLocales) == 0 {
		return append([]string(nil), g.Targets...), nil
	}

	targetSet := make(map[string]struct{}, len(g.Targets))
	for _, target := range g.Targets {
		targetSet[target] = struct{}{}
	}

	var intersection []string
	for _, locale := range requestedLocales {
		if _, ok := targetSet[locale]; ok {
			intersection = append(intersection, locale)
		}
	}
	if len(intersection) == 0 {
		return nil, fmt.Errorf("no locales matched group %q", group)
	}

	return intersection, nil
}

func statusBucketFiles(cfg *config.I18NConfig, bucket string) (map[string]struct{}, error) {
	if bucket == "" {
		return nil, nil
	}
	b, ok := cfg.Buckets[bucket]
	if !ok {
		return nil, fmt.Errorf("unknown bucket %q", bucket)
	}

	files := make(map[string]struct{}, len(b.Files))
	for _, f := range b.Files {
		files[f.From] = struct{}{}
	}

	return files, nil
}

func filterByLocaleAndBucket(entries []storage.Entry, locales []string, bucket string, bucketFiles map[string]struct{}) []storage.Entry {
	var filtered []storage.Entry

	for _, e := range entries {
		if !slices.Contains(locales, e.Locale) {
			continue
		}
		if bucket != "" {
			if _, ok := bucketFiles[e.Namespace]; !ok {
				continue
			}
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

func sortedLocales(entries map[string]storage.Entry) []string {
	locales := make([]string, 0, len(entries))
	for locale := range entries {
		locales = append(locales, locale)
	}
	slices.Sort(locales)
	return locales
}

func writeStatusCSV(w io.Writer, entries []storage.Entry, _ string) error {
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
		for _, locale := range sortedLocales(byLocale) {
			entry := byLocale[locale]
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
