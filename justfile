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
# Usage: just prompt-model sonnet devblog --last 20
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
# DEVLOG
# =============================================================================

# Generate a new devlog post from recent entries
# Usage: just devlog                    # Last 5 entries
#        just devlog --last 10          # Last 10 entries
#        just devlog --since 7d         # Since 7 days ago
devlog *args:
    #!/usr/bin/env bash
    set -euo pipefail

    # Default to --last 5 if no args
    ARGS="${*:---last 5}"

    # Generate title from date
    DATE=$(date +%Y-%m-%d)
    SLUG=$(date +%Y%m%d)-devlog

    echo "Generating devlog post..."
    CONTENT=$(go run ./cmd/timbers prompt devblog $ARGS | go run ./cmd/timbers generate --model local)

    # Extract first sentence for title (up to first period)
    TITLE=$(echo "$CONTENT" | head -1 | sed 's/\..*//' | head -c 60)
    if [ ${#TITLE} -eq 60 ]; then TITLE="$TITLE..."; fi

    # Write with frontmatter
    cat > "devlog/content/posts/${SLUG}.md" << EOF
    ---
    title: "${TITLE}"
    date: ${DATE}
    ---

    ${CONTENT}
    EOF

    # Clean up leading whitespace from heredoc
    sed -i '' 's/^    //' "devlog/content/posts/${SLUG}.md"

    echo "Created: devlog/content/posts/${SLUG}.md"

# Regenerate all devlogs from scratch (clears existing posts)
devlog-regen:
    #!/usr/bin/env bash
    set -euo pipefail

    echo "Clearing existing devlog posts..."
    rm -f devlog/content/posts/*.md

    echo "Generating devlog from all entries..."
    CONTENT=$(go run ./cmd/timbers prompt devblog --last 50 | go run ./cmd/timbers generate --model local)

    DATE=$(date +%Y-%m-%d)
    TITLE="Development Log"

    cat > "devlog/content/posts/${DATE}-devlog.md" << EOF
    ---
    title: "${TITLE}"
    date: ${DATE}
    ---

    ${CONTENT}
    EOF

    sed -i '' 's/^    //' "devlog/content/posts/${DATE}-devlog.md"

    echo "Regenerated devlog: devlog/content/posts/${DATE}-devlog.md"

# Run Hugo dev server for devlog
devlog-serve:
    cd devlog && hugo server -D --bind 0.0.0.0

# Build static devlog site
devlog-build:
    cd devlog && hugo --minify

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
