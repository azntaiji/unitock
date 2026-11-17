# Copy to the repository root and set BINARY. Tools are pinned here and run with
# `go run`, so nothing needs installing and CI uses the same versions (it calls make).
BINARY := <name>

GOLANGCI_LINT_VERSION := v2.15.0
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
GORELEASER_VERSION := v2.19.0
GORELEASER := go run github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)

# Release notes: the CHANGELOG.md section for VERSION (release kit only).
VERSION ?= $(shell cat VERSION 2>/dev/null)
NOTES ?= bin/release-notes.md

.PHONY: build test test-race vet lint fmt golden check release-notes release-check release clean

build:
	go build -o bin/$(BINARY) ./cmd/$(BINARY)

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

# golangci-lint v2 also reports unformatted files, so this is the format check too.
lint:
	$(GOLANGCI_LINT) run

fmt:
	$(GOLANGCI_LINT) fmt

# Rewrite testdata/*.golden after an intended output change, then review the diff.
golden:
	UPDATE_GOLDEN=1 go test -count=1 ./...

# Everything a task must pass before it is called done.
check: build vet lint test-race

# --- Release kit (optional; delete these targets if you did not adopt it) ---------

# Extract the CHANGELOG.md section for VERSION. Run at gate close, after the version is
# applied and before tagging, so the published release cannot lack notes.
release-notes:
	@mkdir -p $(dir $(NOTES))
	@awk -v v="$(VERSION)" '/^## \[/ { if (p) exit; p = (index($$0, "## [" v "]") == 1); next } p' CHANGELOG.md > $(NOTES)
	@grep -q '[^[:space:]]' $(NOTES) || { echo "CHANGELOG.md has no notes under '## [$(VERSION)]'"; exit 1; }

# Validate .goreleaser.yaml and build every target into dist/ without publishing.
# Required before a gate request that proposes a version.
release-check:
	$(GORELEASER) check
	$(GORELEASER) release --snapshot --clean

# Publishes a GitHub Release. Runs in .github/workflows/release.yml on a vX.Y.Z tag.
release: release-notes
	$(GORELEASER) release --clean --release-notes $(NOTES)

clean:
	rm -rf bin dist
