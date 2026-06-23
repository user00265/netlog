package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
)

// maxBodyBytes bounds request bodies to protect the server from oversized input.
const maxBodyBytes = 1 << 20 // 1 MiB

// writeJSON writes v as a JSON response with the given status.
func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		s.logger.Error("encode json response", slog.String("error", err.Error()))
	}
}

// errorBody is the standard error envelope.
type errorBody struct {
	Error string `json:"error"`
}

// writeError writes a JSON error envelope with the given status.
func (s *Server) writeError(w http.ResponseWriter, status int, msg string) {
	s.writeJSON(w, status, errorBody{Error: msg})
}

// decodeJSON strictly decodes a JSON request body into v.
func (s *Server) decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		if err == io.EOF {
			s.writeError(w, http.StatusBadRequest, "request body is empty")
		} else {
			s.writeError(w, http.StatusBadRequest, "invalid request body")
		}
		return false
	}
	// Reject anything after the first JSON value (e.g. a second smuggled object).
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		s.writeError(w, http.StatusBadRequest, "request body must contain a single JSON object")
		return false
	}
	return true
}
