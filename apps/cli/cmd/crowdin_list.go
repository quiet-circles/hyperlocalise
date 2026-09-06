package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	crowdinstorage "github.com/hyperlocalise/hyperlocalise/internal/i18n/storage/crowdin"
	"github.com/spf13/cobra"
)

type crowdinListOptions struct {
	configPath   string
	identityPath string
	branch       string
	languages    []string
	output       string
	all          bool
	name         string
}

func newCrowdinConfigSourcesCmd() *cobra.Command {
	o := crowdinListOptions{output: "text"}
	cmd := &cobra.Command{
		Use:          "sources",
		Short:        "list source files resolved from crowdin.yml",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, _, err := loadCrowdinWorkflowConfig(o.configPath, o.identityPath)
			if err != nil {
				return err
			}
			rows, err := crowdinstorage.ListConfiguredSources(cfg)
			if err != nil {
				return err
			}
			return writeEncodedOutput(cmd.OutOrStdout(), o.output, func() error {
				for _, row := range rows {
					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "path=%s\n", row.Path); err != nil {
						return err
					}
				}
				return nil
			}, rows)
		},
	}
	addCrowdinListFlags(cmd, &o, false, false)
	return cmd
}

func newCrowdinConfigTranslationsCmd() *cobra.Command {
	o := crowdinListOptions{output: "text"}
	cmd := &cobra.Command{
		Use:          "translations",
		Short:        "list translation files resolved from crowdin.yml",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, _, err := loadCrowdinWorkflowConfig(o.configPath, o.identityPath)
			if err != nil {
				return err
			}
			clientCfg, _, err := crowdinstorage.LoadClientConfig(o.configPath, o.identityPath)
			if err != nil {
				return err
			}
			resolver, err := newCrowdinLocaleResolver(clientCfg)
			if err != nil {
				return err
			}
			locales, err := resolver.ResolveLocales(cmd.Context(), clientCfg.ProjectID, o.languages)
			if err != nil {
				return err
			}
			rows, err := crowdinstorage.ListConfiguredTranslationPaths(cfg, locales)
			if err != nil {
				return err
			}
			return writeEncodedOutput(cmd.OutOrStdout(), o.output, func() error {
				for _, row := range rows {
					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "language=%s path=%s\n", row.LanguageID, row.Path); err != nil {
						return err
					}
				}
				return nil
			}, rows)
		},
	}
	addCrowdinListFlags(cmd, &o, true, false)
	return cmd
}

func newCrowdinBranchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "branch",
		Short: "Crowdin branch commands",
	}
	cmd.AddCommand(newCrowdinBranchListCmd())
	cmd.AddCommand(newCrowdinBranchAddCmd())
	return cmd
}

func newCrowdinBranchListCmd() *cobra.Command {
	o := crowdinListOptions{output: "text"}
	cmd := &cobra.Command{
		Use:          "list",
		Short:        "list Crowdin project branches",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, _, err := crowdinstorage.LoadClientConfig(o.configPath, o.identityPath)
			if err != nil {
				return err
			}
			client, err := newCrowdinBranchLister(cfg)
			if err != nil {
				return err
			}
			rows, err := client.ListBranches(cmd.Context(), cfg.ProjectID)
			if err != nil {
				return err
			}
			return writeEncodedOutput(cmd.OutOrStdout(), o.output, func() error {
				for _, row := range rows {
					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "id=%d name=%s\n", row.ID, row.Name); err != nil {
						return err
					}
				}
				return nil
			}, rows)
		},
	}
	addCrowdinListFlags(cmd, &o, false, false)
	return cmd
}

func newCrowdinBranchAddCmd() *cobra.Command {
	o := crowdinListOptions{}
	cmd := &cobra.Command{
		Use:          "add",
		Short:        "create a Crowdin project branch",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, _, err := crowdinstorage.LoadClientConfig(o.configPath, o.identityPath)
			if err != nil {
				return err
			}
			client, err := newCrowdinBranchAdder(cfg)
			if err != nil {
				return err
			}
			row, err := client.AddBranch(cmd.Context(), cfg.ProjectID, o.name)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "id=%d name=%s\n", row.ID, row.Name)
			return err
		},
	}
	cmd.Flags().StringVar(&o.configPath, "config", "", "path to crowdin.yml")
	cmd.Flags().StringVar(&o.identityPath, "identity", "", "path to Crowdin identity file")
	cmd.Flags().StringVar(&o.name, "name", "", "Crowdin branch name")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newCrowdinFileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "file",
		Short: "Crowdin source file commands",
	}
	cmd.AddCommand(newCrowdinFileListCmd())
	cmd.AddCommand(newCrowdinFileUploadCmd())
	cmd.AddCommand(newCrowdinFileDownloadCmd())
	cmd.AddCommand(newCrowdinFileDeleteCmd())
	return cmd
}

