# hyperlocalise

A high-performance localization CLI built in Go for modern development workflows — lightweight, fast, and dependency-minimal.

[![Go Report Card](https://goreportcard.com/badge/github.com/quiet-circles/hyperlocalise)](https://goreportcard.com/report/github.com/quiet-circles/hyperlocalise)

# Table of Contents
<!--ts-->
   * [hyperlocalise](#hyperlocalise)
   * [Features](#features)
   * [Translation Sync (POEditor)](#translation-sync-poeditor)
   * [Project Layout](#project-layout)
   * [How to use this template](#how-to-use-this-template)
   * [Demo Application](#demo-application)
   * [Makefile Targets](#makefile-targets)
   * [Contribute](#contribute)

<!-- Added by: morelly_t1, at: Tue 10 Aug 2021 08:54:24 AM CEST -->

<!--te-->

# Features
- [goreleaser](https://goreleaser.com/) with `deb.` and `.rpm` packer releasing, including `manpages`, `shell completions`, and grouped changelog generation.
- [golangci-lint](https://golangci-lint.run/) for linting and formatting
- [Gitlab CI](.gitlab-ci.yml) Configuration (Lint, Test, Build, Release)
- [cobra](https://cobra.dev/) example setup including tests
- [Makefile](Makefile) - with various useful targets and documentation (see Makefile Targets)
- Storage adapter based translation sync (`sync pull` / `sync push`) with POEditor as the first adapter
- Local provenance sidecar metadata for LLM-vs-curation workflows (`draft` vs `curated`)
<!--- TODO: [pre-commit-hooks](https://pre-commit.com/) for formatting and validating code before committing-->

# Translation Sync (POEditor)

`hyperlocalise` now includes a `StorageAdapter` sync layer for remote translation storage.
The first supported adapter is `poeditor`.

Docs:

- [`internal/i18n/storage/README.md`](internal/i18n/storage/README.md)
- POEditor adapter details: [`internal/i18n/storage/poeditor/README.md`](internal/i18n/storage/poeditor/README.md)

# Project Layout
* [assets/](https://pkg.go.dev/github.com/quiet-circles/hyperlocalise/assets) => docs, images, etc
* [cmd/](https://pkg.go.dev/github.com/quiet-circles/hyperlocalise/cmd)  => commandline configurartions (flags, subcommands)
* [pkg/](https://pkg.go.dev/github.com/quiet-circles/hyperlocalise/pkg)  => packages that are okay to import for other projects
* [internal/](https://pkg.go.dev/github.com/quiet-circles/hyperlocalise/pkg)  => packages that are only for project internal purposes
* [`go.mod`](go.mod) `tool` directives => tracks CLI tooling versions (for example `golangci-lint`, `gofumpt`, `gci`, `goimports`, `staticcheck`) in a Go 1.24+ compatible way
- [`scripts/`](scripts/) => build scripts 

# Demo Application

```sh
$> hyperlocalise -h
hyperlocalise CLI demo application

Usage:
  hyperlocalise [flags]
  hyperlocalise [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  example     example subcommand which adds or multiplies two given integers
  help        Help about any command
  version     hyperlocalise version

Flags:
  -h, --help   help for hyperlocalise

Use "hyperlocalise [command] --help" for more information about a command.
```

```sh
$> hyperlocalise example 2 5 --add
7

$> hyperlocalise example 2 5 --multiply
10
```

# Makefile Targets
```sh
$> make
bootstrap                      download tool and module dependencies
build                          build golang binary
clean                          clean up environment
cover                          display test coverage
fmt                            format go files
help                           list makefile targets
install                        install golang binary
lint                           lint go files
# pre-commit                     run pre-commit hooks
run                            run the app
staticcheck                    run staticcheck directly
test                           run tests with JSON output and coverage
```

# Contribute
If you find issues in that setup or have some nice features / improvements, I would welcome an issue or a PR :)
