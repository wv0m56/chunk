package chunk

import (
	"crypto/sha256"
	"errors"
	"hash"
	"io"
	"sync"
)

// Reconstructor is the object that takes in chunks and streams out the
// reconstructed original whole data.
type Reconstructor struct {
	// read-only's
	checksumToIndex map[Sum224]int
	m               *Metadata

	mu          sync.Mutex // guards below
	w           io.Writer
	h224        hash.Hash
	chunksStash []*indexedC
}

func (rec *Reconstructor) Sink(c *C) error {
	if int64(len(c.b)) > rec.m.Width {
		return errors.New("chunk has incorrect width")
	}

	// TODO:

	//
	return nil
}

// Reconstruct returns a Reconstructor object based on the info in m.
// Every chunk sunk (in any order) into the returned Reconstructor will be
// written to w in order.
func Reconstruct(m *Metadata, w io.Writer) *Reconstructor {
	rec := &Reconstructor{
		make(map[Sum224]int),
		m,
		sync.Mutex{},
		w,
		sha256.New224(),
		[]*indexedC{},
	}

	for i, v := range m.ChunkChecksums {
		rec.checksumToIndex[v] = i
	}

	return rec
}
