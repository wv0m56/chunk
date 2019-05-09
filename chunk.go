package chunk

import (
	"bytes"
	"crypto/sha256"
	"hash"
	"io"
	"io/ioutil"
)

// C (for chunk) represents a fraction of the data resulting from slicing up
// an input stream.
type C struct {
	b    []byte
	h224 hash.Hash
}

// Reader returns a read-only view of the underlying []byte stored in c.
func (c *C) Reader() *bytes.Reader {
	return bytes.NewReader(c.b)
}

// IsHash returns true if SHA224 hash h is the same as the hash recorded
// during the creation of c.
func (c *C) IsHash(h []byte) bool {
	return bytes.Equal(h, c.h224.Sum(nil))
}

// Sum224 returns the SHA-224 checksum of c.
func (c *C) Sum224() Sum224 {
	var res Sum224
	copy(res[:], c.h224.Sum(nil))
	return res
}

// NewChunk consumes r and creates a new C object of the consumed/buffered data.
// Once created, the chunk is read-only.
func NewChunk(r io.Reader) (*C, error) {
	h := sha256.New224()
	tr := io.TeeReader(r, h)

	b, err := ioutil.ReadAll(tr)
	if err != nil {
		return nil, err
	}
	return &C{b, h}, nil
}
