package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nsourov/gcai/internal/config"
	commitgit "github.com/nsourov/gcai/internal/git"
	"github.com/nsourov/gcai/internal/llm"
	"github.com/nsourov/gcai/internal/prompt"
	"github.com/nsourov/gcai/internal/update"
)

const (
	defaultBaseURL = "https://api.openai.com/v1"
	defaultModel   = "gpt-4o-mini"
)

// appVersion is the gcai CLI build version (not the LLM model); set from main via SetVersion.
var appVersion = "dev"

// SetVersion records the CLI build version for gcai version / gcai --version.
func SetVersion(v string) {
	if strings.TrimSpace(v) != "" {
		appVersion = strings.TrimSpace(v)
	}
}

// Version returns the gcai CLI build version (e.g. v0.1.0, or dev).
func Version() string {
	return appVersion
}

type options struct {
	staged   bool
	unstaged bool
	all      bool
	update   bool
}

func NewRootCmd() *cobra.Command {
	opts := options{}

	cmd := &cobra.Command{
		Use:           "gcai",
		Short:         "Generate a short commit message from git diff",
		Version:       appVersion,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.update {
				tag, err := update.Run("")
				if err != nil {
					return err
				}
				fmt.Printf("Installed gcai %s\n", tag)
				return nil
			}
			return run(cmd.Context(), opts)
		},
	}

	cmd.Flags().BoolVar(&opts.staged, "staged", false, "Use staged diff")
	cmd.Flags().BoolVar(&opts.unstaged, "unstaged", false, "Use unstaged diff")
	cmd.Flags().BoolVar(&opts.all, "all", false, "Use both staged and unstaged diff")
	cmd.Flags().BoolVar(&opts.update, "update", false, "Download latest release from GitHub and replace this binary")

	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newVersionCmd())

	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "version",
		Short:         "Print the gcai CLI version (build)",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), appVersion)
		},
	}
}

func run(ctx context.Context, opts options) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.AutoCommit {
		if err := commitgit.EnsureGitRepo(); err != nil {
			return err
		}
		if err := commitgit.AddAll(); err != nil {
			return err
		}
		subject, err := generateCommitSubject(ctx, commitgit.DiffStaged)
		if err != nil {
			return err
		}
		if err := commitgit.Commit(subject); err != nil {
			return err
		}
		fmt.Printf("Committed: %s\n", subject)
		return nil
	}

	mode, err := resolveMode(opts)
	if err != nil {
		return err
	}
	subject, err := generateCommitSubject(ctx, mode)
	if err != nil {
		return err
	}
	fmt.Println(subject)
	return nil
}

// generateCommitSubject builds the LLM prompt from git state and returns a one-line subject.
func generateCommitSubject(ctx context.Context, mode commitgit.DiffMode) (string, error) {
	if err := commitgit.EnsureGitRepo(); err != nil {
		return "", err
	}

	diff, err := commitgit.Diff(mode)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(diff) == "" {
		return "", errors.New("no changes found for selected diff mode")
	}

	diffStat, err := commitgit.DiffStat(mode)
	if err != nil {
		return "", err
	}

	msgs := prompt.BuildMessages(prompt.Input{
		Branch:   commitgit.CurrentBranch(),
		DiffStat: strings.TrimSpace(diffStat),
		Diff:     diff,
	})

	settings, err := resolveSettings()
	if err != nil {
		return "", err
	}

	client := llm.Client{
		BaseURL: settings.BaseURL,
		APIKey:  settings.APIKey,
		Model:   settings.Model,
	}
	return client.GenerateCommitMessage(ctx, msgs)
}

func resolveMode(opts options) (commitgit.DiffMode, error) {
	// --all wins if set.
	if opts.all {
		return commitgit.DiffAll, nil
	}

	if opts.staged && opts.unstaged {
		return commitgit.DiffAll, nil
	}
	if opts.unstaged {
		return commitgit.DiffUnstaged, nil
	}
	if opts.staged {
		return commitgit.DiffStaged, nil
	}

	// Default mode when no mode flags are provided.
	return commitgit.DiffStaged, nil
}

type settings struct {
	APIKey  string
	BaseURL string
	Model   string
}

func resolveSettings() (settings, error) {
	cfg, err := config.Load()
	if err != nil {
		return settings{}, err
	}
	if !cfg.Exists() {
		return settings{}, errors.New("config is missing; run `gcai config set api_key ...` (and base_url, model)")
	}
	if strings.TrimSpace(cfg.APIKey) == "" || strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.Model) == "" {
		return settings{}, errors.New("config is incomplete; run `gcai config set` for missing keys")
	}

	return settings{
		APIKey:  strings.TrimSpace(cfg.APIKey),
		BaseURL: strings.TrimSpace(cfg.BaseURL),
		Model:   strings.TrimSpace(cfg.Model),
	}, nil
}

func fallback(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return strings.TrimSpace(v)
}
