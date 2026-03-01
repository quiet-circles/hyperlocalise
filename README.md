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

# Commands

```
hyperlocalise [flags]
hyperlocalise [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  init        write the latest i18n.jsonc template
  run         generate local translations from source files
  status      show translation status by locale
  sync        synchronize translations with remote storage adapters
  version     hyperlocalise version
```

## run

Generate local translations from configured source files into target files:

```
hyperlocalise run [--config <path>] [--dry-run]
```

Behavior:
- Loads and validates `i18n.jsonc`
- Plans entry-level translation tasks from group/bucket mappings
- Skips tasks already recorded in `.hyperlocalise.lock.json`
- Executes remaining tasks in parallel (worker count = CPU core count)
- Persists each successful task completion to lock state

Flags:
- `--config` - path to i18n config (optional, defaults to i18n.jsonc in cwd)
- `--dry-run` - print plan without translating or writing files

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

# LLM Providers

`hyperlocalise` supports these translation model providers in `llm.profiles.*.provider`:
- `openai`
- `lmstudio`
- `groq`
- `ollama`

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
