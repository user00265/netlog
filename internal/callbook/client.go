// Package callbook looks up amateur callsigns against the QRZ XML and HamQTH
// callbook APIs (configurable primary + fallback) and caches the results.
//
// All providers share one HTTP client that logs every request at TRACE with
// credentials redacted, per the project logging convention.
package callbook

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"netlog/internal/logging"
)

// httpClient is the shared, credential-redacting HTTP client for callbook calls.
type httpClient struct {
	client *http.Client
	logger *slog.Logger
}

func newHTTPClient(client *http.Client, logger *slog.Logger) *httpClient {
	return &httpClient{client: client, logger: logger}
}

// get performs a GET and returns the response body. The request is logged at
// TRACE with sensitive query parameters redacted.
func (h *httpClient) get(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	logging.Trace(ctx, h.logger, "callbook request",
		slog.String("url", redactURL(rawURL)))

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("callbook request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20)) // cap at 4 MiB
	if err != nil {
		return nil, fmt.Errorf("read callbook response: %w", err)
	}
	logging.Trace(ctx, h.logger, "callbook response",
		slog.String("url", redactURL(rawURL)),
		slog.Int("status", resp.StatusCode),
		slog.Int("bytes", len(body)))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("callbook request: unexpected status %s", resp.Status)
	}
	return body, nil
}

// redactedParams are query keys whose values must never be logged.
var redactedParams = map[string]bool{
	"password": true, // QRZ
	"p":        true, // HamQTH password
	"username": true, // QRZ
	"u":        true, // HamQTH username
	"s":        true, // QRZ session key
	"id":       true, // HamQTH session id
}

// redactURL replaces sensitive query parameter values with "***" for logging.
func redactURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "[unparseable url]"
	}
	q := u.Query()
	for key := range q {
		if redactedParams[key] {
			q.Set(key, "***")
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}
