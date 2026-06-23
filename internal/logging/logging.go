// Package logging provides the single shared slog logger used across the whole
// application. Nothing in NetLog should log via the standard log package or
// fmt.Print*; everything goes through a *slog.Logger obtained here.
//
// We extend slog's levels with a TRACE level (below DEBUG) reserved for very
// high-volume diagnostics such as per-request HTTP logging.
package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// LevelTrace sits below slog.LevelDebug (-4). HTTP request logging uses it.
const LevelTrace = slog.Level(-8)

// levelNames maps our levels (including TRACE) to display strings.
var levelNames = map[slog.Level]string{
	LevelTrace: "TRACE",
}

// Options configures the root logger.
type Options struct {
	// Level is the minimum level to emit: trace, debug, info, warn, error.
	Level string
	// Format is "text" or "json".
	Format string
	// Output is where logs are written (typically os.Stdout).
	Output io.Writer
}

// ParseLevel converts a case-insensitive level name into a slog.Level,
// including the custom "trace" level.
func ParseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "trace":
		return LevelTrace, nil
	case "debug":
		return slog.LevelDebug, nil
	case "", "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unknown log level %q", s)
	}
}

// New builds the root logger from Options. An invalid level falls back to info
// rather than failing, but the error is surfaced to the caller.
func New(opts Options) (*slog.Logger, error) {
	level, err := ParseLevel(opts.Level)

	handlerOpts := &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: replaceAttr,
	}

	var handler slog.Handler
	switch strings.ToLower(strings.TrimSpace(opts.Format)) {
	case "json":
		handler = slog.NewJSONHandler(opts.Output, handlerOpts)
	case "", "text":
		handler = slog.NewTextHandler(opts.Output, handlerOpts)
	default:
		return nil, fmt.Errorf("unknown log format %q", opts.Format)
	}

	return slog.New(handler), err
}

// replaceAttr renders our custom TRACE level with its name instead of the
// numeric "DEBUG-4" slog would otherwise print.
func replaceAttr(_ []string, a slog.Attr) slog.Attr {
	if a.Key == slog.LevelKey {
		if lvl, ok := a.Value.Any().(slog.Level); ok {
			if name, found := levelNames[lvl]; found {
				a.Value = slog.StringValue(name)
			}
		}
	}
	return a
}

// Trace logs at the custom TRACE level on the given logger.
func Trace(ctx context.Context, l *slog.Logger, msg string, args ...any) {
	l.Log(ctx, LevelTrace, msg, args...)
}
