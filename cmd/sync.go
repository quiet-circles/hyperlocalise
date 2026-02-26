package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/quiet-circles/hyperlocalise/internal/config"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/localstore"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/storage"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/storage/poeditor"
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
	cmd.Flags().StringVar(&o.output, "output", o.output, "output format: text or json")
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
	if err := reg.Register(poeditor.AdapterName, poeditor.New); err != nil {
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
			"action=%s creates=%d updates=%d unchanged=%d conflicts=%d applied=%d skipped=%d warnings=%d\n",
			report.Action,
			len(report.Creates),
			len(report.Updates),
			len(report.Unchanged),
			len(report.Conflicts),
			len(report.Applied),
			len(report.Skipped),
			len(report.Warnings),
		)
		return err
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	default:
		return fmt.Errorf("unsupported output format %q", output)
	}
}

func backgroundContext() context.Context {
	return context.Background()
}
