VERSION := $(shell echo $(shell git describe --tags 2>/dev/null || git log -1 --format='%h') | sed 's/^v//')
COMMIT := $(shell git rev-parse --short HEAD)
DOCKER := $(shell which docker)
DOCKER_BUF := $(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace bufbuild/buf
IMAGE := ghcr.io/tendermint/docker-build-proto:latest
DOCKER_PROTO_BUILDER := docker run -v $(shell pwd):/workspace --workdir /workspace $(IMAGE)
PROJECT_NAME=$(shell basename "$(PWD)")
HTTPS_GIT := https://github.com/celestiaorg/celestia-zkevm-ibc-demo
SIMAPP_GHCR_REPO := ghcr.io/celestiaorg/celestia-zkevm-ibc-demo/simapp
CELESTIA_PROVER_GHCR_REPO := ghcr.io/celestiaorg/celestia-zkevm-ibc-demo/celestia-prover
EVM_PROVER_GHCR_REPO := ghcr.io/celestiaorg/celestia-zkevm-ibc-demo/evm-prover
INDEXER_GHCR_REPO := ghcr.io/celestiaorg/celestia-zkevm-ibc-demo/indexer

# process linker flags
ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=celestia-zkevm-ibc-demo \
		  -X github.com/cosmos/cosmos-sdk/version.AppName=celestia-zkevm-ibc-demo \
		  -X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
		  -X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \

BUILD_FLAGS := -tags "ledger" -ldflags '$(ldflags)'

## help: Get more info on make commands.
help: Makefile
	@echo " Choose a command run in "$(PROJECT_NAME)":"
	@sed -n 's/^##//p' $< | sort | column -t -s ':' | sed -e 's/^/ /'
.PHONY: help

## install-dependencies: Install all dependencies needed for the demo.
install-dependencies:
	@echo "--> Setting up Solidity IBC Eureka submodule"
	@cd ./solidity-ibc-eureka && bun install && just install-operator
.PHONY: install-dependencies

## check-dependencies: Check if all dependencies are installed.
check-dependencies:
	@echo "--> Checking if all dependencies are installed"
	@if command -v cargo >/dev/null 2>&1; then \
		echo "cargo is installed."; \
	else \
		echo "Error: cargo is not installed. Please install Rust."; \
		exit 1; \
	fi
	@if command -v forge >/dev/null 2>&1; then \
		echo "foundry is installed."; \
	else \
		echo "Error: forge is not installed. Please install Foundry."; \
		exit 1; \
	fi
	@if command -v bun >/dev/null 2>&1; then \
		echo "bun is installed."; \
	else \
		echo "Error: bun is not installed. Please install bun."; \
		exit 1; \
	fi
	@if command -v just >/dev/null 2>&1; then \
		echo "just is installed."; \
	else \
		echo "Error: just is not installed. Please install just."; \
		exit 1; \
	fi
	@if command -v cargo prove >/dev/null 2>&1; then \
		echo "cargo prove is installed."; \
	else \
		echo "Error: succinct is not installed. Please install SP1."; \
		exit 1; \
	fi
	@if command -v operator >/dev/null 2>&1; then \
		echo "operator is installed."; \
	else \
		echo "Error: operator is not installed. Please run install-dependencies."; \
		exit 1; \
	fi
	@echo "All dependencies are installed."
.PHONY: check-dependencies

## demo: Run the entire demo.
demo:
	@make start
	@make setup
	@make transfer
	@make transfer-back
.PHONY: demo

## start: Start all Docker containers for the demo.
start:
	@docker compose -f docker-compose.rollkit.yml up --detach
.PHONY: start

## setup: Set up the IBC light clients.
setup:
	@echo "--> Creating genesis.json for Tendermint light client"
	@cd ./solidity-ibc-eureka && cargo run --quiet --bin operator --release -- genesis -o scripts/genesis.json --proof-type groth16
	@echo "--> Creating IBC light clients"
	@go run ./testing/demo/pkg/setup/
.PHONY: setup

## transfer: Transfer tokens from simapp to the EVM roll-up.
transfer:
	@echo "--> Transferring tokens from simapp to the EVM roll-up"
	@go run ./testing/demo/pkg/transfer/ transfer
.PHONY: transfer

## transfer-back: Transfer tokens back from the EVM roll-up to simapp.
transfer-back:
	@echo "--> Transferring tokens back from the EVM roll-up to simapp"
	@go run ./testing/demo/pkg/transfer/ transfer-back
.PHONY: transfer-back

## query-balance: Query the balances on SimApp and EVM roll-up.
query-balance:
	@echo "--> Querying the balances..."
	@go run ./testing/demo/pkg/transfer/ query-balance
.PHONY: query-balance

## stop: Stop all Docker containers and remove the tmp directory.
stop:
	@echo "--> Stopping all Docker containers"
	@docker compose -f docker-compose.rollkit.yml down
	@docker compose -f docker-compose.rollkit.yml rm
	@echo "--> Removing the tmp directory"
	@rm -rf .tmp
.PHONY: stop

## build: Build the simapp and indexer binaries into the ./build directory.
build: build-simapp build-indexer
.PHONY: build

## build-simapp: Build the simapp binary into the ./build directory.
build-simapp: mod
	@cd ./simapp/simd/
	@mkdir -p build/
	@go build $(BUILD_FLAGS) -o build/ ./simapp/simd/
.PHONY: build-simapp

## build-indexer: Build the indexer binary
build-indexer:
	@cd ./indexer
	@mkdir -p build/
	@cd indexer && go build $(BUILD_FLAGS) -o build/ .
.PHONY: build-indexer

## build-evm-prover: Build the EVM prover binary
build-evm-prover:
	@cargo build --release --bin evm-prover
.PHONY: build-evm-prover

## install: Install the simapp binary into the $GOPATH/bin directory.
install: install-simapp
.PHONY: install

## install-simapp: Build and install the simapp binary into the $GOPATH/bin directory.
install-simapp:
	@echo "--> Installing simd"
	@go install $(BUILD_FLAGS) ./simapp/simd/
.PHONY: install-simapp

## mod: Update all go.mod files.
mod:
	@echo "--> Updating go.mod"
	@go mod tidy
.PHONY: mod

## proto-gen: Generate protobuf files. Requires docker.
proto-gen:
	@echo "--> Generating Protobuf files"
	$(DOCKER_BUF) generate
.PHONY: proto-gen

## proto-lint: Lint protobuf files. Requires docker.
proto-lint:
	@echo "--> Linting Protobuf files"
	@$(DOCKER_BUF) lint --error-format=json
.PHONY: proto-lint

## proto-check-breaking: Check if there are any breaking change to protobuf definitions.
proto-check-breaking:
	@echo "--> Checking if Protobuf definitions have any breaking changes"
	@$(DOCKER_BUF) breaking --against $(HTTPS_GIT)#branch=main
.PHONY: proto-check-breaking

## proto-format: Format protobuf files. Requires Docker.
proto-format:
	@echo "--> Formatting Protobuf files"
	@$(DOCKER_PROTO_BUILDER) find . -name '*.proto' -path "./proto/*" -exec clang-format -i {} \;
.PHONY: proto-format

## docker: Build the all Docker images.
docker: build-simapp-docker build-indexer-docker build-celestia-prover-docker build-evm-prover-docker
.PHONY: docker

## build-simapp-docker: Build the simapp docker image from the current branch. Requires docker.
build-simapp-docker: build-simapp
	@echo "--> Building simapp Docker image"
	$(DOCKER) build -t $(SIMAPP_GHCR_REPO) --file docker/simapp.Dockerfile .
.PHONY: build-simapp-docker

## build-indexer-docker: Build the indexer docker image. Requires docker.
build-indexer-docker: build-indexer
	@echo "--> Building indexer Docker image"
	$(DOCKER) build -t $(INDEXER_GHCR_REPO) --file docker/indexer.Dockerfile indexer
.PHONY: build-indexer-docker

## build-celestia-prover-docker: Build the celestia prover docker image from the current branch. Requires docker.
build-celestia-prover-docker:
	@echo "--> Building celestia prover Docker image"
	$(DOCKER) build -t $(CELESTIA_PROVER_GHCR_REPO) --file docker/celestia_prover.Dockerfile .
.PHONY: build-celestia-prover-docker

## build-evm-prover-docker: Build the EVM prover docker image from the current branch. Requires docker.
build-evm-prover-docker: build-evm-prover
	@echo "--> Building EVM prover Docker image"
	$(DOCKER) build -t $(EVM_PROVER_GHCR_REPO) --file docker/evm_prover.Dockerfile .
.PHONY: build-evm-prover-docker

# publish: Publish all Docker images to GHCR. Requires Docker and authentication.
publish: publish-simapp-docker publish-celestia-prover-docker publish-evm-prover-docker
.PHONY: publish

## publish-simapp-docker: Publish the simapp docker image to GHCR. Requires Docker and authentication.
publish-simapp-docker:
	$(DOCKER) push $(SIMAPP_GHCR_REPO)
.PHONY: publish-simapp-docker

## publish-celestia-prover-docker: Publish the celestia prover docker image. Requires docker.
publish-celestia-prover-docker:
	$(DOCKER) push $(CELESTIA_PROVER_GHCR_REPO)
.PHONY: publish-celestia-prover-docker

## publish-evm-prover-docker: Publish the EVM prover docker image. Requires docker.
publish-evm-prover-docker:
	$(DOCKER) push $(EVM_PROVER_GHCR_REPO)
.PHONY: publish-evm-prover-docker

## lint: Run all linters; golangci-lint, markdownlint, hadolint, yamllint.
lint:
	@echo "--> Running golangci-lint"
	@golangci-lint run
	@echo "--> Running markdownlint"
	@markdownlint --config .markdownlint.yaml '**/*.md'
	@echo "--> Running hadolint"
	@hadolint docker/**
	@echo "--> Running yamllint"
	@yamllint --no-warnings . -c .yamllint.yml
.PHONY: lint

## markdown-link-check: Check all markdown links.
markdown-link-check:
	@echo "--> Running markdown-link-check"
# Skip the solidity-ibc-eureka directory because we don't want to fix their broken links.
	@find . -name \*.md -not -path "./solidity-ibc-eureka/*" -print0 | xargs -0 -n1 markdown-link-check
.PHONY: markdown-link-check

## fmt: Format files per linters golangci-lint and markdownlint.
fmt:
	@echo "--> Running golangci-lint --fix"
	@golangci-lint run --fix
	@echo "--> Running markdownlint --fix"
	@markdownlint --fix --quiet --config .markdownlint.yaml .
.PHONY: fmt

## test: Run tests.
test:
	@echo "--> Running tests"
	@go test -timeout 30m ./...
.PHONY: test
