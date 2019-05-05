package chunk

import (
	"bytes"
	"errors"
	"io"
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

	// check if finished and what error encountered
	fin, err := s.Err()
	assert.True(t, fin)
	assert.Nil(t, err)
}

func TestEdgeCases(t *testing.T) {
	f, err := os.Open("testdata/all")
	assert.Nil(t, err)
	defer f.Close()

	s := ProcessStream(f, 43, 0, 1*time.Second)
	assert.NotNil(t, s)

	fin, _ := s.Err()
	assert.False(t, fin)

	// 1
	c := s.Next()
	assert.Equal(t, sha224bin("testdata/3chunk-1"), c.Sum224().String())

	// 2
	c = s.Next()
	assert.Equal(t, sha224bin("testdata/3chunk-2"), c.Sum224().String())

	// 3
	c = s.Next()
	assert.Equal(t, sha224bin("testdata/3chunk-3"), c.Sum224().String())

	assert.Nil(t, s.Next())
	fin, err = s.Err()
	assert.True(t, fin)
	assert.Nil(t, err)

	sum224, err := s.Sum224()
	assert.Nil(t, err)
	assert.Equal(t, sha224bin("testdata/all"), sum224.String())
}

func TestTimeout(t *testing.T) {
	pr, _ := dummyPipe()
	s := ProcessStream(pr, 100, 100, 500*time.Millisecond)

	fin, _ := s.Err()
	assert.False(t, fin)

	for {
		if c := s.Next(); c == nil {
			break
		}
	}

	fin, err := s.Err()
	assert.True(t, fin)
	assert.Equal(t, "context deadline exceeded", err.Error())
}

func TestStreamError(t *testing.T) {
	pr, pw := dummyPipe()
	s := ProcessStream(pr, 10, 100, 100*time.Second)

	fin, _ := s.Err()
	assert.False(t, fin)

	go func() {
		time.Sleep(50 * time.Millisecond)
		pw.CloseWithError(errors.New("some error"))
	}()

	for {
		if c := s.Next(); c == nil {
			break
		}
	}

	fin, err := s.Err()
	assert.True(t, fin)
	assert.Equal(t, "some error", err.Error())
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

func dummyPipe() (*io.PipeReader, *io.PipeWriter) {
	pr, pw := io.Pipe()

	// simulate a slow, endless stream
	go func() {
		for {
			time.Sleep(10 * time.Millisecond)
			pw.Write([]byte("beep beep"))
		}
	}()

	return pr, pw
}
