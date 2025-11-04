SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin

## Go flags
export GOFLAGS = -mod=vendor

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of the package
# touch $1-$3 is used to allow cache busting in a CI environment
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Removing any outdated version of $(1)";\
rm -f $(1)*;\
echo "Downloading $(2)@$(3)" ;\
GOBIN=$(LOCALBIN) GOFLAGS="-mod=mod" go install "$(2)@$(3)" ;\
touch "$(1)-$(3)";\
rm -rf $$TMP_DIR ;\
}
endef

## Tool Binaries
GOIMPORTS_REVISER = $(LOCALBIN)/goimports-reviser
GOFUMPT = $(LOCALBIN)/gofumpt
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint
MCPTOOLS = $(LOCALBIN)/mcptools

## Tool Versions
GOIMPORTS_REVISER_VERSION = v3.9.1
GOFUMPT_VERSION = v0.8.0
GOLANGCI_LINT_VERSION = v2.1.6
MCPTOOLS_VERSION = v0.7.1

GOIMPORTS_REVISER_ARGS = -project-name github.com/krmcbride/mcp-grafana


.PHONY: all
all: help

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)


##@ Dependencies

$(LOCALBIN):
	mkdir -p $(LOCALBIN)

.PHONY: install-goimports-reviser
install-goimports-reviser: $(GOIMPORTS_REVISER) ## Install goimports-reviser
$(GOIMPORTS_REVISER): $(LOCALBIN)
	$(call go-install-tool,$(GOIMPORTS_REVISER),github.com/incu6us/goimports-reviser/v3,$(GOIMPORTS_REVISER_VERSION))
 
.PHONY: install-gofumpt
install-gofumpt: $(GOFUMPT) ## Install gofumpt
$(GOFUMPT): $(LOCALBIN)
	$(call go-install-tool,$(GOFUMPT),mvdan.cc/gofumpt,$(GOFUMPT_VERSION))
 
.PHONY: install-golangci-lint
install-golangci-lint: $(GOLANGCI_LINT) ## Install golangci-lint
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))
 
.PHONY: install-mcptools
install-mcptools: $(MCPTOOLS) ## Install golangci-lint
$(MCPTOOLS): $(LOCALBIN)
	$(call go-install-tool,$(MCPTOOLS),github.com/f/mcptools/cmd/mcptools,$(MCPTOOLS_VERSION))

.PHONY: install-tools
install-tools: install-goimports-reviser install-gofumpt install-golangci-lint install-mcptools ## download dependencies in one shot
 

##@ Development

.PHONY: fmt
fmt: gofumpt goimports-reviser ## Format code

.PHONY: gofumpt
gofumpt: install-gofumpt ## Format code with gofumpt (strict formatting)
	$(GOFUMPT) -w ./cmd ./internal

.PHONY: goimports-reviser
goimports-reviser: install-goimports-reviser ## Format code and fix imports.
	$(GOIMPORTS_REVISER) $(GOIMPORTS_REVISER_ARGS) -recursive ./cmd ./internal

.PHONY: lint
lint: install-golangci-lint ## Run golangci-lint against code.
	$(GOLANGCI_LINT) run

.PHONY: mcp-shell
mcp-shell: install-mcptools ## Run the MCP server with mcptools shell
	$(MCPTOOLS) shell go run cmd/server/main.go

.PHONY: test
test: ## Run tests.
	go test -v ./...


##@ Build

.PHONY: build
build: ## Build the agent
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o dist/server cmd/server/main.go

##@ CI

.PHONY: test-ci
test-ci: ## Run tests in CI
	GOFLAGS="" go test -coverprofile=coverage.out ./...
	GOFLAGS="" go tool cover -func=coverage.out

.PHONY: format-ci
format-ci: install-gofumpt install-goimports-reviser ## Check code formatting in CI
	@echo "Checking gofumpt..."
	@FILES=$$($(GOFUMPT) -l ./cmd ./internal); \
	if [ -n "$$FILES" ]; then \
		echo "Error: The following files need formatting (run 'make fmt'):"; \
		echo "$$FILES"; \
		exit 1; \
	fi
	@echo "Checking goimports-reviser..."
	@$(GOIMPORTS_REVISER) $(GOIMPORTS_REVISER_ARGS) -list-diff -recursive ./cmd ./internal >/dev/null 2>&1; \
	if [ $$? -ne 0 ]; then \
		echo "Error: Files need import formatting (run 'make fmt')"; \
		$(GOIMPORTS_REVISER) $(GOIMPORTS_REVISER_ARGS) -list-diff -recursive ./cmd ./internal 2>/dev/null || true; \
		exit 1; \
	fi
	@echo "All files are properly formatted!"

.PHONY: lint-ci
lint-ci: install-golangci-lint ## Run linters in CI
	GOFLAGS="" $(GOLANGCI_LINT) run

.PHONY: build-ci
build-ci: ## Build the MCP server in CI
	GOFLAGS="" CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o dist/server cmd/server/main.go

