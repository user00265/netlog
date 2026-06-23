package dxcc

import (
	"io"
	"strings"
	"time"

	"netlog/internal/validate"
)

// cleanContinent normalizes a continent code from cty.xml to a short uppercase
// token (NA, EU, …), dropping anything unexpected.
func cleanContinent(s string) string {
	return strings.ToUpper(validate.CleanLine(s, 4))
}

// Entity is a resolved DXCC entity.
type Entity struct {
	ADIF      int     `json:"adif"`
	Name      string  `json:"name"`
	Continent string  `json:"continent"`
	CQZone    int     `json:"cqZone"`
	ITUZone   int     `json:"ituZone"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Deleted   bool    `json:"deleted"`
	FlagISO2  string  `json:"flagIso2"`
}

// Result is the outcome of a callsign lookup.
type Result struct {
	Entity
	// MatchedBy describes how the entity was resolved: "exception" or "prefix".
	MatchedBy string `json:"matchedBy"`
	// Prefix is the matched prefix (empty for exceptions).
	Prefix string `json:"prefix"`
}

// DB is an immutable, in-memory lookup table built from cty.xml. It is safe for
// concurrent reads.
type DB struct {
	date         string
	entities     map[int]Entity
	exceptions   map[string]ctyRecord
	prefixes     map[string]ctyRecord
	maxPrefixLen int
}

// Date returns the source date of the loaded dataset.
func (db *DB) Date() string { return db.date }

// EntityCount returns the number of entities loaded.
func (db *DB) EntityCount() int { return len(db.entities) }

// Load builds a DB from a cty.xml reader.
func Load(r io.Reader) (*DB, error) {
	f, err := parse(r)
	if err != nil {
		return nil, err
	}
	db := &DB{
		date:       f.Date,
		entities:   make(map[int]Entity, len(f.Entities)),
		exceptions: make(map[string]ctyRecord, len(f.Exceptions)),
		prefixes:   make(map[string]ctyRecord, len(f.Prefixes)),
	}
	for _, e := range f.Entities {
		db.entities[e.ADIF] = Entity{
			ADIF:      e.ADIF,
			Name:      validate.CleanLine(e.Name, 80),
			Continent: cleanContinent(e.Continent),
			CQZone:    e.CQZone,
			ITUZone:   e.ITUZone,
			Latitude:  e.Latitude,
			Longitude: e.Longitude,
			Deleted:   e.Deleted,
			FlagISO2:  flagFor(e.ADIF),
		}
	}
	// cty.xml carries historical records (e.g. a deleted "DL" → old GERMANY
	// alongside the current "DL" → FED. REP. OF GERMANY). Only currently-valid
	// records should resolve a present-day callsign, so date-bounded expired or
	// not-yet-active records are skipped.
	for _, x := range f.Exceptions {
		if currentlyValid(x.Start, x.End) {
			db.exceptions[normalize(x.Call)] = x
		}
	}
	for _, p := range f.Prefixes {
		if !currentlyValid(p.Start, p.End) {
			continue
		}
		key := strings.ToUpper(p.Call)
		db.prefixes[key] = p
		if len(key) > db.maxPrefixLen {
			db.maxPrefixLen = len(key)
		}
	}
	return db, nil
}

// currentlyValid reports whether a record's optional [start, end] validity
// window includes the present moment. Empty bounds are open-ended.
func currentlyValid(start, end string) bool {
	now := time.Now()
	if end != "" {
		if t, err := time.Parse(time.RFC3339, end); err == nil && t.Before(now) {
			return false
		}
	}
	if start != "" {
		if t, err := time.Parse(time.RFC3339, start); err == nil && t.After(now) {
			return false
		}
	}
	return true
}

func normalize(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// modifiers are slash segments that indicate operating conditions, not a change
// of location/entity. Pure-numeric segments (call-area changes) are handled
// separately.
var modifiers = map[string]bool{
	"P": true, "M": true, "MM": true, "AM": true, "A": true, "R": true,
	"QRP": true, "QRPP": true, "LH": true, "LGT": true, "J": true, "Y": true,
	"BCN": true, "BEACON": true, "B": true,
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// baseCall reduces a callsign with portable indicators to the segment that best
// identifies the operating location. For "W1/G3ABC" or "G3ABC/W1" it returns
// the shorter location prefix; modifiers and call-area digits are dropped.
func (db *DB) baseCall(norm string) string {
	if !strings.Contains(norm, "/") {
		return norm
	}
	parts := strings.Split(norm, "/")
	var candidates []string
	for _, p := range parts {
		if p == "" || modifiers[p] || isNumeric(p) {
			continue
		}
		candidates = append(candidates, p)
	}
	switch len(candidates) {
	case 0:
		return parts[0]
	case 1:
		return candidates[0]
	default:
		// Prefer the shorter segment (location prefixes like W1/DL/VE3 are
		// shorter than home callsigns). Tie-break toward an actual prefix match.
		best := candidates[0]
		for _, c := range candidates[1:] {
			if len(c) < len(best) {
				best = c
			}
		}
		return best
	}
}

// longestPrefix returns the most specific prefix record matching s.
func (db *DB) longestPrefix(s string) (ctyRecord, string, bool) {
	maxLen := db.maxPrefixLen
	if len(s) < maxLen {
		maxLen = len(s)
	}
	for i := maxLen; i >= 1; i-- {
		if r, ok := db.prefixes[s[:i]]; ok {
			return r, s[:i], true
		}
	}
	return ctyRecord{}, "", false
}

// Lookup resolves a callsign to a DXCC entity. It checks exact-call exceptions
// first, then longest-prefix matching (handling portable indicators).
func (db *DB) Lookup(callsign string) (Result, bool) {
	norm := normalize(callsign)
	if norm == "" {
		return Result{}, false
	}

	// 1. Exact-call exception on the full callsign.
	if x, ok := db.exceptions[norm]; ok {
		return db.fromRecord(x, "exception", ""), true
	}

	// 2. A slash-containing prefix (e.g. "3D2/C" Conway Reef) must be matched
	//    against the full callsign before we split on "/".
	if r, key, ok := db.longestPrefix(norm); ok && strings.Contains(key, "/") {
		return db.fromRecord(r, "prefix", key), true
	}

	// 3. Reduce to the operating-location segment, then match.
	base := db.baseCall(norm)
	if x, ok := db.exceptions[base]; ok {
		return db.fromRecord(x, "exception", ""), true
	}
	if r, key, ok := db.longestPrefix(base); ok {
		return db.fromRecord(r, "prefix", key), true
	}
	// 4. Fall back to matching the full callsign.
	if r, key, ok := db.longestPrefix(norm); ok {
		return db.fromRecord(r, "prefix", key), true
	}
	return Result{}, false
}

// fromRecord builds a Result, enriching missing fields from the entity table.
func (db *DB) fromRecord(r ctyRecord, matchedBy, prefix string) Result {
	ent := db.entities[r.ADIF]
	// Record-level CQ zone / coordinates override the entity defaults.
	if r.CQZone != 0 {
		ent.CQZone = r.CQZone
	}
	if r.Continent != "" {
		ent.Continent = cleanContinent(r.Continent)
	}
	if r.Latitude != 0 || r.Longitude != 0 {
		ent.Latitude = r.Latitude
		ent.Longitude = r.Longitude
	}
	if ent.Name == "" {
		ent.Name = validate.CleanLine(r.Entity, 80)
	}
	if ent.ADIF == 0 {
		ent.ADIF = r.ADIF
		ent.FlagISO2 = flagFor(r.ADIF)
	}
	return Result{Entity: ent, MatchedBy: matchedBy, Prefix: prefix}
}
