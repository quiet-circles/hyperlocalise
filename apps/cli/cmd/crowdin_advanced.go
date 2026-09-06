package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	crowdinstorage "github.com/hyperlocalise/hyperlocalise/internal/i18n/storage/crowdin"
	"github.com/spf13/cobra"
)

type crowdinStringListOptions struct {
	configPath   string
	identityPath string
	branch       string
	filePath     string
	filter       string
	output       string
}

type crowdinFileUploadOptions struct {
	configPath   string
	identityPath string
	branch       string
	dest         string
}

type crowdinFileDownloadOptions struct {
	configPath   string
	identityPath string
	branch       string
	dest         string
	language     string
	force        bool
}

type crowdinFileDeleteOptions struct {
	configPath   string
	identityPath string
	branch       string
}

type crowdinAutoTranslateOptions struct {
	configPath                    string
	identityPath                  string
	languages                     []string
	filePath                      string
	branch                        string
	directoryID                   int
	method                        string
	autoApproveOption             string
	duplicateTranslations         bool
	skipApprovedTranslations      bool
	translateUntranslatedOnly     bool
	translateWithPerfectMatchOnly bool
}

func newCrowdinStringCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "string",
		Short: "Crowdin source string commands",
	}
	cmd.AddCommand(newCrowdinStringListCmd())
	return cmd
}

func newCrowdinStringListCmd() *cobra.Command {
	o := crowdinStringListOptions{output: "text"}
	cmd := &cobra.Command{
		Use:          "list",
		Short:        "list Crowdin source strings",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, _, err := crowdinstorage.LoadClientConfig(o.configPath, o.identityPath)
			if err != nil {
				return err
			}
			cfg = applyCrowdinBranchFlag(cfg, o.branch)
			client, err := newCrowdinSourceStringLister(cfg)
			if err != nil {
				return err
			}
			rows, err := client.ListProjectSourceStrings(cmd.Context(), crowdinstorage.ListSourceStringsInput{
				ProjectID: cfg.ProjectID,
				Branch:    cfg.Branch,
				FilePath:  o.filePath,
				Filter:    o.filter,
			})
			if err != nil {
				return err
			}
			return writeEncodedOutput(cmd.OutOrStdout(), o.output, func() error {
				for _, row := range rows {
					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "id=%d identifier=%s text=%s context=%s\n", row.ID, row.Identifier, row.Text, row.Context); err != nil {
						return err
					}
				}
				return nil
			}, rows)
		},
	}
	cmd.Flags().StringVar(&o.configPath, "config", "", "path to crowdin.yml")
	cmd.Flags().StringVar(&o.identityPath, "identity", "", "path to Crowdin identity file")
	cmd.Flags().StringVar(&o.branch, "branch", "", "Crowdin branch name")
	cmd.Flags().StringVar(&o.filePath, "file", "", "Crowdin file path to filter strings")
	cmd.Flags().StringVar(&o.filter, "filter", "", "filter strings by identifier, text, or context")
	cmd.Flags().StringVar(&o.output, "output", o.output, "output format: text or json")
	return cmd
}

func newCrowdinFileUploadCmd() *cobra.Command {
	o := crowdinFileUploadOptions{}
	cmd := &cobra.Command{
		Use:          "upload <file>",
		Short:        "upload one local file to a Crowdin project path",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(o.dest) == "" {
				return fmt.Errorf("crowdin file upload: --dest is required")
			}
			cfg, _, err := crowdinstorage.LoadClientConfig(o.configPath, o.identityPath)
			if err != nil {
				return err
			}
			cfg = applyCrowdinBranchFlag(cfg, o.branch)
			client, err := newCrowdinFileOpsClient(cfg)
			if err != nil {
				return err
			}
			fileID, err := client.UploadProjectFile(cmd.Context(), cfg.ProjectID, cfg.Branch, o.dest, args[0])
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "uploaded dest=%s file_id=%d\n", o.dest, fileID)
			return err
		},
	}
	cmd.Flags().StringVar(&o.configPath, "config", "", "path to crowdin.yml")
	cmd.Flags().StringVar(&o.identityPath, "identity", "", "path to Crowdin identity file")
	cmd.Flags().StringVar(&o.branch, "branch", "", "Crowdin branch name")
	cmd.Flags().StringVar(&o.dest, "dest", "", "destination path in the Crowdin project")
	return cmd
}

func newCrowdinFileDownloadCmd() *cobra.Command {
	o := crowdinFileDownloadOptions{}
	cmd := &cobra.Command{
		Use:          "download <file>",
		Short:        "download one Crowdin file by project path",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := crowdinstorage.LoadClientConfig(o.configPath, o.identityPath)
			if err != nil {
				return err
			}
			cfg = applyCrowdinBranchFlag(cfg, o.branch)
			client, err := newCrowdinFileOpsClient(cfg)
			if err != nil {
				return err
			}
			language := strings.TrimSpace(o.language)
			if cmd.Flags().Changed("language") && language == "" {
				return fmt.Errorf("crowdin file download: language is required")
			}
			content, err := client.DownloadProjectFile(cmd.Context(), cfg.ProjectID, cfg.Branch, args[0], language)
			if err != nil {
				return err
			}
			dest := strings.TrimSpace(o.dest)
			if dest == "" || dest == "-" {
				_, err := cmd.OutOrStdout().Write(content)
				return err
			}
			if err := writeCrowdinFileDownload(dest, content, o.force); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "downloaded dest=%s bytes=%d\n", dest, len(content))
			return err
		},
	}
	cmd.Flags().StringVar(&o.configPath, "config", "", "path to crowdin.yml")
	cmd.Flags().StringVar(&o.identityPath, "identity", "", "path to Crowdin identity file")
	cmd.Flags().StringVar(&o.branch, "branch", "", "Crowdin branch name")
	cmd.Flags().StringVar(&o.dest, "dest", "", "local path to write; omit or use - for stdout")
	cmd.Flags().StringVarP(&o.language, "language", "l", "", "download a translation file for this language")
	cmd.Flags().BoolVar(&o.force, "force", false, "overwrite an existing destination file")
	return cmd
}

