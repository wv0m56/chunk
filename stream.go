package chunk

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"hash"
	"io"
	"sync"
	"time"
)

var (
	errInputStreamStillRunning = errors.New("input stream still running")
)

// Sequence represents a sliced up io.Reader into a sequence of smaller chunks.
// It is thread safe.
type Sequence struct {
	c    chan *C
	w    int64     // read only
	h224 hash.Hash // accessed from 1 goroutine sequentially

	// r/w
	mu        sync.Mutex
	fin       bool
	err       error
	chunks224 []hash.Hash
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
		return Sum224{}, errInputStreamStillRunning
	}
	var res Sum224
	copy(res[:], s.h224.Sum(nil))
	return res, nil
}

// Metadata returns the metadata required to reconstruct the original file.
// Must only be called after the input stream has stopped, otherwise an error
// is returned.
func (s *Sequence) Metadata() (*Metadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.fin {
		return nil, errInputStreamStillRunning
	}

	m := &Metadata{}
	copy(m.TopChecksum[:], s.h224.Sum(nil))
	for _, v := range s.chunks224 {
		var tmp Sum224
		copy(tmp[:], v.Sum(nil))
		m.ChunkChecksums = append(m.ChunkChecksums, tmp)
	}
	m.Width = s.w

	return m, nil
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

// SplitStream cuts up an input stream rc into smaller chunks of width w bytes.
// bufSize specifies the length of the underlying buffered channel that holds
// the chunks as they are created.
// SplitStream immediately returns a *Sequence before it finishes reading
// from rc.
// The caller can then access the chunks as they arrive by iterating using Next.
// rc will be closed upon completion, with or without error.
// Check if the returned Sequence object is nil (invalid args) before proceeding,
// which will be the case if w<1, bufSize<0, rc==nil, or timeout<1ms.
func SplitStream(rc io.ReadCloser, w int64, bufSize int, timeout time.Duration) *Sequence {
	if w < 1 || bufSize < 0 || rc == nil || timeout.Nanoseconds() < 1000*1000 {
		return nil
	}

	br := bufio.NewReaderSize(rc, 1024*1024) // 1 MB buffer

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	s := &Sequence{
		make(chan *C, bufSize),
		w,
		sha256.New224(),
		sync.Mutex{},
		false,
		nil,
		nil,
	}

	go func() {
		defer func() {
			rc.Close()
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

				n, err := io.CopyN(mw, br, w)
				if err != nil {
					if err == io.EOF { // last chunk
						if n > 0 {
							s.c <- &C{chunk.Bytes(), h}
							s.chunks224 = append(s.chunks224, h)
						}
						s.doneWith(nil)
					} else {
						s.doneWith(err)
					}

					return
				}

				s.c <- &C{chunk.Bytes(), h}
				s.chunks224 = append(s.chunks224, h)
			}
		}
	}()

	return s
}
