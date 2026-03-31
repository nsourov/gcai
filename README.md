# git-commit-ai (`gcai`)

`gcai` is a CLI that reads your git diff and generates a short commit message using AI.

## Install

### Option A: Curl installer (no Go required)

```bash
curl -fsSL https://raw.githubusercontent.com/nsourov/gcai/main/scripts/install.sh | bash
```

The installer downloads the latest release binary and installs `gcai` to `/usr/local/bin`.

### Option B: Build from source

```bash
gh repo clone nsourov/gcai
cd gcai
go build -o bin/gcai ./cmd/gcai
```

## First-time setup

Run the interactive initializer:

```bash
gcai --init
```

If your shell already has an alias/function named `gcai`, run the local binary directly:

```bash
./bin/gcai --init
```

It prompts for:

- API key (required, from your OpenAI-compatible provider such as OpenAI or OpenRouter)
- Base URL (default: `https://api.openai.com/v1`)
- Model (default: `gpt-4o-mini`)

## Usage

Run inside a git repo:

```bash
gcai
```

If `gcai` is shadowed by a shell alias/function, use:

```bash
./bin/gcai
```

Default mode uses staged changes (`git diff --staged`).

### Flags

- `--init`: run interactive setup and save config
- `--force`: overwrite existing config (only with `--init`)
- `--staged`: use staged changes
- `--unstaged`: use unstaged changes
- `--all`: use staged + unstaged changes
- `-h, --help`: show command help

Mode precedence:

- `--all` overrides `--staged` / `--unstaged`
- `--staged --unstaged` is treated as `--all`
- no mode flags means staged mode

## Configuration

`gcai` uses only saved config from `gcai --init`.

- If config is missing, it errors and asks you to run `gcai --init`.
- If any config field is missing, it errors and asks you to run `gcai --init --force`.
- `--init` will not overwrite existing config unless `--force` is provided.

## Examples

Use staged changes:

```bash
gcai
```

Use unstaged changes:

```bash
gcai --unstaged
```

Use all changes:

```bash
gcai --all
```

Use in commit flow:

```bash
git add .
git commit -m "$(gcai)"
```

Reconfigure:

```bash
gcai --init --force
```

## Requirements

- `git` installed and available in `PATH`
- OpenAI-compatible API credentials (for example OpenAI or OpenRouter)

## Notes

- Diffs can contain secrets; review before sending to remote LLM APIs.
- Very large diffs are truncated before API submission.