func newCrowdinFileDeleteCmd() *cobra.Command {
	o := crowdinFileDeleteOptions{}
	cmd := &cobra.Command{
		Use:          "delete <file>",
		Short:        "delete one Crowdin source file by project path",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := crowdinstorage.LoadClientConfig(o.configPath, o.identityPath)
			if err != nil {
				return err
			}
			cfg = applyCrowdinBranchFlag(cfg, o.branch)
			client, err := newCrowdinFileOpsClient(cfg)
			if err != nil {
				return err
			}
			if err := client.DeleteProjectFile(cmd.Context(), cfg.ProjectID, cfg.Branch, args[0]); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "deleted path=%s\n", args[0])
			return err
		},
	}
	cmd.Flags().StringVar(&o.configPath, "config", "", "path to crowdin.yml")
	cmd.Flags().StringVar(&o.identityPath, "identity", "", "path to Crowdin identity file")
	cmd.Flags().StringVar(&o.branch, "branch", "", "Crowdin branch name")
	return cmd
}

func newCrowdinAutoTranslateCmd() *cobra.Command {
	o := crowdinAutoTranslateOptions{
		method:                    "tm",
		translateUntranslatedOnly: true,
	}
	cmd := &cobra.Command{
		Use:          "auto-translate",
		Aliases:      []string{"pre-translate"},
		Short:        "pre-translate Crowdin files via translation memory",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			method := strings.ToLower(strings.TrimSpace(o.method))
			if method != "" && method != "tm" {
				return fmt.Errorf("crowdin auto-translate: method %q is not supported in v1 (tm only)", o.method)
			}
			cfg, _, err := crowdinstorage.LoadClientConfig(o.configPath, o.identityPath)
			if err != nil {
				return err
			}
			cfg = applyCrowdinBranchFlag(cfg, o.branch)
			client, err := newCrowdinAutoTranslator(cfg)
			if err != nil {
				return err
			}
			branch := cfg.Branch
			if o.directoryID > 0 && !cmd.Flags().Changed("branch") {
				branch = ""
			}
			in := crowdinstorage.PreTranslationInput{
				ProjectID:         cfg.ProjectID,
				Languages:         o.languages,
				FilePath:          o.filePath,
				Branch:            branch,
				DirectoryID:       o.directoryID,
				AutoApproveOption: o.autoApproveOption,
			}
			if cmd.Flags().Changed("duplicate-translations") {
				value := o.duplicateTranslations
				in.DuplicateTranslations = &value
			}
			if cmd.Flags().Changed("skip-approved-translations") {
				value := o.skipApprovedTranslations
				in.SkipApprovedTranslations = &value
			}
			if cmd.Flags().Changed("translate-untranslated-only") {
				value := o.translateUntranslatedOnly
				in.TranslateUntranslatedOnly = &value
			}
			if cmd.Flags().Changed("translate-with-perfect-match-only") {
				value := o.translateWithPerfectMatchOnly
				in.TranslateWithPerfectMatchOnly = &value
			}
			result, err := client.ApplyPreTranslationAndWait(cmd.Context(), in)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "pre-translated status=%s progress=%d identifier=%s\n", result.Status, result.Progress, result.Identifier)
			return err
		},
	}
	cmd.Flags().StringVar(&o.configPath, "config", "", "path to crowdin.yml")
	cmd.Flags().StringVar(&o.identityPath, "identity", "", "path to Crowdin identity file")
	cmd.Flags().StringSliceVarP(&o.languages, "language", "l", nil, "target language(s) to pre-translate")
	cmd.Flags().StringVar(&o.filePath, "file", "", "Crowdin file path to pre-translate")
	cmd.Flags().StringVar(&o.branch, "branch", "", "Crowdin branch name to pre-translate")
	cmd.Flags().IntVar(&o.directoryID, "directory-id", 0, "Crowdin directory identifier to pre-translate")
	cmd.Flags().StringVar(&o.method, "method", "tm", "pre-translation method (v1 supports tm only)")
	cmd.Flags().StringVar(&o.autoApproveOption, "auto-approve-option", "", "TM auto-approve option")
	cmd.Flags().BoolVar(&o.duplicateTranslations, "duplicate-translations", false, "add translations even if the same translation already exists")
	cmd.Flags().BoolVar(&o.skipApprovedTranslations, "skip-approved-translations", false, "skip strings that already have an approved translation")
	cmd.Flags().BoolVar(&o.translateUntranslatedOnly, "translate-untranslated-only", true, "pre-translate untranslated strings only")
	cmd.Flags().BoolVar(&o.translateWithPerfectMatchOnly, "translate-with-perfect-match-only", false, "apply TM matches only when source and context are identical")
	_ = cmd.MarkFlagRequired("language")
	return cmd
}

func writeCrowdinFileDownload(path string, content []byte, force bool) error {
	if info, err := os.Stat(path); err == nil {
		if info.IsDir() {
			return fmt.Errorf("crowdin file download: output file %q is a directory", path)
		}
		if !force {
			return fmt.Errorf("crowdin file download: output file %q already exists; use --force to overwrite", path)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("crowdin file download: stat output file %q: %w", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("crowdin file download: mkdir output directory: %w", err)
	}
	return writeFileAtomic(path, content)
}
