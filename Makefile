projectname?=hyperlocalise
version?=$(shell git describe --abbrev=0 --tags 2>/dev/null || echo dev)
golangci_lint_version?=v2.10.1
gobin?=$(shell go env GOPATH)/bin
golangci_lint_bin?=$(gobin)/golangci-lint

default: help

.PHONY: help
help: ## list makefile targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## build golang binary
	@go build -ldflags "-X main.version=$(version)" -o $(projectname)

.PHONY: install
install: ## install golang binary
	@go install -ldflags "-X main.version=$(version)"

.PHONY: run
run: ## run the app
	@go run -ldflags "-X main.version=$(version)"  main.go

.PHONY: bootstrap
bootstrap: ## download tool and module dependencies
	go mod download
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(golangci_lint_version)


.PHONY: test
test: clean ## run tests with JSON output and coverage
	go test -json -cover -parallel=1 -coverprofile=coverage.out ./... > test-report.jsonl
	go tool cover -func=coverage.out | sort -rnk3


.PHONY: clean
clean: ## clean up environment
	@rm -rf coverage.out test-report.jsonl dist/ $(projectname)


.PHONY: cover
cover: ## display test coverage
	go test -v -race $(shell go list ./... | grep -v /vendor/) -v -coverprofile=coverage.out
	go tool cover -func=coverage.out


.PHONY: fmt
fmt: ## format go files
	go tool gofumpt -w .
	go tool gci write .


.PHONY: lint
lint: ## lint go files
	$(golangci_lint_bin) run

.PHONY: staticcheck
staticcheck: ## run staticcheck directly
	go tool staticcheck ./...
