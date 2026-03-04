package cmd

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/quiet-circles/hyperlocalise/internal/config"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/pathresolver"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/storage"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/syncsvc"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/translationfileparser"
	"github.com/spf13/cobra"
)

type statusOptions struct {
	configPath  string
	locales     []string
	output      string
	group       string
	bucket      string
	interactive bool
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

			buckets, err := selectedStatusBuckets(cfg, o.group, o.bucket)
			if err != nil {
				return err
			}

			entries, err := collectStatusEntries(context.Background(), cfg, syncsvc.LocalReadRequest{
				Locales: locales,
			}, buckets)
			if err != nil {
				return fmt.Errorf("read local translations: %w", err)
			}

			slices.Sort(locales)
			entries = filterByLocale(entries, locales)

			if o.interactive {
				if strings.ToLower(o.output) != "csv" {
					return fmt.Errorf("unsupported output format %q for --interactive (must be csv)", o.output)
				}
				return runStatusDashboard(cmd.OutOrStdout(), entries, locales, o.group, o.bucket)
			}

			switch strings.ToLower(o.output) {
			case "csv":
				return writeStatusCSV(cmd.OutOrStdout(), entries, cfg.Locales.Source)
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
	cmd.Flags().BoolVarP(&o.interactive, "interactive", "i", false, "render interactive status dashboard in TTY")

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

func selectedStatusBuckets(cfg *config.I18NConfig, group, bucket string) ([]string, error) {
	if bucket == "" {
		if group != "" {
			g, ok := cfg.Groups[group]
			if !ok {
				return nil, fmt.Errorf("unknown group %q", group)
			}
			if len(g.Buckets) > 0 {
				for _, name := range g.Buckets {
					if _, ok := cfg.Buckets[name]; !ok {
						return nil, fmt.Errorf("group %q references unknown bucket %q", group, name)
					}
				}
				return append([]string(nil), g.Buckets...), nil
			}
		}

		names := make([]string, 0, len(cfg.Buckets))
		for name := range cfg.Buckets {
			names = append(names, name)
		}
		slices.Sort(names)
		return names, nil
	}
	if _, ok := cfg.Buckets[bucket]; !ok {
		return nil, fmt.Errorf("unknown bucket %q", bucket)
	}
	if group != "" {
		g, ok := cfg.Groups[group]
		if !ok {
			return nil, fmt.Errorf("unknown group %q", group)
		}
		if len(g.Buckets) > 0 && !slices.Contains(g.Buckets, bucket) {
			return nil, fmt.Errorf("bucket %q is not part of group %q", bucket, group)
		}
	}
	return []string{bucket}, nil
}

// statusBucketFiles is kept for backwards compatibility with existing tests.
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

func collectStatusEntries(_ context.Context, cfg *config.I18NConfig, req syncsvc.LocalReadRequest, buckets []string) ([]storage.Entry, error) {
	parser := translationfileparser.NewDefaultStrategy()
	locales := req.Locales
	if len(locales) == 0 {
		locales = append([]string(nil), cfg.Locales.Targets...)
	}

	entries := make([]storage.Entry, 0)
	for _, bucketName := range buckets {
		bucket := cfg.Buckets[bucketName]
		for _, file := range bucket.Files {
			sourcePattern := pathresolver.ResolveSourcePath(file.From, cfg.Locales.Source)
			sourcePaths, err := resolveSourcePathsForStatus(sourcePattern)
			if err != nil {
				return nil, fmt.Errorf("resolve source paths for %q: %w", sourcePattern, err)
			}
			for _, sourcePath := range sourcePaths {
				if shouldIgnoreSourcePathForStatus(sourcePath, cfg.Locales.Targets) {
					continue
				}

				sourceEntries, err := readEntriesForStatus(parser, sourcePath)
				sourceMissing := false
				if err != nil {
					if os.IsNotExist(err) {
						sourceMissing = true
						sourceEntries = map[string]string{}
					} else {
						return nil, err
					}
				}

				targetByLocale := make(map[string]map[string]string, len(locales))
				keySet := make(map[string]struct{}, len(sourceEntries))
				for key := range sourceEntries {
					keySet[key] = struct{}{}
				}

				for _, locale := range locales {
					targetPattern := pathresolver.ResolveTargetPath(file.To, cfg.Locales.Source, locale)
					targetPath, err := resolveTargetPathForStatus(sourcePattern, targetPattern, sourcePath)
					if err != nil {
						return nil, fmt.Errorf("resolve target path for source %q: %w", sourcePath, err)
					}
					targetEntries, err := readTargetEntriesForStatus(parser, sourcePath, targetPath)
					if err != nil {
						if !os.IsNotExist(err) {
							return nil, err
						}
						targetEntries = map[string]string{}
					}
					targetByLocale[locale] = targetEntries
					if sourceMissing {
						for key := range targetEntries {
							keySet[key] = struct{}{}
						}
					}
				}

				keys := make([]string, 0, len(keySet))
				for key := range keySet {
					keys = append(keys, key)
				}
				slices.Sort(keys)

				for _, locale := range locales {
					targetEntries := targetByLocale[locale]
					for _, key := range keys {
						entries = append(entries, storage.Entry{
							Key:       key,
							Namespace: file.From,
							Locale:    locale,
							Value:     targetEntries[key],
							Provenance: storage.EntryProvenance{
								Origin: storage.OriginUnknown,
							},
						})
					}
				}
			}
		}
	}

	return entries, nil
}

func readEntriesForStatus(parser *translationfileparser.Strategy, path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	entries, err := parser.Parse(path, content)
	if err != nil {
		return nil, fmt.Errorf("parse translation file %q: %w", path, err)
	}
	return entries, nil
}

func readTargetEntriesForStatus(parser *translationfileparser.Strategy, sourcePath, targetPath string) (map[string]string, error) {
	ext := strings.ToLower(filepath.Ext(targetPath))
	if ext == ".md" || ext == ".mdx" {
		sourceContent, err := os.ReadFile(sourcePath)
		if err != nil {
			return nil, err
		}
		targetContent, err := os.ReadFile(targetPath)
		if err != nil {
			return nil, err
		}
		return translationfileparser.AlignMarkdownTargetToSource(sourceContent, targetContent), nil
	}
	return readEntriesForStatus(parser, targetPath)
}

func filterByLocale(entries []storage.Entry, locales []string) []storage.Entry {
	var filtered []storage.Entry

	for _, e := range entries {
		if !slices.Contains(locales, e.Locale) {
			continue
		}
		filtered = append(filtered, e)
	}

	return filtered
}

// filterByLocaleAndBucket is kept for backwards compatibility with existing tests.
func filterByLocaleAndBucket(entries []storage.Entry, locales []string, bucket string, bucketFiles map[string]struct{}) []storage.Entry {
	filtered := filterByLocale(entries, locales)
	if bucket == "" {
		return filtered
	}
	withBucket := make([]storage.Entry, 0, len(filtered))
	for _, e := range filtered {
		if _, ok := bucketFiles[e.Namespace]; ok {
			withBucket = append(withBucket, e)
		}
	}
	return withBucket
}

func shouldIgnoreSourcePathForStatus(sourcePath string, targetLocales []string) bool {
	normalized := filepath.ToSlash(sourcePath)
	segments := strings.Split(normalized, "/")
	if len(segments) < 2 {
		return false
	}

	targets := make(map[string]struct{}, len(targetLocales))
	for _, locale := range targetLocales {
		targets[locale] = struct{}{}
	}

	for i := 1; i < len(segments)-1; i++ {
		if _, ok := targets[segments[i]]; ok {
			return true
		}
	}
	return false
}

func resolveSourcePathsForStatus(sourcePattern string) ([]string, error) {
	if !strings.ContainsAny(sourcePattern, "*?[") {
		return []string{sourcePattern}, nil
	}
	if !strings.Contains(sourcePattern, "**") {
		matches, err := filepath.Glob(sourcePattern)
		if err != nil {
			return nil, err
		}
		slices.Sort(matches)
		return matches, nil
	}

	normalizedPattern := filepath.ToSlash(sourcePattern)
	re, err := globToRegexForStatus(normalizedPattern)
	if err != nil {
		return nil, err
	}

	baseDir := baseDirForDoublestarForStatus(sourcePattern)
	matches := make([]string, 0)
	err = filepath.WalkDir(baseDir, func(candidate string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if re.MatchString(filepath.ToSlash(candidate)) {
			matches = append(matches, candidate)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	slices.Sort(matches)
	return matches, nil
}

func resolveTargetPathForStatus(sourcePattern, targetPattern, sourcePath string) (string, error) {
	if !strings.ContainsAny(sourcePattern, "*?[") {
		return targetPattern, nil
	}
	if !strings.ContainsAny(targetPattern, "*?[") {
		return "", fmt.Errorf("target pattern %q must include glob tokens when source pattern %q includes globs", targetPattern, sourcePattern)
	}
	sourceBase := globBaseDirForStatus(sourcePattern)
	targetBase := globBaseDirForStatus(targetPattern)
	relative, err := filepath.Rel(sourceBase, sourcePath)
	if err != nil {
		return "", err
	}
	return filepath.Join(targetBase, relative), nil
}

func baseDirForDoublestarForStatus(pattern string) string {
	normalized := filepath.ToSlash(pattern)
	idx := strings.Index(normalized, "**")
	if idx == -1 {
		return filepath.Dir(pattern)
	}
	prefix := strings.TrimSuffix(normalized[:idx], "/")
	if prefix == "" {
		return "."
	}
	return filepath.FromSlash(prefix)
}

func globBaseDirForStatus(pattern string) string {
	idx := strings.IndexAny(filepath.ToSlash(pattern), "*?[")
	if idx == -1 {
		return filepath.Dir(pattern)
	}
	prefix := filepath.ToSlash(pattern)[:idx]
	prefix = strings.TrimSuffix(prefix, "/")
	if prefix == "" {
		return "."
	}
	return filepath.FromSlash(prefix)
}

func globToRegexForStatus(pattern string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); {
		switch pattern[i] {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				if i+2 < len(pattern) && pattern[i+2] == '/' {
					b.WriteString("(?:.*/)?")
					i += 3
					continue
				}
				b.WriteString(".*")
				i += 2
				continue
			}
			b.WriteString("[^/]*")
		case '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(pattern[i : i+1]))
		}
		i++
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
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

	byNamespaceKeyLocale := make(map[string]map[string]storage.Entry)
	for _, e := range entries {
		composite := e.Namespace + "\x1f" + e.Key
		if byNamespaceKeyLocale[composite] == nil {
			byNamespaceKeyLocale[composite] = make(map[string]storage.Entry)
		}
		byNamespaceKeyLocale[composite][e.Locale] = e
	}

	compositeKeys := make([]string, 0, len(byNamespaceKeyLocale))
	for k := range byNamespaceKeyLocale {
		compositeKeys = append(compositeKeys, k)
	}
	slices.Sort(compositeKeys)

	for _, composite := range compositeKeys {
		byLocale := byNamespaceKeyLocale[composite]
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
