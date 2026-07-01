package httpapi

import (
	"io"
	"net/http"
	"strings"
)

// spaHandler serves the embedded SPA. Real static files are served directly with
// the file server; unknown non-API paths fall back to index.html so client-side
// routes (e.g. /nets/123) load the app. Unknown /api/ paths return a JSON 404.
func (s *Server) spaHandler() http.Handler {
	fileServer := http.FileServer(http.FS(s.spa))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			s.writeError(w, http.StatusNotFound, "not found")
			return
		}

		clean := strings.TrimPrefix(r.URL.Path, "/")
		if clean != "" {
			if f, err := s.spa.Open(clean); err == nil {
				info, statErr := f.Stat()
				_ = f.Close()
				if statErr == nil && !info.IsDir() {
					// Content-hashed build assets are immutable and cache forever;
					// every other file (index.html, sw.js, manifest, icons) must be
					// revalidated so a new deploy is picked up. sw.js especially:
					// without no-cache a proxy (e.g. Cloudflare) caches it by its
					// .js extension and the PWA never sees the update — leaving the
					// app stuck on the old bundle until a manual cache purge.
					if strings.HasPrefix(clean, "assets/") {
						w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
					} else {
						w.Header().Set("Cache-Control", "no-cache")
					}
					// Go's MIME table has no .webmanifest entry, so the file server
					// would mis-sniff it; set the correct type so browsers accept the
					// PWA manifest. (sw.js is fine: .js → text/javascript, a valid
					// service-worker MIME type.)
					if strings.HasSuffix(clean, ".webmanifest") {
						w.Header().Set("Content-Type", "application/manifest+json")
					}
					fileServer.ServeHTTP(w, r)
					return
				}
			}
		}
		s.serveIndex(w)
	})
}

// serveIndex writes index.html for SPA fallback routes with no-cache headers so
// clients always pick up the latest build.
func (s *Server) serveIndex(w http.ResponseWriter) {
	f, err := s.spa.Open("index.html")
	if err != nil {
		http.Error(w, "frontend not built", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	body, err := io.ReadAll(f)
	if err != nil {
		http.Error(w, "frontend not built", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
