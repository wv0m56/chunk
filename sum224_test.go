package chunk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSum224(t *testing.T) {
	s, err := NewSum224("6bcc3cb34fce8aeddf37c797df54ea04fe8a35363904463050dbfd87")
	assert.Nil(t, err)
	assert.Equal(t, "6bcc3cb34fce8aeddf37c797df54ea04fe8a35363904463050dbfd87", s.String())

	_, err = NewSum224("6bcc3cb34fce8aeddf37c797df54ea04fe8a35363904463050dbfd8777")
	assert.NotNil(t, err)
	assert.Equal(t, "hex string not 224-bit", err.Error())
}
