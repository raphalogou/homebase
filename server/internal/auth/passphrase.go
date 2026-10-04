// Package auth checks the passphrase, limits login attempts and manages
// sessions.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

// Hash is a parsed argon2id hash in PHC string form:
// $argon2id$v=19$m=65536,t=3,p=4$<salt>$<key>
type Hash struct {
	memory  uint32 // KiB
	time    uint32
	threads uint8
	salt    []byte
	key     []byte
}

// Parameters for new hashes: 64 MiB, 3 passes, 4 lanes, above the OWASP
// minimum while still taking well under a second on a small server.
const (
	newMemory  = 64 * 1024
	newTime    = 3
	newThreads = 4
	saltLen    = 16
	keyLen     = 32

	// Limits on parsed hashes, so a mistyped variable cannot make each
	// login allocate gigabytes.
	maxMemory = 1024 * 1024
	maxTime   = 20
)

// MinPassphrase and MaxPassphrase bound a new passphrase, counted in
// characters. There are no composition rules: length is what makes it strong.
const (
	MinPassphrase = 12
	MaxPassphrase = 1000
)

// CheckNewPassphrase says why a passphrase cannot be set, in the words the
// forms show under the field.
func CheckNewPassphrase(passphrase string) error {
	n := utf8.RuneCountInString(passphrase)
	switch {
	case n < MinPassphrase:
		return fmt.Errorf("Use at least %d characters. You have %d.", MinPassphrase, n)
	case n > MaxPassphrase:
		return fmt.Errorf("Use at most %d characters.", MaxPassphrase)
	}
	return nil
}

// HashPassphrase returns the PHC string for a new passphrase.
func HashPassphrase(passphrase string) (string, error) {
	if err := CheckNewPassphrase(passphrase); err != nil {
		return "", err
	}
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(passphrase), salt, newTime, newMemory, newThreads, keyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, newMemory, newTime, newThreads,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

// ParseHash reads a PHC string produced by HashPassphrase.
func ParseHash(s string) (Hash, error) {
	bad := errors.New("not an argon2id hash; create one with: homebase hash-passphrase")
	parts := strings.Split(s, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return Hash{}, bad
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return Hash{}, fmt.Errorf("unsupported argon2 version %q", parts[2])
	}
	var h Hash
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &h.memory, &h.time, &h.threads); err != nil {
		return Hash{}, bad
	}
	if h.memory < 8 || h.memory > maxMemory || h.time < 1 || h.time > maxTime || h.threads < 1 {
		return Hash{}, fmt.Errorf("argon2 parameters out of range: %s", parts[3])
	}
	var err error
	if h.salt, err = base64.RawStdEncoding.DecodeString(parts[4]); err != nil || len(h.salt) < 8 {
		return Hash{}, bad
	}
	if h.key, err = base64.RawStdEncoding.DecodeString(parts[5]); err != nil || len(h.key) < 16 {
		return Hash{}, bad
	}
	return h, nil
}

// Matches reports whether passphrase produces this hash, in constant time.
func (h Hash) Matches(passphrase string) bool {
	key := argon2.IDKey([]byte(passphrase), h.salt, h.time, h.memory, h.threads, uint32(len(h.key)))
	return subtle.ConstantTimeCompare(key, h.key) == 1
}