func newCrowdinFileListCmd() *cobra.Command {
	o := crowdinListOptions{output: "text"}
	cmd := &cobra.Command{
		Use:          "list",
		Short:        "list Crowdin source files",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, _, err := crowdinstorage.LoadClientConfig(o.configPath, o.identityPath)
			if err != nil {
				return err
			}
			cfg = applyCrowdinBranchFlag(cfg, o.branch)
			client, err := newCrowdinFileLister(cfg)
			if err != nil {
				return err
			}
			rows, err := client.ListProjectFiles(cmd.Context(), cfg.ProjectID, cfg.Branch)
			if err != nil {
				return err
			}
			return writeEncodedOutput(cmd.OutOrStdout(), o.output, func() error {
				for _, row := range rows {
					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "id=%d name=%s path=%s\n", row.ID, row.Name, row.Path); err != nil {
						return err
					}
				}
				return nil
			}, rows)
		},
	}
	addCrowdinListFlags(cmd, &o, false, true)
	return cmd
}

func newCrowdinLanguageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "language",
		Short: "Crowdin language commands",
	}
	cmd.AddCommand(newCrowdinLanguageListCmd())
	return cmd
}

func newCrowdinLanguageListCmd() *cobra.Command {
	o := crowdinListOptions{output: "text"}
	cmd := &cobra.Command{
		Use:          "list",
		Short:        "list Crowdin languages",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, _, err := crowdinstorage.LoadClientConfig(o.configPath, o.identityPath)
			if err != nil {
				return err
			}
			client, err := newCrowdinLanguageLister(cfg)
			if err != nil {
				return err
			}
			var rows []crowdinstorage.ProjectLanguage
			if o.all {
				rows, err = client.ListAllLanguages(cmd.Context())
			} else {
				rows, err = client.ListProjectLanguages(cmd.Context(), cfg.ProjectID)
			}
			if err != nil {
				return err
			}
			return writeEncodedOutput(cmd.OutOrStdout(), o.output, func() error {
				for _, row := range rows {
					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "id=%s name=%s locale=%s\n", row.ID, row.Name, row.Locale); err != nil {
						return err
					}
				}
				return nil
			}, rows)
		},
	}
	addCrowdinListFlags(cmd, &o, false, false)
	cmd.Flags().BoolVar(&o.all, "all", false, "list all account languages instead of project target languages")
	return cmd
}

func newCrowdinGlossaryListCmd() *cobra.Command {
	o := crowdinListOptions{output: "text"}
	cmd := &cobra.Command{
		Use:          "list",
		Short:        "list Crowdin glossaries",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, _, err := crowdinstorage.LoadClientConfig(o.configPath, o.identityPath)
			if err != nil {
				return err
			}
			client, err := newCrowdinGlossaryLister(cfg)
			if err != nil {
				return err
			}
			rows, err := client.ListGlossaries(cmd.Context(), cfg.ProjectID)
			if err != nil {
				return err
			}
			return writeEncodedOutput(cmd.OutOrStdout(), o.output, func() error {
				for _, row := range rows {
					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "id=%d name=%s\n", row.ID, row.Name); err != nil {
						return err
					}
				}
				return nil
			}, rows)
		},
	}
	addCrowdinListFlags(cmd, &o, false, false)
	return cmd
}

func newCrowdinTranslationMemoryListCmd() *cobra.Command {
	o := crowdinListOptions{output: "text"}
	cmd := &cobra.Command{
		Use:          "list",
		Short:        "list Crowdin translation memories",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, _, err := crowdinstorage.LoadClientConfig(o.configPath, o.identityPath)
			if err != nil {
				return err
			}
			client, err := newCrowdinTranslationMemoryLister(cfg)
			if err != nil {
				return err
			}
			rows, err := client.ListTranslationMemories(cmd.Context(), cfg.ProjectID)
			if err != nil {
				return err
			}
			return writeEncodedOutput(cmd.OutOrStdout(), o.output, func() error {
				for _, row := range rows {
					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "id=%d name=%s\n", row.ID, row.Name); err != nil {
						return err
					}
				}
				return nil
			}, rows)
		},
	}
	addCrowdinListFlags(cmd, &o, false, false)
	return cmd
}

func addCrowdinListFlags(cmd *cobra.Command, o *crowdinListOptions, includeLanguages, includeBranch bool) {
	cmd.Flags().StringVar(&o.configPath, "config", "", "path to crowdin.yml")
	cmd.Flags().StringVar(&o.identityPath, "identity", "", "path to Crowdin identity file")
	cmd.Flags().StringVar(&o.output, "output", o.output, "output format: text or json")
	if includeBranch {
		cmd.Flags().StringVar(&o.branch, "branch", "", "Crowdin branch name")
	}
	if includeLanguages {
		cmd.Flags().StringSliceVarP(&o.languages, "language", "l", nil, "target language(s) to include")
	}
}

func writeEncodedOutput(w io.Writer, output string, writeText func() error, value any) error {
	switch strings.ToLower(strings.TrimSpace(output)) {
	case "", "text":
		return writeText()
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(value)
	default:
		return fmt.Errorf("unsupported output format %q", output)
	}
}
