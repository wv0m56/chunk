package chunk

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStream(t *testing.T) {
	_, err := exec.LookPath("sha224sum")
	assert.Nil(t, err, `must have "sha224sum" executable in path`)

	f, err := os.Open("testdata/all")
	assert.Nil(t, err)
	defer f.Close()

	s := ProcessStream(f, 30, 2, 1*time.Second)
	assert.NotNil(t, s)

	// 1
	c := s.Next()
	assert.Equal(t, sha224bin("testdata/chunk1"), c.Sum224().String())

	// 2
	c = s.Next()
	assert.Equal(t, sha224bin("testdata/chunk2"), c.Sum224().String())

	// check top hash mid stream
	_, err = s.Sum224()
	assert.NotNil(t, err)

	// 3
	c = s.Next()
	assert.Equal(t, sha224bin("testdata/chunk3"), c.Sum224().String())

	// because the buffer size is 2, it *very likely* that the 2 final chunks have
	// been written into the hashes even before we retrieve the chunks.
	sum224, err := s.Sum224()
	assert.Nil(t, err)
	assert.Equal(t, sha224bin("testdata/all"), sum224.String())

	// 4
	c = s.Next()
	assert.Equal(t, sha224bin("testdata/chunk4"), c.Sum224().String())

	// 5
	c = s.Next()
	assert.Equal(t, sha224bin("testdata/chunk5"), c.Sum224().String())

	// 6
	assert.Nil(t, s.Next())

	// top hash still the same
	sum224, err = s.Sum224()
	assert.Nil(t, err)
	assert.Equal(t, sha224bin("testdata/all"), sum224.String())
}

func sha224bin(path string) string {
	cmd := exec.Command("sha224sum", path)
	buf := bytes.NewBuffer(nil)
	cmd.Stdout = buf
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
	return strings.Split(string(buf.Bytes()), "  ")[0]
}
