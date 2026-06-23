package validate

import (
	"strings"
	"testing"
)

func TestValidCallsign(t *testing.T) {
	good := []string{"W1AW", "kf0acn", "VE3ABC/P", "3D2/C"}
	bad := []string{"", "ab", "W1AW!", "<script>", "W1 AW"}
	for _, c := range good {
		if !ValidCallsign(c) {
			t.Errorf("expected %q valid", c)
		}
	}
	for _, c := range bad {
		if ValidCallsign(c) {
			t.Errorf("expected %q invalid", c)
		}
	}
}

func TestValidID(t *testing.T) {
	good := []string{"11111111-1111-1111-1111-111111111111", "id-abc-def", "abc123"}
	bad := []string{"", "../etc", "a b", "drop;table", strings.Repeat("x", 65)}
	for _, c := range good {
		if !ValidID(c) {
			t.Errorf("expected %q valid id", c)
		}
	}
	for _, c := range bad {
		if ValidID(c) {
			t.Errorf("expected %q invalid id", c)
		}
	}
}

func TestValidDate(t *testing.T) {
	if !ValidDate("2026-06-23") {
		t.Error("expected valid date")
	}
	for _, d := range []string{"2026-13-01", "2026-02-30", "not-a-date", "2026/06/23", ""} {
		if ValidDate(d) {
			t.Errorf("expected %q invalid date", d)
		}
	}
}

func TestSanitizeLine(t *testing.T) {
	cases := []struct{ in, want string }{
		{"  Hello   World  ", "Hello World"},
		{"tab\tsep", "tab sep"},
		{"zero\u200bwidth", "zerowidth"},
		{"new\nline", "new line"},
		{"\ufeffBOM", "BOM"},
		{"normal", "normal"},
	}
	for _, c := range cases {
		if got := SanitizeLine(c.in); got != c.want {
			t.Errorf("SanitizeLine(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSanitizeMultiline(t *testing.T) {
	got := SanitizeMultiline("line1\r\nline2\n\tindented")
	want := "line1\nline2\n\tindented"
	if got != want {
		t.Errorf("SanitizeMultiline = %q, want %q", got, want)
	}
}

func TestTruncate(t *testing.T) {
	if got := Truncate("hello", 3); got != "hel" {
		t.Errorf("Truncate = %q", got)
	}
	if got := Truncate("héllo", 2); got != "hé" {
		t.Errorf("Truncate multibyte = %q", got)
	}
	if got := Truncate("hi", 10); got != "hi" {
		t.Errorf("Truncate short = %q", got)
	}
}

func TestSafeURL(t *testing.T) {
	if SafeURL("https://example.com") != "https://example.com" {
		t.Error("expected https URL kept")
	}
	for _, u := range []string{"javascript:alert(1)", "data:text/html,x", "ftp://x", "  ", "example.com"} {
		if SafeURL(u) != "" {
			t.Errorf("expected %q rejected", u)
		}
	}
}
