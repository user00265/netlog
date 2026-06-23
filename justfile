# NetLog task runner. `just check` is the sole verify gate.

set shell := ["bash", "-uc"]

# Best-effort version stamping for local/dev builds. GoReleaser stamps release
# builds with the real tag/commit/date (see .goreleaser.yaml).
commit := `git rev-parse --short HEAD 2>/dev/null || echo unknown`
ldflags := "-X netlog/internal/config.Version=dev -X netlog/internal/config.Revision=" + commit

# Default: list recipes
default:
    @just --list

# Install all dependencies (Go modules + frontend)
install:
    go mod download
    cd web && pnpm install

# Wire git up to the tracked .githooks directory (run once after cloning).
# The pre-commit hook runs `just check` and refuses the commit on failure.
install-hooks:
    git config core.hooksPath .githooks

# Build the embedded SPA via `go generate` (runs the //go:generate in assets.go).
generate:
    go generate ./...

# Build the frontend SPA into web/dist (embedded by the Go binary). dist is wiped
# first so the embed only ever contains the latest assets, never stale bundles.
build-web:
    rm -rf web/dist
    cd web && pnpm build

# Remove generated build artifacts (embedded SPA + binary).
clean:
    rm -rf web/dist netlog

# Deep clean: also drop installed deps and Go's test cache. Run `install` after.
clean-all: clean
    rm -rf web/node_modules
    go clean -testcache

# Build the single binary (embeds web/dist; builds the SPA first)
build: build-web
    CGO_ENABLED=0 go build -ldflags "{{ ldflags }}" -o netlog ./cmd/netlog

# Clean rebuild: wipe generated assets, then build fresh.
rebuild: clean build

# Run the server (expects ./config.yaml or NETLOG_CONFIG)
run: build-web
    go run -ldflags "{{ ldflags }}" ./cmd/netlog

# Run frontend dev server (proxies /api to the Go backend on :8080)
dev-web:
    cd web && pnpm dev

# Format Go + frontend
fmt:
    gofmt -w -s .
    cd web && pnpm format

# The sole verification gate: everything must pass here.
check: build-web go-check fe-check

go-check:
    gofmt -w -s .
    go mod tidy
    go vet ./...
    golangci-lint run ./...
    go test ./...

fe-check:
    cd web && pnpm lint && pnpm check
