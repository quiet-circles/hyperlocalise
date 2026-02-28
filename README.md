# hyperlocalise

A high-performance localization CLI built in Go for modern development workflows — lightweight, fast, and dependency-minimal.

[![Go Report Card](https://goreportcard.com/badge/github.com/quiet-circles/hyperlocalise)](https://goreportcard.com/report/github.com/quiet-circles/hyperlocalise)

# Table of Contents
<!--ts-->
   * [hyperlocalise](#hyperlocalise)
   * [Features](#features)
   * [Commands](#commands)
   * [Storage Adapters](#storage-adapters)
   * [Project Layout](#project-layout)
   * [Makefile Targets](#makefile-targets)
   * [Contribute](#contribute)

<!--te-->

# Features
- [goreleaser](https://goreleaser.com/) with `deb.` and `.rpm` packer releasing, including `manpages`, `shell completions`, and grouped changelog generation.
- [golangci-lint](https://golangci-lint.run/) for linting and formatting
- [cobra](https://cobra.dev/) setup including tests
- [Makefile](Makefile) - with various useful targets and documentation (see Makefile Targets)
- Storage adapter based translation sync with POEditor and Lokalise support
- Local provenance sidecar metadata for LLM-vs-curation workflows (`draft` vs `curated`)

# Commands

```
hyperlocalise [flags]
hyperlocalise [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  init        write the latest i18n.jsonc template
  status      show translation status by locale
  sync        synchronize translations with remote storage adapters
  version     hyperlocalise version
```

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

# Contribute
If you find issues in that setup or have some nice features / improvements, I would welcome an issue or a PR :)
