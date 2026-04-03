# gcai

Suggest a one-line git commit message from your diff. Uses an **OpenAI-compatible** HTTP API (OpenAI, OpenRouter, and similar providers).

## Install

**Release (curl):**

```bash
curl -fsSL https://raw.githubusercontent.com/nsourov/gcai/main/scripts/install.sh | bash
```

**From source:**

```bash
git clone https://github.com/nsourov/gcai.git
cd gcai
go build -o bin/gcai ./cmd/gcai
```

## Configuration

Store settings in a JSON file under your user config directory — **no environment variables**. Set values with:

```bash
gcai config set api_key "your-api-key"
gcai config set base_url "https://api.openai.com/v1"
gcai config set model "gpt-4o-mini"
```

| Key           | Typical value                                                         |
| ------------- | --------------------------------------------------------------------- |
| `api_key`     | Provider secret                                                       |
| `base_url`    | e.g. `https://api.openai.com/v1` or `https://openrouter.ai/api/v1`    |
| `model`       | e.g. `gpt-4o-mini` or `openai/gpt-4o-mini` (OpenRouter)               |
| `auto_commit` | `true` or `false` — if `true`, **`gcai`** runs add → message → commit |

### Other useful `gcai config` commands

```bash
gcai config path              # print config file path
gcai config show              # JSON (API key masked by default)
gcai config show --plain      # include real API key — only if safe
gcai config --auto-commit     # store auto_commit=true (used only when you run gcai)
gcai config --no-auto-commit  # store auto_commit=false
```

With **`auto_commit`** enabled, **`gcai`** runs `git add -A`, generates a subject from the **staged** diff, then `git commit -m "..."`. Otherwise **`gcai`** only prints a suggested subject (default diff: staged).

## Usage

Run inside a git repository. **Default:** staged diff only (`git diff --staged`), unless **`auto_commit`** is on (then always add-all → staged message → commit).

```bash
gcai              # subject only, or add+commit when auto_commit is true
gcai --unstaged   # working tree (ignored if auto_commit is true)
gcai --all        # staged + unstaged (ignored if auto_commit is true)
gcai --update     # Update to the latest GitHub release
gcai version        # CLI build version (same as gcai --version)
gcai --help       # lists flags
```

Example:

```bash
git add .
git commit -m "$(gcai)"
```

## Notes

- Diffs may contain secrets; review before calling a remote API.
- Large diffs are truncated before the request.
