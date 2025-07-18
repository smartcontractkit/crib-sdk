package filehandler

import (
	"errors"
	"io"
	"io/fs"

	"github.com/smartcontractkit/crib-sdk/internal/adapter/mempools"
)

// Reader is a function that takes an io.Reader and returns a new io.Reader, potentially modifying
// the original reader.
type Reader func(io.Reader) io.Reader

// BufferedReader is a reader that re-uses buffer pools and implements the fs.ReadFileFS interface.
type BufferedReader struct {
	fs fs.FS
	r  Reader
}

// DefaultReader is a default reader that does not modify the original reader.
func DefaultReader(r io.Reader) io.Reader {
	return r
}

// LimitReader returns a new io.Reader that reads from the given io.Reader with a limit.
func LimitReader(limit int64) func(io.Reader) io.Reader {
	return func(r io.Reader) io.Reader {
		return io.LimitReader(r, limit)
	}
}

// NewBufferedReader creates a new BufferedReader with the given fs.FS and Reader.
func NewBufferedReader(f fs.FS, r Reader) *BufferedReader {
	return &BufferedReader{
		fs: f,
		r:  r,
	}
}

// Open opens a file at the given name relative to the BufferedReader's root directory.
func (b *BufferedReader) Open(name string) (fs.File, error) {
	return b.fs.Open(name)
}

// ReadFile reads a file at the given name relative to the BufferedReader's root directory.
func (b *BufferedReader) ReadFile(name string) (raw []byte, err error) {
	f, err := b.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Join(err, f.Close())
	}()

	buf, ret := mempools.BytesBuffer.Get()
	defer ret()

	if _, err := io.Copy(buf, b.r(f)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
