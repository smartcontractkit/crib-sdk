package mempools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBytesBuffer(t *testing.T) {
	// Do not use t.Parallel() here as it uses a global pool.

	const str = "Hello, World!"
	is := assert.New(t)

	// Pull a fresh buffer from the pool.
	buf, ret := BytesBuffer.Get()
	// Assert emptiness.
	is.Equal(buf.Cap(), 0)
	is.Equal(buf.Len(), 0)
	is.Equal(buf.Available(), 0)

	// Write a string to the buffer and assert the length.
	w, err := buf.WriteString(str)
	is.NoError(err)
	is.Equal(w, len(str))
	is.Equal(buf.Len(), len(str))
	is.GreaterOrEqual(buf.Len(), len(str))
	is.Equal(buf.Available(), buf.Cap()-buf.Len())

	// Return the buffer to the pool.
	ret()

	// Pull another buffer from the pool.
	buf2, ret2 := BytesBuffer.Get()
	// Assert that the buffer is allocated but empty.
	is.Equal(buf2.Len(), 0)
	is.GreaterOrEqual(buf2.Cap(), 0)
	is.Equal(buf2.Available(), buf2.Cap())

	// Return the buffer to the pool.
	ret2()
}
