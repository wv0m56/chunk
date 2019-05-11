package chunk

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// 6bcc3cb34fce8aeddf37c797df54ea04fe8a35363904463050dbfd87  all
// d0b4d664a97100ce9fd81a8ddd0051b80dfdbdcefb0d98a56231909d  chunk1
// 0a159b778546794379682eef59eb6cec6da039dc9222e4c65660f98e  chunk2
// 8203d16e9251e47c7ae59613d858a191f05b7b88efe2f37d0cae9eb5  chunk3
// 7e06a68ee69e1c1944045fe17d71b99bc0714ea930310ca0c323b096  chunk4
// fcbd8149fb4c6fcb49770ae28e5720e2f7e74e7bc60989829ccf68d6  chunk5

func TestReconstructor(t *testing.T) {
	c1sum, err := NewSum224("d0b4d664a97100ce9fd81a8ddd0051b80dfdbdcefb0d98a56231909d")
	assert.Nil(t, err)
	c2sum, err := NewSum224("0a159b778546794379682eef59eb6cec6da039dc9222e4c65660f98e")
	assert.Nil(t, err)
	c3sum, err := NewSum224("8203d16e9251e47c7ae59613d858a191f05b7b88efe2f37d0cae9eb5")
	assert.Nil(t, err)
	c4sum, err := NewSum224("7e06a68ee69e1c1944045fe17d71b99bc0714ea930310ca0c323b096")
	assert.Nil(t, err)
	c5sum, err := NewSum224("fcbd8149fb4c6fcb49770ae28e5720e2f7e74e7bc60989829ccf68d6")
	assert.Nil(t, err)

	out := noopCloseWriteCloser{bytes.NewBuffer(nil)}

	rec := Reconstruct(
		out,
		[]Sum224{c1sum, c2sum, c3sum, c4sum, c5sum},
		1000*time.Millisecond,
	)
	fin, _ := rec.Err()
	assert.False(t, fin)

	// out of order chunks
	// 2
	err = rec.Submit(cFromFile(t, "testdata/chunk2"))
	assert.Nil(t, err)

	// 4
	err = rec.Submit(cFromFile(t, "testdata/chunk4"))
	assert.Nil(t, err)

	// 3
	err = rec.Submit(cFromFile(t, "testdata/chunk3"))
	assert.Nil(t, err)

	// 1
	err = rec.Submit(cFromFile(t, "testdata/chunk1"))
	assert.Nil(t, err)

	// 5
	err = rec.Submit(cFromFile(t, "testdata/chunk5"))
	assert.Nil(t, err)

	time.Sleep(200 * time.Millisecond) // wait for mutexes, flushes to resolve
	fin, err = rec.Err()
	assert.Nil(t, err)
	assert.True(t, fin)

	top224, err := rec.Sum224()
	assert.Nil(t, err)
	assert.Equal(t, "6bcc3cb34fce8aeddf37c797df54ea04fe8a35363904463050dbfd87",
		top224.String())
	assert.Equal(t, "Package bytes implements functions for the manipulation of byte slices. It is analogous to the facilities of the strings package.",
		out.String())
}

// 478745e3d663ce49a06aa6a897f5369bc575f380a0a954459d48a517  repeated
// 773b42e98a8b235ccccaf49d7dd41943cfb57638ded6ab08aef19f52  repeated-chunk
func TestReconstructRepeatedData(t *testing.T) {
	chunkSum, err := NewSum224("773b42e98a8b235ccccaf49d7dd41943cfb57638ded6ab08aef19f52")
	assert.Nil(t, err)

	out := noopCloseWriteCloser{bytes.NewBuffer(nil)}
	rec := Reconstruct(
		out,
		[]Sum224{chunkSum, chunkSum, chunkSum},
		1000*time.Millisecond,
	)

	err = rec.Submit(cFromFile(t, "testdata/repeated-chunk"))
	assert.Nil(t, err)

	time.Sleep(100 * time.Millisecond)
	err = rec.Submit(cFromFile(t, "testdata/repeated-chunk"))
	assert.Nil(t, err)

	time.Sleep(200 * time.Millisecond) // wait for mutexes, flushes to resolve

	top224, err := rec.Sum224()
	assert.Nil(t, err)
	assert.Equal(t, "478745e3d663ce49a06aa6a897f5369bc575f380a0a954459d48a517",
		top224.String())
	assert.Equal(t, "This is repeated dataThis is repeated dataThis is repeated data",
		out.String())
}

func cFromFile(t *testing.T, path string) *C {
	f, err := os.Open(path)
	assert.Nil(t, err)
	defer f.Close()
	c, err := NewChunk(f)
	assert.Nil(t, err)
	return c
}

type noopCloseWriteCloser struct {
	*bytes.Buffer
}

func (ncwc noopCloseWriteCloser) Close() error {
	return nil
}
