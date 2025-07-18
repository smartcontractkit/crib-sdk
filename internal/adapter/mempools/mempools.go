// Package mempools provides methods to interact with shared allocated memory pools.
//
// Example usage:
//
//	buf, ret := mempools.BytesBuffer.Get()
//	defer ret() // Return the buffer to the pool when done.
//	buf.WriteString("Hello, World!")
package mempools

import (
	"bytes"
	"sync"

	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

// BytesBuffer is a shared global pool of bytes.Buffer objects.
// Calls are goroutine-safe and the pool is shared across all packages.
// The buffer _must_ not be used after putting it back in the pool.
var BytesBuffer = NewPool(func() *bytes.Buffer {
	return new(bytes.Buffer)
})

type resettable interface {
	Reset()
}

type Pool[T resettable] struct {
	sync.Pool
}

func (p *Pool[T]) Get() (t T, closeFn func()) {
	t = dry.As[T](p.Pool.Get())
	t.Reset()
	return t, func() {
		p.put(t)
	}
}

func (p *Pool[T]) put(x T) {
	p.Put(x)
}

// NewPool creates a new pool of objects of type resettable.
func NewPool[T resettable](newF func() T) *Pool[T] {
	return &Pool[T]{
		Pool: sync.Pool{
			New: func() any {
				return newF()
			},
		},
	}
}
