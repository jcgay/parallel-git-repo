# For macOS users:
#  - brew install coreutils
# Prepare a GNU sed for macOS users (brew install gnu-sed)
sed = $(shell { command -v gsed || command -v sed; } 2>/dev/null)

# Set an output prefix, which is the local directory if not specified
PREFIX?=$(shell pwd)

# Setup name variables for the package/tool
NAME := parallel-git-repo

# Set any default go build tags
BUILDTAGS :=

# The version and commit are read at runtime from the binary's build info, so
# no version ldflags are needed here; GoReleaser stamps the version on release.
GO_LDFLAGS_STATIC=-ldflags "-w -extldflags -static"

all: clean build fmt lint test vet install ## Runs a clean, build, fmt, lint, test, vet and install

.PHONY: build
build: $(NAME) ## Builds a dynamic executable or package

$(NAME): *.go
	@echo "+ $@"
	go build -tags "$(BUILDTAGS)" -o $(NAME) .

.PHONY: static
static: ## Builds a static executable
	@echo "+ $@"
	CGO_ENABLED=0 go build \
				-tags "$(BUILDTAGS) static_build" \
				${GO_LDFLAGS_STATIC} -o $(NAME) .

.PHONY: fmt
fmt: ## Verifies all files have men `gofmt`ed
	@echo "+ $@"
	@gofmt -s -l . | grep -v vendor | tee /dev/stderr

.PHONY: lint
lint: ## Verifies `golint` passes
	@echo "+ $@"
	@staticcheck ./... | grep -v vendor | tee /dev/stderr

.PHONY: test
test: ## Runs the go tests
	@echo "+ $@"
	@go test -v -tags "$(BUILDTAGS)" $(shell go list ./... | grep -v vendor)

.PHONY: vet
vet: ## Verifies `go vet` passes
	@echo "+ $@"
	@go vet $(shell go list ./... | grep -v vendor) | tee /dev/stderr

.PHONY: install
install: ## Installs the executable or package
	@echo "+ $@"
	@go install .

.PHONY: tag
tag: ## Create a new git tag to prepare a release, e.g. make tag VERSION=v1.2.0
	@test -n "$(VERSION)" || { echo "VERSION is required, e.g. make tag VERSION=v1.2.0"; exit 1; }
	git tag -sa $(VERSION) -m "$(VERSION)"
	@echo "Run git push origin $(VERSION) to push your new tag to GitHub; goreleaser then builds and publishes the release."

.PHONY: clean
clean: ## Cleanup any build binaries or packages
	@echo "+ $@"
	$(RM) $(NAME)
	$(RM) -r dist

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'