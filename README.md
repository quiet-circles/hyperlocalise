# hyperlocalise

A high-performance localization CLI built in Go for modern development workflows — lightweight, fast, and dependency-minimal.

[![Go Report Card](https://goreportcard.com/badge/github.com/quiet-circles/hyperlocalise)](https://goreportcard.com/report/github.com/quiet-circles/hyperlocalise)

# Table of Contents
<!--ts-->
   * [hyperlocalise](#hyperlocalise)
   * [Features](#features)
   * [Commands](#commands)
   * [LLM Providers](#llm-providers)
   * [Storage Adapters](#storage-adapters)
   * [Project Layout](#project-layout)
   * [Makefile Targets](#makefile-targets)
   * [Release](#release)
   * [Contribute](#contribute)

<!--te-->

# Features
- [goreleaser](https://goreleaser.com/) releases publishing multi-arch (`amd64`/`arm64`) binaries for macOS and Linux, plus `.deb` and `.rpm` packages.
- [golangci-lint](https://golangci-lint.run/) for linting and formatting
- [cobra](https://cobra.dev/) setup including tests
- [Makefile](Makefile) - with various useful targets and documentation (see Makefile Targets)
- Storage adapter based translation sync with POEditor, Lokalise, Crowdin, and Smartling support
- Local provenance sidecar metadata for LLM-vs-curation workflows (`draft` vs `curated`)

## Install

Use the stable bootstrap URL:

```bash
curl -fsSL https://raw.githubusercontent.com/quiet-circles/hyperlocalise/main/install.sh | bash
```

Pin a specific release:

```bash
curl -fsSL https://raw.githubusercontent.com/quiet-circles/hyperlocalise/main/install.sh | VERSION=v1.2.3 bash
```

## Install the Hyperlocalise Agent Skill (skills.sh)

Install from this repository (recommended when you already cloned the repo):

```bash
npx skills add . --skill hyperlocalise
```

Install directly from GitHub:

```bash
npx skills add https://github.com/quiet-circles/hyperlocalise --skill hyperlocalise
```

This uses the [skills.sh](https://skills.sh) installer via `npx`.

# Commands

```
hyperlocalise [flags]
hyperlocalise [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  eval        evaluate translation quality across experiment variants
  init        write the latest i18n.jsonc template
  run         generate local translations from source files
  status      show translation status by locale
  sync        synchronize translations with remote storage adapters
  update      update hyperlocalise using the bootstrap installer
  version     hyperlocalise version
```

## run

Generate local translations from configured source files into target files:

```
hyperlocalise run [--config <path>] [--group <name>] [--bucket <name>] [--dry-run] [--force] [--prune] [--prune-max-deletions <n>] [--prune-force] [--workers <count>] [--output <report.json>]
```

Behavior:
- Loads and validates `i18n.jsonc`
- Plans entry-level translation tasks from group/bucket mappings
- Skips tasks already recorded in `.hyperlocalise.lock.json`
- Rehydrates staged values from lock checkpoints for interrupted runs
- Executes remaining tasks in parallel (worker count = CPU core count) with transient-error retries
- Persists each successful task completion and checkpoint to lock state

JSON format support in `run`:
- Standard nested JSON key/value objects are supported.
- FormatJS message JSON is also supported when the root strictly matches:
  `{"[id]": {"defaultMessage": "[message]", "description": "[description]"}}`
- In FormatJS mode, only `defaultMessage` is translated. Message IDs, `description`, and other metadata are preserved.

Flutter ARB format support in `run`:
- `.arb` files are supported for source and target mappings.
- Only message keys are translated; metadata keys such as `@key` and `@@locale` are preserved.

Prune workflow recommendation:
- Run `hyperlocalise run --dry-run --prune` regularly (for example weekly or before releases) to review stale-key candidates.
- Apply approved cleanup with `hyperlocalise run --prune` in a dedicated change so key deletions are easy to audit.
- Keep the safety limit enabled and only use `--prune-force` for intentional large restructures.

Flags:
- `--config` - path to i18n config (optional, defaults to i18n.jsonc in cwd)
- `--group` - run only tasks for one configured group
- `--bucket` - run only tasks for one configured bucket
- `--dry-run` - print plan without translating or writing files
- `--force` - rerun all planned tasks and ignore lockfile skip state
- `--prune` - preview/apply deletion of stale target keys missing from source
- `--prune-max-deletions` - safety guard for max deletions per run before requiring override (default: 100)
- `--prune-force` - bypass the prune safety limit
- `--workers` - number of parallel translation workers (defaults to CPU core count)
- `--progress` - progress rendering mode (`auto|on|off`, default: `auto`)
- `--output` - write machine-readable JSON run report to the given path

Run report output:
- stdout summary now includes token totals: `prompt_tokens`, `completion_tokens`, and `total_tokens`
- stdout includes per-locale token lines: `locale_usage locale=<locale> ...`
- `--output` writes a JSON report with run metadata (`generatedAt`, `configPath`), aggregate token usage, per-locale usage, and per-entry batch usage

Progress debug logging (optional):
- `HYPERLOCALISE_PROGRESS_DEBUG=1` enables progress debug logging
- `HYPERLOCALISE_PROGRESS_DEBUG_FILE=<path>` overrides the debug log location
- default path when enabled: `.hyperlocalise/logs/run.log`


## eval

Run experiment-based translation quality checks and compare reports:

```
hyperlocalise eval run --eval-set <path> [--profile <name> ...] [--provider <name> ...] [--model <name> ...] [--prompt <text> | --prompt-file <path>] [--output <report.json>]
hyperlocalise eval compare --candidate <report.json> --baseline <report.json> [--min-score <value>] [--max-regression <value>]
```

`eval run` prints a concise per-experiment table with score, pass rate, placeholder violations, and latency.

`eval compare` supports CI gating:
- `--min-score` fails when candidate weighted score is below threshold
- `--max-regression` fails when candidate regresses more than allowed versus baseline

## sync

The `sync` command synchronizes translations between local storage and remote storage adapters.

```
hyperlocalise sync [command]

Available Commands:
  pull  pull translations from remote storage
  push  push translations to remote storage
```

### sync pull

Pull translations from a remote storage adapter:

```
hyperlocalise sync pull [--config <path>] [flags]
```

Flags:
- `--config` - path to i18n config (optional, defaults to i18n.jsonc in cwd)
- `--locale` - target locale(s) to sync (can be repeated)
- `--dry-run` - preview changes without applying (default: true)
- `--output` - output format: text or json
- `--fail-on-conflict` - return error if conflicts are detected (default: true)
- `--apply-curated-over-draft` - allow pull to update local draft entries with curated remote values (default: true)

### sync push

Push translations to a remote storage adapter:

```
hyperlocalise sync push [--config <path>] [flags]
```

Flags:
- `--config` - path to i18n config (optional, defaults to i18n.jsonc in cwd)
- `--locale` - target locale(s) to sync (can be repeated)
- `--dry-run` - preview changes without applying (default: true)
- `--output` - output format: text or json
- `--fail-on-conflict` - return error if conflicts are detected (default: true)
- `--force-conflicts` - allow overwriting remote mismatches despite conflict policies (default: false)

## status

Show translation status by locale:

```
hyperlocalise status [--config <path>] [flags]
```

Output shows translation status for each locale:
- `translated` - has a non-empty translation value
- `needs_review` - LLM-generated translation not yet curated
- `untranslated` - empty translation value

Flags:
- `--config` - path to i18n config (optional, defaults to i18n.jsonc in cwd)
- `--locale` - target locale(s) to report (can be repeated)
- `--output` - output format: csv
- `--group` - filter by group name
- `--bucket` - filter by bucket name
- `-i`, `--interactive` - render interactive dashboard in TTY (requires `--output csv`)

# LLM Providers

`hyperlocalise` supports these translation model providers in `llm.profiles.*.provider`:
- `openai`
- `azure_openai`
- `anthropic`
- `lmstudio`
- `groq`
- `mistral`
- `ollama`
- `gemini`
- `bedrock`

`llm.profiles.default` is required, and each profile requires:
- `provider`
- `model`
- `prompt`

Prompt variables:
- `{{source}}`
- `{{target}}`
- `{{input}}`

## OpenAI Example

Config:
```json
{
  "llm": {
    "profiles": {
      "default": {
        "provider": "openai",
        "model": "gpt-5.2",
        "prompt": "Translate from {{source}} to {{target}}:\n\n{{input}}"
      }
    }
  }
}
```

Environment:
```bash
export OPENAI_API_KEY="your-openai-api-key"
```

## Azure OpenAI Example

Config:
```json
{
  "llm": {
    "profiles": {
      "default": {
        "provider": "azure_openai",
        "model": "gpt-4.1-mini",
        "prompt": "Translate from {{source}} to {{target}}:\n\n{{input}}"
      }
    }
  }
}
```

Environment:
```bash
# Example: https://<resource>.openai.azure.com/openai/v1
export AZURE_OPENAI_BASE_URL="https://<resource>.openai.azure.com/openai/v1"
export AZURE_OPENAI_API_KEY="your-azure-openai-api-key"
```

## Gemini Example

Config:
```json
{
  "llm": {
    "profiles": {
      "default": {
        "provider": "gemini",
        "model": "gemini-2.5-flash",
        "prompt": "Translate from {{source}} to {{target}}:\n\n{{input}}"
      }
    }
  }
}
```

Environment:
```bash
# Optional, defaults to https://generativelanguage.googleapis.com/v1beta/openai
export GEMINI_BASE_URL="https://generativelanguage.googleapis.com/v1beta/openai"

export GEMINI_API_KEY="your-gemini-api-key"
```

## Anthropic Example

Config:
```json
{
  "llm": {
    "profiles": {
      "default": {
        "provider": "anthropic",
        "model": "claude-sonnet-4-5",
        "prompt": "Translate from {{source}} to {{target}}:\n\n{{input}}"
      }
    }
  }
}
```

Environment:
```bash
# Optional, defaults to https://api.anthropic.com/v1
export ANTHROPIC_BASE_URL="https://api.anthropic.com/v1"

export ANTHROPIC_API_KEY="your-anthropic-api-key"
```

## AWS Bedrock Example

Config:
```json
{
  "llm": {
    "profiles": {
      "default": {
        "provider": "bedrock",
        "model": "anthropic.claude-3-5-sonnet-20241022-v2:0",
        "prompt": "Translate from {{source}} to {{target}}:\n\n{{input}}"
      }
    }
  }
}
```

Environment:
```bash
export AWS_REGION="us-east-1"
export AWS_ACCESS_KEY_ID="your-access-key-id"
export AWS_SECRET_ACCESS_KEY="your-secret-access-key"
# Optional when using temporary credentials:
export AWS_SESSION_TOKEN="your-session-token"
```

## LM Studio Example (Local Model)

Config:
```json
{
  "llm": {
    "profiles": {
      "default": {
        "provider": "lmstudio",
        "model": "qwen2.5-7b-instruct",
        "prompt": "Translate from {{source}} to {{target}}:\n\n{{input}}"
      }
    }
  }
}
```

Environment:
```bash
# Optional, defaults to http://127.0.0.1:1234/v1
export LM_STUDIO_BASE_URL="http://127.0.0.1:1234/v1"

# Optional, defaults to lm-studio
export LM_STUDIO_API_KEY="lm-studio"
```

Notes:
- LM Studio must be running locally and serving its OpenAI-compatible API.
- `model` must match an identifier exposed by your local LM Studio server.

## Ollama Example (Local Model)

Config:
```json
{
  "llm": {
    "profiles": {
      "default": {
        "provider": "ollama",
        "model": "qwen2.5:7b",
        "prompt": "Translate from {{source}} to {{target}}:\n\n{{input}}"
      }
    }
  }
}
```

Environment:
```bash
# Optional, defaults to http://127.0.0.1:11434/v1
export OLLAMA_BASE_URL="http://127.0.0.1:11434/v1"

# Optional, defaults to ollama
export OLLAMA_API_KEY="ollama"
```

Notes:
- Ollama must be running locally with OpenAI-compatible API access enabled.
- `model` must match an identifier available in your local Ollama instance.

## Groq Example

Config:
```json
{
  "llm": {
    "profiles": {
      "default": {
        "provider": "groq",
        "model": "llama-3.3-70b-versatile",
        "prompt": "Translate from {{source}} to {{target}}:\n\n{{input}}"
      }
    }
  }
}
```

Environment:
```bash
# Optional, defaults to https://api.groq.com/openai/v1
export GROQ_BASE_URL="https://api.groq.com/openai/v1"

export GROQ_API_KEY="your-groq-api-key"
```

## Mistral Example

Config:
```json
{
  "llm": {
    "profiles": {
      "default": {
        "provider": "mistral",
        "model": "mistral-large-latest",
        "prompt": "Translate from {{source}} to {{target}}:\n\n{{input}}"
      }
    }
  }
}
```

Environment:
```bash
# Optional, defaults to https://api.mistral.ai/v1
export MISTRAL_BASE_URL="https://api.mistral.ai/v1"

export MISTRAL_API_KEY="your-mistral-api-key"
```


# Storage Adapters

`hyperlocalise` supports multiple translation management system (TMS) adapters through a pluggable storage adapter interface.

Launch readiness details, known limitations, and integration test matrix:
- [`docs/launch-readiness.md`](docs/launch-readiness.md)

## Supported Adapters

### POEditor

Docs: [`internal/i18n/storage/poeditor/README.md`](internal/i18n/storage/poeditor/README.md)

Configuration:
```json
{
  "adapter": "poeditor",
  "config": {
    "projectID": "your-project-id",
    "apiTokenEnv": "POEDITOR_API_TOKEN",
    "sourceLanguage": "en",
    "targetLanguages": ["fr", "de", "es"]
  }
}
```

Environment variable: `POEDITOR_API_TOKEN`

### Lokalise

Docs: [`internal/i18n/storage/lokalise/README.md`](internal/i18n/storage/lokalise/README.md)

Configuration:
```json
{
  "adapter": "lokalise",
  "config": {
    "projectID": "your-project-id",
    "apiTokenEnv": "LOKALISE_API_TOKEN",
    "sourceLanguage": "en",
    "targetLanguages": ["fr", "de", "es"]
  }
}
```

Environment variable: `LOKALISE_API_TOKEN`

### Crowdin

Configuration:
```json
{
  "adapter": "crowdin",
  "config": {
    "projectID": "123456",
    "apiTokenEnv": "CROWDIN_API_TOKEN",
    "sourceLanguage": "en",
    "targetLanguages": ["fr", "de", "es"]
  }
}
```

Environment variable: `CROWDIN_API_TOKEN`

### Phrase

Docs: [`internal/i18n/storage/phrase/README.md`](internal/i18n/storage/phrase/README.md)

Configuration:
```json
{
  "adapter": "phrase",
  "config": {
    "projectID": "your-project-id",
    "apiTokenEnv": "PHRASE_API_TOKEN",
    "mode": "strings",
    "sourceLanguage": "en",
    "targetLanguages": ["fr", "de", "es"]
  }
}
```

Environment variable: `PHRASE_API_TOKEN`

### Smartling

Docs: [`internal/i18n/storage/smartling/README.md`](internal/i18n/storage/smartling/README.md)

Configuration:
```json
{
  "adapter": "smartling",
  "config": {
    "projectID": "your-project-id",
    "userIdentifier": "your-user-identifier",
    "userSecretEnv": "SMARTLING_USER_SECRET",
    "targetLanguages": ["fr", "de", "es"],
    "timeoutSeconds": 30
  }
}
```

Environment variable: `SMARTLING_USER_SECRET`

For more details on the storage system and sync model, see [`internal/i18n/storage/README.md`](internal/i18n/storage/README.md).

# Project Layout
* [assets/](https://pkg.go.dev/github.com/quiet-circles/hyperlocalise/assets) => docs, images, etc
* [cmd/](https://pkg.go.dev/github.com/quiet-circles/hyperlocalise/cmd)  => commandline configurations (flags, subcommands)
* [internal/](https://pkg.go.dev/github.com/quiet-circles/hyperlocalise/internal)  => packages that are only for project internal purposes
* [`go.mod`](go.mod) `tool` directives => tracks CLI tooling versions (for example `golangci-lint`, `gofumpt`, `gci`, `goimports`, `staticcheck`) in a Go 1.24+ compatible way
- [`scripts/`](scripts/) => build scripts 

# Makefile Targets
```sh
$> make
bootstrap                      download tool and module dependencies
check-build                    check golang build
clean                          clean up environment
cover                          display test coverage
fmt                            format go files
help                           list makefile targets
install                        install golang binary
lint                           lint go files
precommit                      run local CI validation flow
run                            run the app
staticcheck                    run staticcheck directly
test                           run tests with JSON output and coverage
```

# Release
- Create and push a semantic version tag to trigger release CI:
  ```sh
  git tag v0.1.0
  git push origin v0.1.0
  ```
- Workflow: [`.github/workflows/release.yml`](.github/workflows/release.yml)
- GoReleaser config: [`.goreleaser.yml`](.goreleaser.yml)
- Published artifacts include:
  - `darwin/amd64`, `darwin/arm64` archives
  - `linux/amd64`, `linux/arm64` archives
  - Linux `.deb` and `.rpm` packages
- No extra repository secrets are required beyond the default `GITHUB_TOKEN`.

# Contribute
If you find issues in that setup or have some nice features / improvements, I would welcome an issue or a PR :)
