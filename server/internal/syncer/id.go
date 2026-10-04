package syncer

import (
	"crypto/rand"
	"encoding/binary"
	"time"
)

const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// NewID returns a ULID: 48 bits of milliseconds, then 80 random bits, in
// Crockford base32. Clients generate the same kind of ID.
func NewID(now time.Time) string {
	var b [16]byte
	binary.BigEndian.PutUint64(b[:8], uint64(now.UnixMilli())<<16)
	_, _ = rand.Read(b[6:]) // crypto/rand.Read never fails

	hi := binary.BigEndian.Uint64(b[:8])
	lo := binary.BigEndian.Uint64(b[8:])
	var out [26]byte
	// 26 characters hold 130 bits; the top 2 are always zero.
	for i := 25; i >= 0; i-- {
		out[i] = crockford[lo&31]
		lo = lo>>5 | hi<<59
		hi >>= 5
	}
	return string(out[:])
}

// ValidID reports whether id is a ULID in canonical upper-case form.
func ValidID(id string) bool {
	if len(id) != 26 || id[0] > '7' {
		return false
	}
	for i := 0; i < len(id); i++ {
		if !isCrockford(id[i]) {
			return false
		}
	}
	return true
}

func isCrockford(c byte) bool {
	for i := 0; i < len(crockford); i++ {
		if crockford[i] == c {
			return true
		}
	}
	return false
}
