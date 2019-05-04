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
	h224 hash.Hash

	mu  sync.Mutex // guards below
	fin bool
	err error
}

// Next returns the next data chunk if any.
// If s has finished consuming from the input io.Reader, the returned chunk is nil.
func (s *Sequence) Next() *C {
	return <-s.c
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

// Width returns the width of each chunk except the last.
func (s *Sequence) Width() int64 {
	return s.w
}

// Err returns any error encountered when processing the input stream
// if finished==true.
// If finished==false, err is undefined.
func (s *Sequence) Err() (finished bool, err error) {
	s.mu.Lock()
	finished = s.fin
	err = s.err
	s.mu.Unlock()
	return
}

func (s *Sequence) doneWith(err error) {
	s.mu.Lock()
	s.fin = true
	s.err = err
	s.mu.Unlock()
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
		sha256.New224(),
		sync.Mutex{},
		false,
		nil,
	}

	go func() {
		defer func() {
			close(s.c)
			cancel()
		}()

		for {
			select {

			case <-ctx.Done():
				s.doneWith(ctx.Err())
				return

			default:
				chunk := bytes.NewBuffer(nil)
				h := sha256.New224()
				mw := io.MultiWriter(s.h224, chunk, h)

				n, err := io.CopyN(mw, r, w)
				if err != nil {
					if err == io.EOF { // last chunk
						if n > 0 {
							s.c <- &C{chunk.Bytes(), h}
						}
						s.doneWith(nil)
					} else {
						s.doneWith(err)
					}

					return
				}

				s.c <- &C{chunk.Bytes(), h}
			}
		}
	}()

	return s
}
