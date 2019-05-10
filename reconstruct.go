package chunk

import (
	"bufio"
	"context"
	"crypto/sha256"
	"errors"
	"hash"
	"io"
	"sort"
	"sync"
	"time"
)

// Reconstructor is the object that takes in chunks and streams out the
// reconstructed original whole data.
// It is thread safe.
type Reconstructor struct {
	lastReceivedIndex chan int

	// read-only's
	checksumToIndex map[Sum224]int
	m               *Metadata
	h224            hash.Hash // accessed from 1 goroutine squentially

	// r/w
	mu                    sync.Mutex
	sorter                byReverseIndex
	submittedChunkIndexes map[int]struct{}
	fin                   bool
	err                   error
}

// Submit sinks chunk c (in any order) for the purpose of reconstructing the
// original file.
// All chunks will be reordered by rec.
// Submiting the same chunk more than once will yield an error.
func (rec *Reconstructor) Submit(c *C) error {
	if int64(len(c.b)) > rec.m.Width {
		return errors.New("chunk has incorrect width")
	}

	chunkHashRef := c.Sum224()
	i, ok := rec.checksumToIndex[chunkHashRef]
	if !ok {
		return errNoChunkInMetadata
	}

	rec.mu.Lock()

	if _, ok = rec.submittedChunkIndexes[i]; ok {
		rec.mu.Unlock()
		return errResubmitSameChunk
	}
	ic := &indexedC{C{c.b, c.h224}, i}

	if rec.fin {
		rec.mu.Unlock()
		return errFinishedReconstructor
	}

	rec.sorter = append(rec.sorter, ic)
	sort.Sort(rec.sorter)
	rec.submittedChunkIndexes[i] = struct{}{}

	rec.mu.Unlock()

	rec.lastReceivedIndex <- i
	if i+1 == len(rec.m.ChunkChecksums) {
		close(rec.lastReceivedIndex)
	}

	return nil
}

// Sum224 checks whether the streaming of chunks to the output stream is finished.
// If it is ongoing, an error is returned.
// Otherwise, the SHA-224 checksum of the stream is returned with no error.
func (rec *Reconstructor) Sum224() (Sum224, error) {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if !rec.fin {
		return Sum224{}, errStreamStillRunning
	}
	var res Sum224
	copy(res[:], rec.h224.Sum(nil))
	return res, nil
}

// Err returns any error encountered when writing to the output stream
// if finished==true.
// If finished==false, err is undefined.
func (rec *Reconstructor) Err() (finished bool, err error) {
	rec.mu.Lock()
	finished = rec.fin
	err = rec.err
	rec.mu.Unlock()
	return
}

// Reconstruct returns a Reconstructor object based on the info in m.
// Every chunk sunk (in any order) into the returned Reconstructor will be
// written to w in order.
// Non-sensical arg yields nil returned Reconstructor.
func Reconstruct(wc io.WriteCloser, m *Metadata, timeout time.Duration) *Reconstructor {
	if len(m.ChunkChecksums) < 1 {
		return nil
	}
	if m.TopChecksum == (Sum224{}) {
		return nil
	}

	rec := &Reconstructor{
		make(chan int),
		make(map[Sum224]int),
		m,
		sha256.New224(),
		sync.Mutex{},
		[]*indexedC{},
		make(map[int]struct{}),
		false,
		nil,
	}

	for i, v := range m.ChunkChecksums {
		rec.checksumToIndex[v] = i
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	go func() {
		bw := bufio.NewWriterSize(wc, writeBufferSize)
		defer func() {
			rec.doneWith(bw.Flush())
			wc.Close()
			cancel()
		}()

		nextIndex := 0
		for {

			select {

			case <-ctx.Done():
				rec.doneWith(ctx.Err())
				return

			case i := <-rec.lastReceivedIndex:
				if nextIndex == len(rec.m.ChunkChecksums) {
					return
				}

				if nextIndex == i {

					rec.mu.Lock()
					for len(rec.sorter) > 0 && nextIndex == rec.sorter[len(rec.sorter)-1].idx {
						var c *indexedC
						c, rec.sorter = pop(rec.sorter)
						h := sha256.New224()
						mw := io.MultiWriter(bw, h, rec.h224)
						_, err := io.Copy(mw, c.Reader())
						if err != nil {
							rec.err = err
							rec.fin = true
							rec.mu.Unlock()
							return
						}

						// hash check
						if !c.IsHash(h.Sum(nil)) {
							rec.err = errChunkChecksum
							rec.fin = true
							rec.mu.Unlock()
							return
						}

						nextIndex++
					}
					rec.mu.Unlock()
				}
			}
		}
	}()

	return rec
}

// assume external locking
func pop(s byReverseIndex) (*indexedC, byReverseIndex) {
	if len(s) == 0 {
		return nil, s
	}
	c := s[len(s)-1]
	s = s[:len(s)-1]
	return c, s
}

func (rec *Reconstructor) doneWith(err error) {
	rec.mu.Lock()
	rec.fin = true
	rec.err = err
	rec.mu.Unlock()
	return
}
