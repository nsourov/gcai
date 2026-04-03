package git

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type DiffMode string

const (
	DiffStaged   DiffMode = "staged"
	DiffUnstaged DiffMode = "unstaged"
	DiffAll      DiffMode = "all"
)

func EnsureGitRepo() error {
	out, err := runGit("rev-parse", "--is-inside-work-tree")
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) != "true" {
		return errors.New("not inside a git repository")
	}
	return nil
}

func CurrentBranch() string {
	out, err := runGit("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "unknown"
	}
	branch := strings.TrimSpace(out)
	if branch == "" {
		return "unknown"
	}
	return branch
}

func DiffStat(mode DiffMode) (string, error) {
	switch mode {
	case DiffStaged:
		return runGit("diff", "--staged", "--stat")
	case DiffUnstaged:
		return runGit("diff", "--stat")
	case DiffAll:
		staged, err := runGit("diff", "--staged", "--stat")
		if err != nil {
			return "", err
		}
		unstaged, err := runGit("diff", "--stat")
		if err != nil {
			return "", err
		}
		return combine(staged, unstaged), nil
	default:
		return "", fmt.Errorf("unsupported diff mode: %s", mode)
	}
}

func Diff(mode DiffMode) (string, error) {
	switch mode {
	case DiffStaged:
		return runGit("diff", "--staged")
	case DiffUnstaged:
		return runGit("diff")
	case DiffAll:
		staged, err := runGit("diff", "--staged")
		if err != nil {
			return "", err
		}
		unstaged, err := runGit("diff")
		if err != nil {
			return "", err
		}
		return combine(staged, unstaged), nil
	default:
		return "", fmt.Errorf("unsupported diff mode: %s", mode)
	}
}

// AddAll runs `git add -A` from the current working directory (repository root or subdir).
func AddAll() error {
	_, err := runGit("add", "-A")
	return err
}

// Commit runs `git commit -m <message>`.
func Commit(message string) error {
	if strings.TrimSpace(message) == "" {
		return errors.New("commit message is empty")
	}
	_, err := runGit("commit", "-m", message)
	return err
}

func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s failed: %s", strings.Join(args, " "), msg)
	}
	return out.String(), nil
}

func combine(a, b string) string {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	switch {
	case a == "" && b == "":
		return ""
	case a == "":
		return b + "\n"
	case b == "":
		return a + "\n"
	default:
		return a + "\n\n" + b + "\n"
	}
}
