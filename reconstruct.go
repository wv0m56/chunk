package chunk

import (
	"bufio"
	"context"
	"crypto/sha256"
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
	// read-only's
	checksumToIndexes map[Sum224][]int
	chunkHashes       []Sum224

	// r/w, mutex inside embed
	submittedChunks map[Sum224]struct{}

	reconstructor
}

// Sum224 checks whether the streaming of chunks to the output stream is finished.
// If it is ongoing, an error is returned.
// Otherwise, the SHA-224 checksum of the stream is returned with no error.
func (rec *Reconstructor) Sum224() (Sum224, error) {
	return rec.sum224()
}

// Err returns any error encountered when writing to the output stream
// if finished==true.
// If finished==false, err is undefined.
func (rec *Reconstructor) Err() (finished bool, err error) {
	return rec.finErr()
}

// Submit sinks chunk c (in any order) for the purpose of reconstructing the
// original file.
// All chunks will be reordered by rec.
// Submiting the same chunk more than once does nothing.
func (rec *Reconstructor) Submit(c *C) error {
	chunkHashRef := c.Sum224()

	idxs, ok := rec.checksumToIndexes[chunkHashRef]
	if !ok {
		return errNoChunkInMetadata
	}

	rec.mu.Lock()

	if _, ok := rec.submittedChunks[chunkHashRef]; ok {
		rec.mu.Unlock()
		return nil
	}

	if rec.fin {
		rec.mu.Unlock()
		return errFinishedReconstructor
	}

	rec.submittedChunks[chunkHashRef] = struct{}{}

	var ics []*indexedC
	for _, v := range idxs {
		ics = append(ics, &indexedC{C{c.b, c.h224}, v})
	}
	rec.sorter = append(rec.sorter, ics...)
	sort.Sort(rec.sorter)

	rec.mu.Unlock()

	rec.lastReceivedIndex <- idxs[0]

	if idxs[len(idxs)-1]+1 == len(rec.chunkHashes) {
		close(rec.lastReceivedIndex)
	}

	return nil
}

// Reconstruct returns a Reconstructor object based on the info in m.
// Every chunk sunk (in any order) into the returned Reconstructor will be
// written to w in order.
// The returned Reconstructor is nil if chunkHashes has 0 length.
func Reconstruct(wc io.WriteCloser, chunkHashes []Sum224, timeout time.Duration) *Reconstructor {
	if len(chunkHashes) < 1 {
		return nil
	}

	rec := &Reconstructor{
		make(map[Sum224][]int),
		chunkHashes,
		make(map[Sum224]struct{}),
		reconstructor{
			make(chan int),
			sync.Mutex{},
			sha256.New224(),
			[]*indexedC{},
			false,
			nil,
		},
	}

	for i, v := range chunkHashes {
		rec.checksumToIndexes[v] = append(rec.checksumToIndexes[v], i)
	}
	for _, v := range rec.checksumToIndexes {
		if len(v) > 1 {
			sort.Ints(v)
		}
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

			case <-ctx.Done():
				rec.doneWith(ctx.Err())
				return

			case i := <-rec.lastReceivedIndex:
				if nextIndex == len(rec.chunkHashes) {
					rec.doneWith(nil)
					return
				}

				if nextIndex == i {

					rec.mu.Lock()
					for len(rec.sorter) > 0 && nextIndex == rec.sorter[len(rec.sorter)-1].idx {
						err := rec.writeChunk(bw)
						if err != nil {
							rec.err = err
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

// assume external lock
func pop(s byReverseIndex) (*indexedC, byReverseIndex) {
	if len(s) == 0 {
		return nil, s
	}
	c := s[len(s)-1]
	s = s[:len(s)-1]
	return c, s
}

type reconstructor struct {
	lastReceivedIndex chan int
	mu                sync.Mutex
	h224              hash.Hash
	sorter            byReverseIndex
	fin               bool
	err               error
}

func (rec *reconstructor) doneWith(err error) {
	rec.mu.Lock()
	rec.fin = true
	rec.err = err
	rec.mu.Unlock()
	return
}

func (rec *reconstructor) finErr() (finished bool, err error) {
	rec.mu.Lock()
	finished = rec.fin
	err = rec.err
	rec.mu.Unlock()
	return
}

func (rec *reconstructor) sum224() (Sum224, error) {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if !rec.fin {
		return Sum224{}, errStreamStillRunning
	}
	var res Sum224
	copy(res[:], rec.h224.Sum(nil))
	return res, nil
}

// assume external lock
func (rec *reconstructor) writeChunk(w io.Writer) error {
	var c *indexedC
	c, rec.sorter = pop(rec.sorter)
	h := sha256.New224()
	mw := io.MultiWriter(w, h, rec.h224)
	_, err := io.Copy(mw, c.Reader())
	if err != nil {
		return err
	}

	// hash check
	if !c.IsHash(h.Sum(nil)) {
		return err
	}

	return nil
}
