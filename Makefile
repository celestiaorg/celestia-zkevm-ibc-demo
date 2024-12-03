VERSION := $(shell echo $(shell git describe --tags 2>/dev/null || git log -1 --format='%h') | sed 's/^v//')
COMMIT := $(shell git rev-parse --short HEAD)
DOCKER := $(shell which docker)
DOCKER_BUF := $(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace bufbuild/buf
IMAGE := ghcr.io/tendermint/docker-build-proto:latest
DOCKER_PROTO_BUILDER := docker run -v $(shell pwd):/workspace --workdir /workspace $(IMAGE)
PROJECT_NAME=$(shell basename "$(PWD)")
HTTPS_GIT := https://github.com/celestiaorg/celestia-zkevm-ibc-demo
GHCR_REPO := ghcr.io/celestiaorg/simapp

# process linker flags
ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=celestia-zkevm-ibc-demo \
		  -X github.com/cosmos/cosmos-sdk/version.AppName=celestia-zkevm-ibc-demo \
		  -X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
		  -X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \

BUILD_FLAGS := -tags "ledger" -ldflags '$(ldflags)'

## help: Get more info on make commands.
help: Makefile
	@echo " Choose a command run in "$(PROJECT_NAME)":"
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
.PHONY: help

## build: Build the celestia-appd binary into the ./build directory.
build: mod
	@cd ./simapp/simd/
	@mkdir -p build/
	@go build $(BUILD_FLAGS) -o build/ ./simapp/simd/
.PHONY: build

## install: Build and install the celestia-appd binary into the $GOPATH/bin directory.
install:
	@echo "--> Installing simd"
	@go install $(BUILD_FLAGS) ./simapp/simd/
.PHONY: install

## mod: Update all go.mod files.
mod:
	@echo "--> Updating go.mod"
	@go mod tidy
.PHONY: mod

## mod-verify: Verify dependencies have expected content.
mod-verify: mod
	@echo "--> Verifying dependencies have expected content"
	GO111MODULE=on go mod verify
.PHONY: mod-verify

## proto-gen: Generate protobuf files. Requires docker.
proto-gen:
	@echo "--> Generating Protobuf files"
	$(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace tendermintdev/sdk-proto-gen:v0.7 sh ./scripts/protocgen.sh
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

## build-docker: Build the simapp docker image from the current branch. Requires docker.
build-docker:
	@echo "--> Building Docker image"
	$(DOCKER) build -t $(GHCR_REPO) -f docker/Dockerfile .
.PHONY: build-docker

## publish-docker: Publish the simapp docker image to GHCR. Requires Docker and authentication.
publish-docker:
	$(DOCKER) push $(GHCR_REPO)
.PHONY: publish-docker

## lint: Run all linters; golangci-lint, markdownlint, hadolint, yamllint.
lint:
	@echo "--> Running golangci-lint"
	@golangci-lint run
	@echo "--> Running markdownlint"
	@markdownlint --config .markdownlint.yaml '**/*.md'
	@echo "--> Running hadolint"
	@hadolint docker/Dockerfile
	@echo "--> Running yamllint"
	@yamllint --no-warnings . -c .yamllint.yml
.PHONY: lint

## markdown-link-check: Check all markdown links.
markdown-link-check:
	@echo "--> Running markdown-link-check"
	@find . -name \*.md -print0 | xargs -0 -n1 markdown-link-check
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

## init-simapp: Initializes a single local node network. It is useful for testing and development. Try make install && make init-simapp && simd start
init-simapp:
# Warning this will remove all data in simapp home directory
	./scripts/init-simapp.sh
.PHONY: init-simapp

## build-demo: Builds the demo binary into the ./build directory.
build-demo:
	go build -o ./build/demo ./testing/demo
.PHONY: build-demo

## run-demo: Runs the demo binary.
run-demo:
	go run ./testing/demo/main.go
.PHONY: run-demo

## deploy-contracts: Deploys the IBC smart contracts.
deploy-contracts:
	@echo "--> Deploying IBC smart contracts"
	@cd ./solidity-ibc-eureka/scripts && bun install
	@cd ./solidity-ibc-eureka/scripts && forge script E2ETestDeploy.s.sol:E2ETestDeploy --rpc-url http://localhost:8545 --private-key 0x82bfcfadbf1712f6550d8d2c00a39f05b33ec78939d0167be2a737d691f33a6a --broadcast
.PHONY: deploy-contracts
