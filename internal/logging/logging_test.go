package logging

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	cases := map[string]slog.Level{
		"trace": LevelTrace,
		"TRACE": LevelTrace,
		"debug": slog.LevelDebug,
		"":      slog.LevelInfo,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}
	for in, want := range cases {
		got, err := ParseLevel(in)
		if err != nil {
			t.Fatalf("ParseLevel(%q) error: %v", in, err)
		}
		if got != want {
			t.Errorf("ParseLevel(%q) = %v, want %v", in, got, want)
		}
	}
	if _, err := ParseLevel("nonsense"); err == nil {
		t.Error("expected error for unknown level")
	}
}

func TestTraceRendersNameAndRespectsLevel(t *testing.T) {
	var buf bytes.Buffer
	logger, err := New(Options{Level: "trace", Format: "text", Output: &buf})
	if err != nil {
		t.Fatal(err)
	}
	Trace(context.Background(), logger, "hello")
	out := buf.String()
	if !strings.Contains(out, "TRACE") {
		t.Errorf("expected TRACE label in output, got %q", out)
	}

	// At info level, TRACE messages must be suppressed.
	buf.Reset()
	logger, err = New(Options{Level: "info", Format: "text", Output: &buf})
	if err != nil {
		t.Fatal(err)
	}
	Trace(context.Background(), logger, "should-not-appear")
	if buf.Len() != 0 {
		t.Errorf("expected TRACE suppressed at info level, got %q", buf.String())
	}
}

func TestNewRejectsBadFormat(t *testing.T) {
	if _, err := New(Options{Level: "info", Format: "xml", Output: &bytes.Buffer{}}); err == nil {
		t.Error("expected error for unknown format")
	}
}
