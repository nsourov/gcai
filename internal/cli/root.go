package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/nsourov/gcai/internal/config"
	commitgit "github.com/nsourov/gcai/internal/git"
	"github.com/nsourov/gcai/internal/llm"
	"github.com/nsourov/gcai/internal/prompt"
)

const (
	defaultBaseURL = "https://api.openai.com/v1"
	defaultModel   = "gpt-4o-mini"
)

type options struct {
	staged   bool
	unstaged bool
	all      bool
	init     bool
	force    bool
}

func NewRootCmd() *cobra.Command {
	opts := options{}

	cmd := &cobra.Command{
		Use:           "gcai",
		Short:         "Generate a short commit message from git diff",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), opts)
		},
	}

	cmd.Flags().BoolVar(&opts.staged, "staged", false, "Use staged diff")
	cmd.Flags().BoolVar(&opts.unstaged, "unstaged", false, "Use unstaged diff")
	cmd.Flags().BoolVar(&opts.all, "all", false, "Use both staged and unstaged diff")
	cmd.Flags().BoolVar(&opts.init, "init", false, "Initialize gcai config interactively")
	cmd.Flags().BoolVar(&opts.force, "force", false, "Overwrite existing config (use with --init)")

	return cmd
}

func run(ctx context.Context, opts options) error {
	if opts.init {
		return runInit(opts.force)
	}

	mode, err := resolveMode(opts)
	if err != nil {
		return err
	}
	if err := commitgit.EnsureGitRepo(); err != nil {
		return err
	}

	diff, err := commitgit.Diff(mode)
	if err != nil {
		return err
	}
	if strings.TrimSpace(diff) == "" {
		return errors.New("no changes found for selected diff mode")
	}

	diffStat, err := commitgit.DiffStat(mode)
	if err != nil {
		return err
	}

	msgs := prompt.BuildMessages(prompt.Input{
		Branch:   commitgit.CurrentBranch(),
		DiffStat: strings.TrimSpace(diffStat),
		Diff:     diff,
	})

	settings, err := resolveSettings()
	if err != nil {
		return err
	}

	client := llm.Client{
		BaseURL: settings.BaseURL,
		APIKey:  settings.APIKey,
		Model:   settings.Model,
	}
	subject, err := client.GenerateCommitMessage(ctx, msgs)
	if err != nil {
		return err
	}

	fmt.Println(subject)
	return nil
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

func runInit(force bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.Exists() && !force {
		return errors.New("config already exists; re-run with `gcai --init --force` to overwrite")
	}

	answers := struct {
		APIKey  string
		BaseURL string
		Model   string
	}{}

	questions := []*survey.Question{
		{
			Name: "APIKey",
			Prompt: &survey.Password{
				Message: "API key (OpenAI/OpenRouter-compatible):",
			},
			Validate: survey.Required,
		},
		{
			Name: "BaseURL",
			Prompt: &survey.Input{
				Message: "Base URL:",
				Default: fallback(cfg.BaseURL, defaultBaseURL),
			},
			Validate: survey.Required,
		},
		{
			Name: "Model",
			Prompt: &survey.Input{
				Message: "Model:",
				Default: fallback(cfg.Model, defaultModel),
			},
			Validate: survey.Required,
		},
	}

	if err := survey.Ask(questions, &answers); err != nil {
		return fmt.Errorf("init cancelled: %w", err)
	}

	if err := config.Save(config.Config{
		APIKey:  strings.TrimSpace(answers.APIKey),
		BaseURL: strings.TrimSpace(answers.BaseURL),
		Model:   strings.TrimSpace(answers.Model),
	}); err != nil {
		return err
	}

	path, _ := config.Path()
	fmt.Printf("Saved config to %s\n", path)
	fmt.Println("Run `gcai --help` to see usage.")
	return nil
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
		return settings{}, errors.New("config is missing; run `gcai --init`")
	}
	if strings.TrimSpace(cfg.APIKey) == "" || strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.Model) == "" {
		return settings{}, errors.New("config is incomplete; run `gcai --init --force`")
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
