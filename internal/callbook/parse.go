package callbook

import (
	"context"
	"errors"
	"strconv"
	"strings"
)

// ErrCallsignNotFound indicates the provider has no record for the callsign.
var ErrCallsignNotFound = errors.New("callsign not found")

// Provider looks up a callsign in a single callbook.
type Provider interface {
	// Name returns the provider key (e.g. "qrz").
	Name() string
	// Lookup returns a normalized record, ErrCallsignNotFound, or another error.
	Lookup(ctx context.Context, callsign string) (*Record, error)
}

func intPtr(s string) *int {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &v
}

func floatPtr(s string) *float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}
