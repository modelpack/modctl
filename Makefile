#/*
# *     Copyright 2024 The CNAI Authors
# *
# * Licensed under the Apache License, Version 2.0 (the "License");
# * you may not use this file except in compliance with the License.
# * You may obtain a copy of the License at
# *
# *      http://www.apache.org/licenses/LICENSE-2.0
# *
# * Unless required by applicable law or agreed to in writing, software
# * distributed under the License is distributed on an "AS IS" BASIS,
# * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# * See the License for the specific language governing permissions and
# * limitations under the License.
# */

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif
# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

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

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: fmt vet ## Run unit test and display the coverage.
	go test $$(go list ./pkg/...) -coverprofile cover.out
	go tool cover -func cover.out

GOLANGCI_LINT = $(shell pwd)/bin/golangci-lint
GOLANGCI_LINT_VERSION ?= v1.54.2
golangci-lint:
	@[ -f $(GOLANGCI_LINT) ] || { \
	set -e ;\
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell dirname $(GOLANGCI_LINT)) $(GOLANGCI_LINT_VERSION) ;\
	}

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter & yamllint
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes

GOIMPORTS := $(shell command -v goimports 2> /dev/null)
ifeq ($(GOIMPORTS),)
GOIMPORTS_INSTALL = go install golang.org/x/tools/cmd/goimports@latest
else
GOIMPORTS_INSTALL =
endif

$(GOIMPORTS_INSTALL):
	$(GOIMPORTS_INSTALL)

lint-fix: $(GOIMPORTS_INSTALL)
	$(GOIMPORTS) -l -w .
	$(GOLANGCI_LINT) run --fix

##@ Build

.PHONY: build
build: fmt vet
	go build -o output/modctl main.go

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

.PHONY: gen
gen: gen-mockery## Generate all we need!

.PHONY: gen-mockery check-mockery install-mockery
gen-mockery: check-mockery ## Generate mockery code
	@echo "generating mockery code according to .mockery.yaml"
	@mockery

check-mockery:
	@which mockery > /dev/null || { echo "mockery not found. Trying to install via Homebrew..."; $(MAKE)  install-mockery; }
	@mockery --version | grep -q "2.46.3" || { echo "mockery version is not v2.46.3. Trying to install the correct version..."; $(MAKE)  install-mockery; }

install-mockery:
	@if command -v brew > /dev/null; then \
		echo "Installing mockery via Homebrew"; \
		brew install mockery; \
	else \
		echo "Error: Homebrew is not installed. Please install Homebrew first and ensure it's in your PATH."; \
		exit 1; \
	fi
