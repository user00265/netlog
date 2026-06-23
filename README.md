# NetLog

An offline-first web app for logging amateur-radio **directed nets**. A Net
Control Station (NCS) opens a net, logs check-ins live, and everything syncs
between the browser and the backend — working the same whether you're online or
not. It ships as a **single Go binary** with the Svelte SPA embedded, backed by
SQLite.

## Features

- **Offline-first logging** — check-ins write to the browser (IndexedDB/Dexie)
  first, then sync to the backend in the background; open the app on another
  machine and it syncs down.
- **Net lifecycle** — create a net, OPEN it (start time stamped in UTC), log
  check-ins, then CLOSE it (end time stamped). Closed nets are read-only.
- **Check-in details** — callsign, on-air name, has-traffic and short-time
  flags, with NCS edit/delete while the net is open.
- **Callbook + DXCC enrichment** — each callsign is looked up against QRZ and
  HamQTH (configurable primary + fallback) and a `cty.xml` DXCC dataset to show
  the country flag; results are cached, with a per-callsign refresh button.
- **Auto-saving net notes** the NCS can keep during a live net.
- **Export** closed nets to CSV (offline) or a print-ready PDF.
- **Accounts** — username/password (argon2id) with optional OIDC; the first
  account is the admin, and only the admin can add operators.
- **Times in UTC + your local time**, 24h or 12h, per operator.
- Dark/light themes, responsive/mobile, installable PWA.

## Requirements

- Go 1.26+
- Node 20+ and [pnpm](https://pnpm.io/)
- [`just`](https://github.com/casey/just) (recommended)
- Optionally [GoReleaser](https://goreleaser.com/) for release builds

## Configuration

Copy the example config and fill in your details (callbook credentials, OIDC,
etc.). Secrets may also be supplied via environment variables.

```sh
cp config.example.yaml config.yaml
```

The server defaults to `http://localhost:8080`. On first run with an empty
database you'll be prompted to create the admin account.

## Building

### With `just` (recommended)

```sh
just install     # fetch Go modules + frontend deps
just build       # build the SPA and the single ./netlog binary
just run         # build the SPA and run the server

just rebuild     # clean wipe of generated assets, then a fresh build
just check       # the full verification gate (lint, vet, tests, frontend)
just dev-web     # frontend dev server (proxies /api to :8080)
```

The frontend lives in `web/` and is embedded into the binary via `go:embed`, so
the SPA is always rebuilt before the binary — `just build` can't ship stale
assets.

### With GoReleaser (optional, for releases)

```sh
goreleaser release --clean            # tagged release
goreleaser build --clean --snapshot   # local cross-platform snapshot
```

GoReleaser runs `go generate ./...` (which rebuilds the SPA) and `just check`
before building, and stamps the version/commit into the binary.

### By hand (manual commands)

```sh
# 1. Build the frontend into web/dist (embedded by the binary)
cd web && pnpm install && pnpm build && cd ..

# 2. Build the binary (pure Go, no cgo)
CGO_ENABLED=0 go build -o netlog ./cmd/netlog

# 3. Run it
./netlog -config config.yaml
```

Note: the binary embeds `web/dist`, so the frontend must be built first — a bare
`go build` will fail if `web/dist` doesn't exist.

## License

NetLog is released under the [MIT License](LICENSE) — © 2026 Elisamuel "Sam"
Resto Donate, KF0ACN.

## A note on AI assistance

This program was made with assistance of an AI LLM. In practice that means parts
of the code, configuration, and documentation were drafted or refined with an AI
coding assistant and then reviewed and tested. It is not a guarantee of
correctness: please don't assume any piece is bug-free just because it looks
polished, and review the code yourself before trusting it in production.

## Contributing

If you hit a bug, please **open an issue** with steps to reproduce. If you're
able, **pull requests are very welcome** — fixes and improvements of any size.
If you found NetLog useful and want to make it better, jump in. 73!
