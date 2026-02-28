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

# Check for known vulnerabilities (informational — stdlib vulns require Go upgrades)
vulncheck:
    go tool govulncheck ./...

# Run linter (skip site/ which is Hugo-only)
lint:
    go tool golangci-lint run ./cmd/... ./internal/...

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

# Build with version info from git (for local testing)
build-local:
    #!/usr/bin/env bash
    set -euo pipefail
    version=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
    commit=$(git rev-parse --short HEAD)
    date=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    go build -ldflags "-X main.version=$version -X main.commit=$commit -X main.date=$date" -o bin/timbers ./cmd/timbers
    echo "Built: bin/timbers $version ($commit)"

# Build with explicit version info
build-release version:
    go build -ldflags "-X main.version={{version}}" -o bin/timbers ./cmd/timbers

# Install dev build to ~/.local/bin (overwrites release binary)
install-local: build-local
    cp bin/timbers ~/.local/bin/timbers
    @echo "Installed: $(~/.local/bin/timbers --version)"

# Install latest release from GitHub
install-release:
    curl -fsSL https://raw.githubusercontent.com/gorewood/timbers/main/install.sh | bash

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

# Generate a report using timbers draft piped to claude
# This uses `claude -p` which is cheaper than API token calls.
# Usage: just draft changelog --since 7d
#        just draft standup --last 10
draft +args:
    @go run ./cmd/timbers draft {{args}} | claude -p --model opus

# Generate a report with specific model
# Usage: just draft-model sonnet devblog --last 20
draft-model model +args:
    @go run ./cmd/timbers draft {{args}} | claude -p --model {{model}}

# Regenerate all site example pages from current ledger
examples:
    #!/usr/bin/env bash
    set -eo pipefail
    DATE=$(date +%Y-%m-%d)
    # Dynamic examples: regenerated each release to stay fresh
    TEMPLATES=("release-notes" "decision-log")
    RANGES=("--last 20" "--last 20")
    PIDS=()
    NAMES=()
    for i in "${!TEMPLATES[@]}"; do
        tmpl="${TEMPLATES[$i]}"
        RANGE="${RANGES[$i]}"
        FILE="site/content/examples/${tmpl}.md"
        if [ -f "$FILE" ] && ! git diff --quiet -- "$FILE" 2>/dev/null; then
            echo "Skipping $tmpl (already modified)"
            continue
        fi
        echo "Generating $tmpl ($RANGE)..."
        (
            TITLE=$(echo "$tmpl" | sed 's/-/ /g' | awk '{for(i=1;i<=NF;i++) $i=toupper(substr($i,1,1)) substr($i,2)}1')
            CONTENT=$(go run ./cmd/timbers draft "$tmpl" $RANGE | claude -p --model opus)
            {
                printf '+++\n'
                echo "title = '${TITLE}'"
                echo "date = '${DATE}'"
                echo "tags = ['example', '$tmpl']"
                printf '+++\n'
                echo ""
                echo "Generated with \`timbers draft $tmpl $RANGE | claude -p --model opus\`"
                echo ""
                echo "---"
                echo ""
                echo "$CONTENT"
            } > "site/content/examples/${tmpl}.md"
            echo "Done: $tmpl"
        ) &
        PIDS+=($!)
        NAMES+=("$tmpl")
    done
    # Wait for all and report failures
    FAILED=0
    for i in "${!PIDS[@]}"; do
        if ! wait "${PIDS[$i]}"; then
            echo "FAILED: ${NAMES[$i]}"
            FAILED=1
        fi
    done
    if [ "$FAILED" -eq 1 ]; then
        echo "Some examples failed. Re-run to retry only the failed ones."
        exit 1
    fi
    echo "Done. Static examples (standup, pr-description, sprint-report) are managed by 'just examples-static'."

# Regenerate static examples from a known-good date range (one-time, not per-release)
examples-static:
    #!/usr/bin/env bash
    set -eo pipefail
    DATE=$(date +%Y-%m-%d)
    # Static examples use a fixed date range with dense, high-quality entries (Feb 10-14 2026)
    TEMPLATES=("standup" "pr-description" "sprint-report")
    RANGES=("--since 2026-02-13 --until 2026-02-13" "--since 2026-02-10 --until 2026-02-11" "--since 2026-02-10 --until 2026-02-14")
    PIDS=()
    NAMES=()
    for i in "${!TEMPLATES[@]}"; do
        tmpl="${TEMPLATES[$i]}"
        RANGE="${RANGES[$i]}"
        FILE="site/content/examples/${tmpl}.md"
        if [ -f "$FILE" ] && ! git diff --quiet -- "$FILE" 2>/dev/null; then
            echo "Skipping $tmpl (already modified)"
            continue
        fi
        echo "Generating $tmpl ($RANGE)..."
        (
            TITLE=$(echo "$tmpl" | sed 's/-/ /g' | awk '{for(i=1;i<=NF;i++) $i=toupper(substr($i,1,1)) substr($i,2)}1')
            CONTENT=$(go run ./cmd/timbers draft "$tmpl" $RANGE | claude -p --model opus)
            {
                printf '+++\n'
                echo "title = '${TITLE}'"
                echo "date = '${DATE}'"
                echo "tags = ['example', '$tmpl']"
                printf '+++\n'
                echo ""
                echo "Generated with \`timbers draft $tmpl $RANGE | claude -p --model opus\`"
                echo ""
                echo "---"
                echo ""
                echo "$CONTENT"
            } > "site/content/examples/${tmpl}.md"
            echo "Done: $tmpl"
        ) &
        PIDS+=($!)
        NAMES+=("$tmpl")
    done
    FAILED=0
    for i in "${!PIDS[@]}"; do
        if ! wait "${PIDS[$i]}"; then
            echo "FAILED: ${NAMES[$i]}"
            FAILED=1
        fi
    done
    if [ "$FAILED" -eq 1 ]; then
        echo "Some examples failed. Re-run to retry only the failed ones."
        exit 1
    fi
    echo "Done. Run 'just examples' for dynamic examples (release-notes, decision-log)."

