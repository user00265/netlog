package config

import (
	"fmt"
	"runtime"
)

// Build information. These are overridden at release time via -ldflags by
// GoReleaser (see .goreleaser.yaml); the defaults apply to local/dev builds.
var (
	Version   = "dev"     // release tag, e.g. 1.3.7
	Revision  = "unknown" // short commit hash
	BuildTime = ""        // RFC3339 build timestamp
)

// VersionString returns a human-readable build identifier.
func VersionString() string {
	return fmt.Sprintf("%s (%s, %s/%s)", Version, Revision, runtime.GOOS, runtime.GOARCH)
}
