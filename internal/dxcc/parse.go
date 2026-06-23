// Package dxcc resolves amateur callsigns to DXCC entities (and a country flag)
// using the Clublog cty.xml dataset mirrored by the wavelog project.
package dxcc

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
)

// ctyFile mirrors the Clublog cty.xml document structure.
type ctyFile struct {
	Date           string       `xml:"date,attr"`
	Entities       []ctyEntity  `xml:"entities>entity"`
	Exceptions     []ctyRecord  `xml:"exceptions>exception"`
	Prefixes       []ctyRecord  `xml:"prefixes>prefix"`
	ZoneExceptions []ctyZoneExc `xml:"zone_exceptions>zone_exception"`
}

// ctyEntity is a DXCC entity. ITUZ and End are optional.
type ctyEntity struct {
	ADIF      int     `xml:"adif"`
	Name      string  `xml:"name"`
	Prefix    string  `xml:"prefix"`
	Deleted   bool    `xml:"deleted"`
	CQZone    int     `xml:"cqz"`
	ITUZone   int     `xml:"ituz"`
	Continent string  `xml:"cont"`
	Longitude float64 `xml:"long"`
	Latitude  float64 `xml:"lat"`
	End       string  `xml:"end"`
}

// ctyRecord is a prefix or exact-call exception record.
type ctyRecord struct {
	Call      string  `xml:"call"`
	Entity    string  `xml:"entity"`
	ADIF      int     `xml:"adif"`
	CQZone    int     `xml:"cqz"`
	Continent string  `xml:"cont"`
	Longitude float64 `xml:"long"`
	Latitude  float64 `xml:"lat"`
	Start     string  `xml:"start"`
	End       string  `xml:"end"`
}

// ctyZoneExc overrides the CQ zone for an exact callsign.
type ctyZoneExc struct {
	Call  string `xml:"call"`
	Zone  int    `xml:"zone"`
	Start string `xml:"start"`
	End   string `xml:"end"`
}

// parse decodes a cty.xml document from r.
func parse(r io.Reader) (*ctyFile, error) {
	var f ctyFile
	dec := xml.NewDecoder(r)
	if err := dec.Decode(&f); err != nil {
		return nil, fmt.Errorf("decode cty.xml: %w", err)
	}
	if len(f.Entities) == 0 {
		return nil, errors.New("cty.xml contained no entities")
	}
	return &f, nil
}
