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

	err := br.Submit(cFromFile(t, "testdata/chunk2"), 2)
	assert.Nil(t, err)
	err = br.Submit(cFromFile(t, "testdata/chunk2"), 2)
	assert.NotNil(t, err)

	err = br.Close()
	assert.NotNil(t, err)
	assert.Equal(t, errUnprocessedChunksQueued, err)

	top224, err := br.Sum224()
	assert.Nil(t, err)
	assert.Equal(t, "d14a028c2a3a2bc9476102bb288234c415a2b01f828ea62ac5b3e42f", // sum of empty data
		top224.String())
}
