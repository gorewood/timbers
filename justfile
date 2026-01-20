# Timbers Build System
# Usage: just --list

set quiet

default:
    @just --list

# =============================================================================
# SETUP
# =============================================================================

# First-time setup
setup:
    mise trust
    mise install
    go mod download
    @echo "Ready. Run 'just check' to verify."

# Validate toolchain versions
doctor:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Checking toolchain..."

    # Validate Go version (requires 1.25+)
    GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
    if [[ "$(printf '%s\n' "1.25" "$GO_VERSION" | sort -V | head -1)" != "1.25" ]]; then
        echo "FAIL: Go $GO_VERSION < 1.25 required"
        exit 1
    fi
    echo "✓ Go $GO_VERSION"

    # Check just
    just --version >/dev/null 2>&1 && echo "✓ just $(just --version | head -1)" || echo "WARN: just not found"

    echo "All checks passed"

# =============================================================================
# QUALITY GATES
# =============================================================================

# Run all quality checks
check: fmt-check lint test

# Run linter
lint:
    go tool golangci-lint run

# Run tests
test:
    go test -race ./...

# Run tests with coverage
test-cover:
    go test -race -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html

# Check formatting without modifying
fmt-check:
    @test -z "$(gofmt -l .)" || (echo "Files need formatting:" && gofmt -l . && exit 1)

# =============================================================================
# AUTO-FIX
# =============================================================================

# Auto-fix lint and format issues
fix:
    go tool golangci-lint run --fix
    go tool goimports -w .
    gofmt -w .

# =============================================================================
# BUILD
# =============================================================================

# Build the CLI
build:
    go build -o bin/timbers ./cmd/timbers

# Build with version info
build-release version:
    go build -ldflags "-X main.version={{version}}" -o bin/timbers ./cmd/timbers

# Install locally
install:
    go install ./cmd/timbers

# =============================================================================
# DEV WORKFLOW
# =============================================================================

# Run the CLI (pass args after --)
run *args:
    go run ./cmd/timbers {{args}}

# Watch and rebuild on changes (requires watchexec)
watch:
    watchexec -e go -- just build

# =============================================================================
# REPORTING
# =============================================================================

# Generate a report using timbers prompt + claude
# Usage: just prompt changelog --since 7d
#        just prompt exec-summary --last 10
# Model defaults to haiku; override with: just prompt-model sonnet changelog --since 7d
prompt +args:
    @claude -p --model haiku "$(go run ./cmd/timbers prompt {{args}})"

# Generate a report with specific model
# Usage: just prompt-model sonnet devblog-opensource --last 20
prompt-model model +args:
    @claude -p --model {{model}} "$(go run ./cmd/timbers prompt {{args}})"

# =============================================================================
# RELEASE (goreleaser)
# =============================================================================

# Install from source to GOPATH for local testing before release
# Injects version info from git
install-local:
    #!/usr/bin/env bash
    set -euo pipefail
    version=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
    commit=$(git rev-parse --short HEAD)
    date=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    echo "Installing timbers $version ($commit) to GOPATH..."
    go install -ldflags "-X main.version=$version -X main.commit=$commit -X main.date=$date" ./cmd/timbers
    echo "Installed: $(which timbers)"
    timbers --version

# Tag and push a release (triggers GitHub Actions)
# Usage: just release 0.1.0
release version:
    #!/usr/bin/env bash
    set -euo pipefail
    if [[ "{{version}}" =~ ^v ]]; then
        tag="{{version}}"
    else
        tag="v{{version}}"
    fi
    echo "Creating release $tag..."
    git tag "$tag"
    git push origin "$tag"
    echo "Release $tag pushed. GitHub Actions will build and publish."

# Validate goreleaser configuration
release-check:
    goreleaser check

# Test release build locally (no publish)
release-snapshot:
    goreleaser release --snapshot --clean

# Build with goreleaser locally (no publish, no tag required)
release-build:
    goreleaser release --snapshot --clean

# =============================================================================
# CLEANUP
# =============================================================================

# Remove build artifacts
clean:
    rm -rf bin/ dist/
    rm -f coverage.out coverage.html

# Deep clean including caches
clean-all: clean
    go clean -cache -testcache
