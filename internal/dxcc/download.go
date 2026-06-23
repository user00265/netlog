package dxcc

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// SourceURL is the wavelog-mirrored, gzipped Clublog cty.xml.
const SourceURL = "https://raw.githubusercontent.com/wavelog/dxcc_data/refs/heads/master/cty.xml.gz"

// download fetches the gzipped cty.xml from url into a temp file alongside dest
// and returns the temp file path. It does NOT install it; the caller validates
// the download and renames it into place.
func download(ctx context.Context, client *http.Client, url, dest string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download cty.xml: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download cty.xml: unexpected status %s", resp.Status)
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dest), ".cty-*.tmp")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return "", fmt.Errorf("write cty.xml: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return "", err
	}
	// Return the temp path WITHOUT installing it. The caller validates the
	// download (parses it) before renaming over the existing good file, so a
	// corrupt-but-HTTP-200 response can never destroy the last-known-good dataset.
	return tmpName, nil
}

// loadFile opens a cty file (gzip-decompressing when the name ends in .gz) and
// builds a DB.
func loadFile(path string) (*DB, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var r io.Reader = f
	if strings.HasSuffix(path, ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return nil, fmt.Errorf("gunzip cty.xml: %w", err)
		}
		defer gz.Close()
		r = gz
	}
	return Load(r)
}
