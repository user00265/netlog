package auth

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/google/uuid"
)

// NewID returns a random UUIDv4 string for use as an entity primary key.
func NewID() string {
	return uuid.NewString()
}

// NewToken returns a 256-bit cryptographically random, URL-safe token used as a
// session identifier.
func NewToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
