package chunk

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
)

type Sum224 [sha256.Size224]byte

func (s Sum224) String() string {
	return hex.EncodeToString(s[:])
}

func (s Sum224) Eq(s2 Sum224) bool {
	return bytes.Equal(s[:], s2[:])
}

func (s Sum224) EqB(b []byte) bool {
	return bytes.Equal(s[:], b)
}