# =============================================================================
# RELEASE (goreleaser)
# =============================================================================

# Generate changelog, commit, tag, and push a release
# Usage: just release 0.3.0
release version:
    #!/usr/bin/env bash
    set -euo pipefail
    ver="{{version}}"
    ver="${ver#v}"
    tag="v${ver}"
    tag_date=$(date +%Y-%m-%d)

    # Check for clean working tree (allow dirty release outputs — they may be pre-generated)
    if ! git diff --quiet -- ':!site/content/examples/' ':!CHANGELOG.md' || ! git diff --cached --quiet; then
        echo "ERROR: Working tree is dirty. Commit or stash changes first."
        exit 1
    fi

    PREV_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
    echo "Generating changelog for $tag (since ${PREV_TAG:-beginning})..."

    # Generate versioned changelog section (pipe to claude -p: uses subscription, not API tokens)
    APPEND="This is release v${ver}. Output ONLY the version section starting from ## [${ver}], no top-level header or preamble."
    if [ -n "$PREV_TAG" ]; then
        RAW=$(go run ./cmd/timbers draft changelog --range "$PREV_TAG"..HEAD \
            --append "$APPEND" | claude -p --model opus)
    else
        RAW=$(go run ./cmd/timbers draft changelog --last 50 \
            --append "$APPEND" | claude -p --model opus)
    fi
    # Strip code fences and duplicate headers that LLMs sometimes add
    SECTION=$(echo "$RAW" | sed '/^```/d' | sed '/^# Changelog/,/^$/d')

    # Prepend new section to existing CHANGELOG.md (after the header)
    if [ -f CHANGELOG.md ]; then
        # Extract header (first 6 lines: title + blank + description + blank + format + blank)
        HEADER=$(head -6 CHANGELOG.md)
        REST=$(tail -n +7 CHANGELOG.md)
        {
            echo "$HEADER"
            echo ""
            echo "$SECTION"
            echo ""
            echo "$REST"
        } > CHANGELOG.md
    else
        echo "$SECTION" > CHANGELOG.md
    fi

    # Add version reference link at the bottom
    LINK="[${ver}]: https://github.com/gorewood/timbers/releases/tag/${tag}"
    if ! grep -q "^\[${ver}\]:" CHANGELOG.md; then
        echo "$LINK" >> CHANGELOG.md
    fi

    echo "Updated CHANGELOG.md"
    echo "---"
    head -30 CHANGELOG.md
    echo "..."
    echo "---"

    # Sync changelog to site example
    {
        printf '+''+''+\n'
        echo "title = 'Changelog'"
        echo "date = '${tag_date}'"
        echo "tags = ['example', 'changelog']"
        printf '+''+''+\n'
        echo ""
        echo "Copied from the repo's [CHANGELOG.md](https://github.com/gorewood/timbers/blob/main/CHANGELOG.md), which is generated by \`just release\` using \`timbers draft changelog | claude -p --model opus\`."
        echo ""
        echo "---"
        echo ""
        cat CHANGELOG.md
    } > site/content/examples/changelog.md

    # Regenerate other site examples
    just examples

    # Update landing page version badge
    sed -i '' "s/v[0-9]*\.[0-9]*\.[0-9]* \&middot; Open Source/v${ver} \&middot; Open Source/" site/layouts/index.html

    # Commit, tag, push
    git add CHANGELOG.md site/content/examples/ site/layouts/index.html
    git commit -m "chore: changelog for $tag"
    git tag "$tag"
    git push origin main "$tag"
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
# BLOG
# =============================================================================

# Generate dev blog post
blog:
    #!/usr/bin/env bash
    set -euo pipefail
    DATE=$(date +%Y-%m-%d)
    WEEK=$(date +%Y)-week-$(date +%V)
    FILE="site/content/posts/${DATE}-${WEEK}.md"
    mkdir -p site/content/posts
    {
        echo "+++"
        echo "title = 'Weekly Update: Week $(date +%V), $(date +%Y)'"
        echo "date = '${DATE}'"
        echo "+++"
        echo ""
        go run ./cmd/timbers draft devblog --since 7d | claude -p --model opus
    } > "$FILE"
    echo "Created: $FILE"

# Preview blog locally
blog-serve:
    cd site && hugo server -D

# Preview changelog for next release (does not modify CHANGELOG.md)
# Usage: just changelog              # preview unreleased changes
#        just changelog 0.3.0        # preview with version heading
changelog version="":
    #!/usr/bin/env bash
    set -euo pipefail
    PREV_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
    APPEND=""
    if [ -n "{{version}}" ]; then
        APPEND="--append 'This is release v{{version}}'"
    fi
    if [ -n "$PREV_TAG" ]; then
        eval go run ./cmd/timbers draft changelog --range "$PREV_TAG"..HEAD $APPEND | claude -p --model opus
    else
        eval go run ./cmd/timbers draft changelog --last 50 $APPEND | claude -p --model opus
    fi

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
