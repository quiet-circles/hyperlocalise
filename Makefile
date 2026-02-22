projectname?=hyperlocalise
version?=$(shell git describe --abbrev=0 --tags 2>/dev/null || echo dev)

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
	go tool golangci-lint run

.PHONY: staticcheck
staticcheck: ## run staticcheck directly
	go tool staticcheck ./...
