package chunk

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"hash"
	"io"
	"sync"
	"time"
)

// Sequence represents a sliced up io.Reader into a sequence of smaller chunks.
type Sequence struct {
	c    chan *C
	w    int64
	err  chan error
	h224 hash.Hash

	mu  sync.Mutex // guards below
	fin bool
}

// Next returns the next data chunk if any.
// If s has finished consuming from the input io.Reader, then
// fin==true and err captures the error encountered when processing
// the input stream.
// If lastChunk==false, err is undefined.
// Calling Next again after the call that returns fin==true yields undefined
// return values.
func (s *Sequence) Next() (c *C, lastChunk bool, err error) {
	select {
	case c = <-s.c:
		return
	case err = <-s.err:
		lastChunk = true
		s.mu.Lock()
		s.fin = true
		s.mu.Unlock()
		return
	}
}

// Sum224 checks whether the processing of the input stream is finished.
// If it is ongoing, an error is returned.
// Otherwise, the SHA-224 checksum of the stream is returned with no error.
func (s *Sequence) Sum224() (Sum224, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.fin {
		return Sum224{}, errors.New("called Sum224 when input stream is still running")
	}
	var res Sum224
	copy(res[:], s.h224.Sum(nil))
	return res, nil
}

// ProcessStream cuts up an input stream r into smaller chunks of width w bytes.
// bufSize specifies the length of the underlying buffered channel that holds
// the chunks as they are created.
// ProcessStream immediately returns a *Sequence before it finishes reading
// from r.
// The caller can then access the chunks as they arrive by iterating using Next.
func ProcessStream(r io.Reader, w int64, bufSize int, timeout time.Duration) *Sequence {
	if w < 1 || bufSize < 0 || r == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	s := &Sequence{
		make(chan *C, bufSize),
		w,
		make(chan error),
		sha256.New224(),
		sync.Mutex{},
		false,
	}

	go func() {
		defer func() {
			close(s.c)
			cancel()
		}()

		for {
			select {

			case <-ctx.Done():
				s.err <- ctx.Err()

			default:
				chunk := bytes.NewBuffer(nil)
				h := sha256.New224()
				mw := io.MultiWriter(s.h224, chunk, h)

				_, err := io.CopyN(mw, r, w)
				if err != nil {

					// last chunk
					if err == io.EOF {
						s.c <- &C{chunk.Bytes(), h}
						s.err <- nil
					} else {
						s.err <- err
					}
				}
			}
		}
	}()

	return s
}
