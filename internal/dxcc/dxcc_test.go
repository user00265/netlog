package dxcc

import (
	"os"
	"testing"
)

func loadSample(t *testing.T) *DB {
	t.Helper()
	f, err := os.Open("testdata/cty_sample.xml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	db, err := Load(f)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return db
}

func TestLookup(t *testing.T) {
	db := loadSample(t)

	cases := []struct {
		call     string
		wantADIF int
		wantFlag string
		wantBy   string
	}{
		{"VE3ABC", 1, "ca", "prefix"},
		{"ve3abc", 1, "ca", "prefix"},  // case-insensitive
		{"K1ABC", 291, "us", "prefix"}, // longest-prefix
		{"W1AW", 291, "us", "prefix"},  // W1 more specific than W
		{"G3ABC", 223, "gb-eng", "prefix"},
		{"DL1XYZ", 230, "de", "prefix"},
		{"ZZ1Z", 1, "ca", "exception"},    // exact-call exception
		{"VE3ABC/P", 1, "ca", "prefix"},   // portable modifier stripped
		{"W1/G3ABC", 291, "us", "prefix"}, // operating location wins
		{"G3ABC/W1", 291, "us", "prefix"}, // either order
		{"3D2/C", 489, "fj", "prefix"},    // slash-containing prefix
	}
	for _, c := range cases {
		got, ok := db.Lookup(c.call)
		if !ok {
			t.Errorf("Lookup(%q): not found", c.call)
			continue
		}
		if got.ADIF != c.wantADIF {
			t.Errorf("Lookup(%q): adif = %d, want %d", c.call, got.ADIF, c.wantADIF)
		}
		if got.FlagISO2 != c.wantFlag {
			t.Errorf("Lookup(%q): flag = %q, want %q", c.call, got.FlagISO2, c.wantFlag)
		}
		if got.MatchedBy != c.wantBy {
			t.Errorf("Lookup(%q): matchedBy = %q, want %q", c.call, got.MatchedBy, c.wantBy)
		}
	}
}

func TestLookupUnknown(t *testing.T) {
	db := loadSample(t)
	if _, ok := db.Lookup("QQ9XYZ"); ok {
		t.Error("expected no match for unknown prefix")
	}
	if _, ok := db.Lookup(""); ok {
		t.Error("expected no match for empty callsign")
	}
}

func TestFlagMapSpotChecks(t *testing.T) {
	// A few well-known entities must resolve to the expected flag-icons code.
	cases := map[int]string{
		291: "us", 1: "ca", 230: "de", 339: "jp", 223: "gb-eng",
		279: "gb-sct", 294: "gb-wls", 265: "gb-nir", 522: "xk",
	}
	for adif, want := range cases {
		if got := flagFor(adif); got != want {
			t.Errorf("flagFor(%d) = %q, want %q", adif, got, want)
		}
	}
	// Non-geographic entity has no flag.
	if got := flagFor(289); got != "un" {
		t.Errorf("flagFor(UN HQ) = %q, want un", got)
	}
}
