// Package validate centralizes input validation and sanitization. NetLog
// distrusts all external input — from the browser, from callbook/cty APIs, and
// from the backend to the SPA — so values are checked and normalized at every
// boundary before use.
package validate

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/go-playground/validator/v10"
)

// validatorInstance is shared and safe for concurrent use.
var validatorInstance = validator.New(validator.WithRequiredStructEnabled())

// Struct validates a struct using its `validate` tags, returning a friendly
// error listing the offending fields.
func Struct(v any) error {
	if err := validatorInstance.Struct(v); err != nil {
		var invalid *validator.InvalidValidationError
		if errors.As(err, &invalid) {
			return fmt.Errorf("validate: %w", err)
		}
		var fields []string
		var verrs validator.ValidationErrors
		if errors.As(err, &verrs) {
			for _, fe := range verrs {
				fields = append(fields, fe.Field())
			}
		}
		return fmt.Errorf("invalid input: %s", strings.Join(fields, ", "))
	}
	return nil
}

// callsignPattern bounds an amateur callsign after normalization: letters,
// digits, and slashes (for portable/operating indicators), 3–16 chars.
var callsignPattern = regexp.MustCompile(`^[A-Z0-9]+(/[A-Z0-9]+)*$`)

// NormalizeCallsign trims, uppercases, and collapses a callsign for storage and
// lookup. The result is not guaranteed valid — use ValidCallsign to check.
func NormalizeCallsign(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// ValidCallsign reports whether s is a plausibly valid callsign after
// normalization.
func ValidCallsign(s string) bool {
	s = NormalizeCallsign(s)
	if len(s) < 3 || len(s) > 16 {
		return false
	}
	return callsignPattern.MatchString(s)
}

// Email reports whether s is a syntactically valid email address.
func Email(s string) bool {
	return validatorInstance.Var(strings.TrimSpace(s), "required,email") == nil
}

// idPattern bounds entity identifiers (UUIDs and the frontend's fallback ids):
// printable, conservative charset, length-capped.
var idPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// ValidID reports whether s is an acceptable entity identifier (e.g. a net or
// check-in id supplied by the client).
func ValidID(s string) bool {
	return idPattern.MatchString(s)
}

var datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// ValidDate reports whether s is a real YYYY-MM-DD calendar date.
func ValidDate(s string) bool {
	if !datePattern.MatchString(s) {
		return false
	}
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

// SanitizeLine normalizes a single-line free-text value: it drops control and
// zero-width characters, collapses internal whitespace to single spaces, ensures
// valid UTF-8, and trims. Use for names, nicknames, etc.
func SanitizeLine(s string) string {
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "")
	}
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for _, r := range s {
		switch {
		case r == '\u200b' || r == '\ufeff': // zero-width space / BOM
			continue
		case unicode.IsControl(r) || unicode.IsSpace(r):
			if prevSpace {
				continue
			}
			b.WriteByte(' ')
			prevSpace = true
		default:
			b.WriteRune(r)
			prevSpace = false
		}
	}
	return strings.TrimSpace(b.String())
}

// SanitizeMultiline normalizes a multi-line free-text value: it preserves
// newlines and tabs, drops other control characters, normalizes line endings,
// ensures valid UTF-8, and trims. Use for notes.
func SanitizeMultiline(s string) string {
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "")
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if r == '\u200b' || r == '\ufeff' || unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
	return strings.TrimSpace(s)
}

// Truncate rune-safely caps s to at most maxRunes runes.
func Truncate(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	r := []rune(s)
	return string(r[:maxRunes])
}

// CleanLine sanitizes and truncates a single-line value in one step.
func CleanLine(s string, maxRunes int) string {
	return Truncate(SanitizeLine(s), maxRunes)
}

// CleanMultiline sanitizes and truncates a multi-line value in one step.
func CleanMultiline(s string, maxRunes int) string {
	return Truncate(SanitizeMultiline(s), maxRunes)
}

// SafeURL returns s if it is a plausibly safe http(s) URL, otherwise "". It
// rejects javascript:, data:, and other schemes that could be dangerous if the
// value is ever rendered as a link.
func SafeURL(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return Truncate(s, 2048)
	}
	return ""
}
