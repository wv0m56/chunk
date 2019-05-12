package chunk

import (
	"bufio"
	"context"
	"crypto/sha256"
	"io"
	"sort"
	"sync"
	"time"
)

// BlindReconstructor is similar in many ways to Reconstructor except that it
// has no knowledge of the original file's metadata and needs to be closed
// manually by the caller.
// Like Reconstructor, it is thread safe.
type BlindReconstructor struct {
	closed chan struct{}

	// r/w, mutex inside embed
	submittedIndexes map[int]struct{}

	reconstructor
}

// Sum224 checks whether the streaming of chunks to the output stream is finished.
// If it is ongoing, an error is returned.
// Otherwise, the SHA-224 checksum of the stream is returned with no error.
func (br *BlindReconstructor) Sum224() (Sum224, error) {
	return br.sum224()
}

// Err returns any error encountered when writing to the output stream
// if finished==true.
// If finished==false, err is undefined.
func (br *BlindReconstructor) Err() (finished bool, err error) {
	return br.finErr()
}

// Submit sinks chunk c (in any order) for the purpose of reconstructing the
// original file.
// All chunks will be reordered by br.
// Submitting the same idx more than once will yield an error but does not
// change br's state.
// Submitting the same chunk under a different idx is OK.
func (br *BlindReconstructor) Submit(c *C, idx int) error {
	br.mu.Lock()

	if _, ok := br.submittedIndexes[idx]; ok {
		br.mu.Unlock()
		return errResubmitSameIndex
	}

	if br.fin {
		br.mu.Unlock()
		return errFinishedReconstructor
	}

	br.submittedIndexes[idx] = struct{}{}

	br.sorter = append(br.sorter, &indexedC{C{c.b, c.h224}, idx})
	sort.Sort(br.sorter)

	br.mu.Unlock()

	br.lastReceivedIndex <- idx

	return nil
}

// Close closes and cleans up after br. Close signals the underlying writer to
// be closed, but does not wait until it happens.
func (br *BlindReconstructor) Close() error {
	br.closed <- struct{}{}
	br.mu.Lock()
	defer br.mu.Unlock()
	if len(br.sorter) > 0 {
		br.err = errUnprocessedChunksQueued
	}
	br.fin = true
	return br.err
}

// BlindReconstruct is similar to Reconstruct() without the aid of metadata
// information.
// Every chunk sunk (in any order) into the returned Reconstructor will be
// written to w in order, but the caller must specify the chunk's index.
func BlindReconstruct(wc io.WriteCloser, timeout time.Duration) *BlindReconstructor {

	br := &BlindReconstructor{
		make(chan struct{}),
		make(map[int]struct{}),
		reconstructor{
			make(chan int),
			sync.Mutex{},
			sha256.New224(),
			[]*indexedC{},
			false,
			nil,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	go func() {
		bw := bufio.NewWriterSize(wc, writeBufferSize)
		defer func() {
			bw.Flush()
			wc.Close()
			cancel()
		}()

		nextIndex := 0
		for {

			select {

			case <-br.closed:
				return

			case <-ctx.Done():
				br.doneWith(ctx.Err())
				return

			case i := <-br.lastReceivedIndex:

				if nextIndex == i {

					br.mu.Lock()
					for len(br.sorter) > 0 && nextIndex == br.sorter[len(br.sorter)-1].idx {
						err := br.writeChunk(bw)
						if err != nil {
							br.err = err
							br.fin = true
							br.mu.Unlock()
							return
						}
						nextIndex++
					}
					br.mu.Unlock()
				}
			}
		}
	}()

	return br
}
