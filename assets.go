// Package netlog exposes the embedded, pre-built Svelte SPA so the single binary
// can serve the frontend without external files.
//
// web/dist is produced by `pnpm build` (see justfile). It is .gitignored, so a
// fresh checkout must build the frontend before `go build` — `just build` and
// `just check` do this automatically.
package netlog

//go:generate just build-web

import (
	"embed"
	"io/fs"
)

// web/dist is fully regenerated on every build (Vite's emptyOutDir clears it and
// `just build-web` wipes it first), so only the latest assets are embedded — the
// binary doesn't accrue stale bundles. The plain (non-"all:") form also skips any
// stray dot/underscore files, which the SPA never needs.
//
//go:embed web/dist
var distFS embed.FS

// DistFS returns the embedded SPA rooted at web/dist (so "index.html" resolves
// directly).
func DistFS() (fs.FS, error) {
	return fs.Sub(distFS, "web/dist")
}
