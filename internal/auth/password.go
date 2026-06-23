package auth

import (
	"fmt"
	"runtime"

	"github.com/alexedwards/argon2id"
)

// passwordParams are the argon2id parameters used for all password hashing.
// Memory and iterations follow current OWASP/RFC 9106 guidance; parallelism
// tracks available CPUs (capped to keep the encoded value stable and modest).
var passwordParams = &argon2id.Params{
	Memory:      64 * 1024, // 64 MiB
	Iterations:  3,
	Parallelism: parallelism(),
	SaltLength:  16,
	KeyLength:   32,
}

func parallelism() uint8 {
	n := runtime.NumCPU()
	if n < 1 {
		n = 1
	}
	if n > 4 {
		n = 4
	}
	return uint8(n)
}

// HashPassword returns an argon2id encoded hash of the plaintext password.
func HashPassword(plaintext string) (string, error) {
	hash, err := argon2id.CreateHash(plaintext, passwordParams)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return hash, nil
}

// VerifyPassword reports whether plaintext matches the encoded argon2id hash.
func VerifyPassword(plaintext, encoded string) (bool, error) {
	if encoded == "" {
		return false, nil
	}
	match, _, err := argon2id.CheckHash(plaintext, encoded)
	if err != nil {
		return false, fmt.Errorf("verify password: %w", err)
	}
	return match, nil
}
