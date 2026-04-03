package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/nsourov/gcai/internal/config"
)

func configComplete(cfg config.Config) bool {
	return strings.TrimSpace(cfg.APIKey) != "" &&
		strings.TrimSpace(cfg.BaseURL) != "" &&
		strings.TrimSpace(cfg.Model) != ""
}

func newConfigCmd() *cobra.Command {
	var autoCommit bool
	var noAutoCommit bool

	cmd := &cobra.Command{
		Use:           "config",
		Short:         "Config file helpers (set, show, path); --auto-commit / --no-auto-commit store preferences for gcai",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			autoF := cmd.Flags().Lookup("auto-commit")
			noF := cmd.Flags().Lookup("no-auto-commit")
			if autoF.Changed || noF.Changed {
				want := autoCommit && (!noF.Changed || !noAutoCommit)
				cfg, err := config.Load()
				if err != nil {
					return err
				}
				cfg.AutoCommit = want
				if err := config.Save(cfg); err != nil {
					return err
				}
				p, _ := config.Path()
				fmt.Fprintf(os.Stdout, "Saved auto_commit=%v in %s (takes effect on `gcai`).\n", want, p)
				return nil
			}
			return runConfigUpsert()
		},
	}

	cmd.Flags().BoolVar(&autoCommit, "auto-commit", false, "Store auto_commit=true: gcai will git add -A, generate a subject, and git commit")
	cmd.Flags().BoolVar(&noAutoCommit, "no-auto-commit", true, "With --auto-commit, blocks saving auto_commit unless false; alone saves auto_commit=false")

	cmd.AddCommand(newConfigSetCmd())
	cmd.AddCommand(newConfigPathCmd())
	cmd.AddCommand(newConfigShowCmd())

	return cmd
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "set <key> <value>",
		Short:         "Set one config field (merges with existing file)",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := strings.TrimSpace(args[0])
			value := strings.TrimSpace(strings.Join(args[1:], " "))
			if value == "" {
				return errors.New("value cannot be empty")
			}
			field, err := normalizeConfigKey(key)
			if err != nil {
				return err
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			out := config.Config{
				APIKey:     cfg.APIKey,
				BaseURL:    fallback(cfg.BaseURL, defaultBaseURL),
				Model:      fallback(cfg.Model, defaultModel),
				AutoCommit: cfg.AutoCommit,
			}
			switch field {
			case "api_key":
				out.APIKey = value
			case "base_url":
				out.BaseURL = value
			case "model":
				out.Model = value
			case "auto_commit":
				v, err := parseBoolConfig(value)
				if err != nil {
					return err
				}
				out.AutoCommit = v
			}
			if err := config.Save(out); err != nil {
				return err
			}
			p, _ := config.Path()
			fmt.Printf("Updated %s in %s\n", field, p)
			return nil
		},
	}
}

func newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "path",
		Short:         "Print the config file path",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := config.Path()
			if err != nil {
				return err
			}
			fmt.Println(p)
			return nil
		},
	}
}

func newConfigShowCmd() *cobra.Command {
	var redacted bool
	var plain bool
	c := &cobra.Command{
		Use:           "show",
		Short:         "Print current config as JSON",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			mask := redacted && !plain
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			out := cfg
			if mask && strings.TrimSpace(out.APIKey) != "" {
				out.APIKey = redactAPIKey(out.APIKey)
			}
			b, err := json.MarshalIndent(out, "", "  ")
			if err != nil {
				return err
			}
			_, err = os.Stdout.Write(append(b, '\n'))
			return err
		},
	}
	c.Flags().BoolVar(&redacted, "redacted", true, "Mask API key")
	c.Flags().BoolVar(&plain, "plain", false, "Print API key in full (trusted environment only; overrides --redacted)")
	return c
}

func normalizeConfigKey(key string) (string, error) {
	k := strings.ToLower(strings.TrimSpace(key))
	k = strings.ReplaceAll(k, "-", "_")
	switch k {
	case "api_key", "apikey", "key":
		return "api_key", nil
	case "base_url", "baseurl", "url":
		return "base_url", nil
	case "model":
		return "model", nil
	case "auto_commit", "autocommit":
		return "auto_commit", nil
	default:
		return "", fmt.Errorf("unknown key %q (use api_key, base_url, model, or auto_commit)", key)
	}
}

func parseBoolConfig(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "t", "yes", "y", "on":
		return true, nil
	case "0", "false", "f", "no", "n", "off":
		return false, nil
	default:
		return false, fmt.Errorf("expected true or false for auto_commit, got %q", s)
	}
}

func redactAPIKey(key string) string {
	key = strings.TrimSpace(key)
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "…" + key[len(key)-4:]
}

func runConfigUpsert() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if configComplete(cfg) {
		return errors.New("config is already complete; change values with: gcai config set api_key|base_url|model ...")
	}

	out := config.Config{
		APIKey:     cfg.APIKey,
		BaseURL:    fallback(cfg.BaseURL, defaultBaseURL),
		Model:      fallback(cfg.Model, defaultModel),
		AutoCommit: cfg.AutoCommit,
	}

	var updateAPIKey, updateBaseURL, updateModel bool
	if !cfg.Exists() {
		updateAPIKey, updateBaseURL, updateModel = true, true, true
	} else {
		updateAPIKey = strings.TrimSpace(cfg.APIKey) == ""
		updateBaseURL = strings.TrimSpace(cfg.BaseURL) == ""
		updateModel = strings.TrimSpace(cfg.Model) == ""
	}

	if updateAPIKey {
		var key string
		if err := survey.AskOne(&survey.Password{
			Message: "API key (OpenAI/OpenRouter-compatible):",
		}, &key, survey.WithValidator(survey.Required)); err != nil {
			return fmt.Errorf("config cancelled: %w", err)
		}
		out.APIKey = strings.TrimSpace(key)
	}
	if updateBaseURL {
		var u string
		if err := survey.AskOne(&survey.Input{
			Message: "Base URL:",
			Default: out.BaseURL,
		}, &u, survey.WithValidator(survey.Required)); err != nil {
			return fmt.Errorf("config cancelled: %w", err)
		}
		out.BaseURL = strings.TrimSpace(u)
	}
	if updateModel {
		var m string
		if err := survey.AskOne(&survey.Input{
			Message: "Model:",
			Default: out.Model,
		}, &m, survey.WithValidator(survey.Required)); err != nil {
			return fmt.Errorf("config cancelled: %w", err)
		}
		out.Model = strings.TrimSpace(m)
	}

	if err := config.Save(out); err != nil {
		return err
	}

	path, _ := config.Path()
	fmt.Printf("Saved config to %s\n", path)
	fmt.Println("Run `gcai --help` to see usage.")
	return nil
}
