package chunk

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

// Sum224 is SHA-224 as byte array.
type Sum224 [sha256.Size224]byte

// String returns the hex string representation of s.
func (s Sum224) String() string {
	return hex.EncodeToString(s[:])
}

// Eq returns whether s2 is equal to s.
func (s Sum224) Eq(s2 Sum224) bool {
	return bytes.Equal(s[:], s2[:])
}

// EqB returns whether b is equal to s.
func (s Sum224) EqB(b []byte) bool {
	return bytes.Equal(s[:], b)
}

// NewSum224 returns a new Sum224 from hex string.
func NewSum224(s string) (Sum224, error) {
	src := []byte(s)
	dst := make([]byte, hex.DecodedLen(len(src)))
	n, err := hex.Decode(dst, src)
	if err != nil {
		return Sum224{}, err
	}
	if n != 28 {
		return Sum224{}, errors.New("hex string not 224-bit")
	}
	var res Sum224
	copy(res[:], dst)
	return res, nil
}
