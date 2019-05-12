package chunk

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBlindRecEarlyClose(t *testing.T) {
	out := noopCloseWriteCloser{bytes.NewBuffer(nil)}
	br := BlindReconstruct(out, 1*time.Second)

	err := br.Submit(cFromFile(t, "testdata/chunk2"), 1)
	assert.Nil(t, err)
	err = br.Submit(cFromFile(t, "testdata/chunk2"), 1)
	assert.NotNil(t, err)

	err = br.Close()
	assert.NotNil(t, err)
	assert.Equal(t, errUnprocessedChunksQueued, err)

	top224, err := br.Sum224()
	assert.Nil(t, err)
	assert.Equal(t, "d14a028c2a3a2bc9476102bb288234c415a2b01f828ea62ac5b3e42f", // sum of empty data
		top224.String())
}

func TestBlindReconstructor(t *testing.T) {
	out := noopCloseWriteCloser{bytes.NewBuffer(nil)}
	br := BlindReconstruct(out, 1*time.Second)

	fin, _ := br.Err()
	assert.False(t, fin)

	// 2
	err := br.Submit(cFromFile(t, "testdata/chunk2"), 1)
	assert.Nil(t, err)

	// 4
	err = br.Submit(cFromFile(t, "testdata/chunk4"), 3)
	assert.Nil(t, err)

	// 3
	err = br.Submit(cFromFile(t, "testdata/chunk3"), 2)
	assert.Nil(t, err)

	// 1
	err = br.Submit(cFromFile(t, "testdata/chunk1"), 0)
	assert.Nil(t, err)

	// 5
	err = br.Submit(cFromFile(t, "testdata/chunk5"), 4)
	assert.Nil(t, err)

	assert.Nil(t, br.Close())

	time.Sleep(200 * time.Millisecond)
	fin, err = br.Err()
	assert.Nil(t, err)
	assert.True(t, fin)

	top224, err := br.Sum224()
	assert.Nil(t, err)
	assert.Equal(t, "6bcc3cb34fce8aeddf37c797df54ea04fe8a35363904463050dbfd87",
		top224.String())
	assert.Equal(t, "Package bytes implements functions for the manipulation of byte slices. It is analogous to the facilities of the strings package.",
		out.String())
}
