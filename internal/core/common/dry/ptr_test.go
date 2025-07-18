package dry

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func ExampleToPtr() {
	v := 1
	p := ToPtr(v)
	fmt.Printf("%T(%v)", p, *p)

	// Output: *int(1)
}

func ExampleFromPtr() {
	v := 1
	p := ToPtr(v)
	fmt.Printf("%T(%v)", p, FromPtr(p))

	// Output: *int(1)
}

func TestToFrom(t *testing.T) {
	t.Parallel()

	t.Run("string", func(t *testing.T) {
		t.Parallel()
		v := "test"
		p := ToPtr("test")
		assert.Equal(t, v, FromPtr(p))
	})
	t.Run("int", func(t *testing.T) {
		t.Parallel()
		v := 42
		p := ToPtr(v)
		assert.Equal(t, v, FromPtr(p))
	})
	t.Run("bool", func(t *testing.T) {
		t.Parallel()
		v := true
		p := ToPtr(v)
		assert.Equal(t, v, FromPtr(p))
	})
}

func TestFrom(t *testing.T) {
	t.Parallel()

	t.Run("nil pointer", func(t *testing.T) {
		t.Parallel()
		var p *int
		assert.Equal(t, 0, FromPtr(p)) // Expect zero value for int
	})

	t.Run("non-nil pointer", func(t *testing.T) {
		t.Parallel()
		v := 100
		p := &v
		assert.Equal(t, v, FromPtr(p)) // Expect the value pointed to by p
	})

	t.Run("empty type", func(t *testing.T) {
		t.Parallel()
		var p *struct{}
		assert.Equal(t, struct{}{}, FromPtr(p)) // Expect zero value for struct{}
	})
}

func TestPtrMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   func(*testing.T)
	}{
		{
			name: "nil map",
			fn: func(t *testing.T) {
				var m map[string]string
				ptrMappingHelper(t, m)
			},
		},
		{
			name: "map[string]string",
			fn: func(t *testing.T) {
				m := map[string]string{"key1": "value1", "key2": "value2"}
				ptrMappingHelper(t, m)
			},
		},
		{
			name: "map[int]string",
			fn: func(t *testing.T) {
				m := map[int]string{1: "one", 2: "two"}
				ptrMappingHelper(t, m)
			},
		},
		{
			name: "map[string]string - empty map",
			fn: func(t *testing.T) {
				m := make(map[string]string)
				ptrMappingHelper(t, m)
			},
		},
		{
			name: "map[string]struct{}",
			fn: func(t *testing.T) {
				m := map[string]struct{}{"key1": {}, "key2": {}}
				ptrMappingHelper(t, m)
			},
		},
		{
			name: "complex example",
			fn: func(t *testing.T) {
				type T struct {
					A string
					B int
					C bool
				}

				m := map[string]T{
					"item1": {A: "alpha", B: 1, C: true},
					"item2": {A: "beta", B: 2, C: false},
				}
				ptrMappingHelper(t, m)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.fn(t)
		})
	}
}

func ptrMappingHelper[K comparable, V any](t *testing.T, m map[K]V) {
	t.Helper()
	got := PtrMapping(m)
	nilAssert := When(m == nil, assert.Nil, assert.NotNil)
	nilAssert(t, got)
	assert.IsType(t, &map[K]*V{}, got)
	if m != nil {
		assert.Len(t, *got, len(m))
	}
}

func TestPtrSlice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   func(*testing.T)
	}{
		{
			name: "nil slice",
			fn: func(t *testing.T) {
				m := ([]string)(nil)
				ptrSliceHelper(t, m)
			},
		},
		{
			name: "empty slice",
			fn: func(t *testing.T) {
				var m []string
				ptrSliceHelper(t, m)
			},
		},
		{
			name: "strings",
			fn: func(t *testing.T) {
				m := []string{"hello", "world", "test"}
				ptrSliceHelper(t, m)
			},
		},
		{
			name: "integers",
			fn: func(t *testing.T) {
				m := []int{1, 2, 3, 42}
				ptrSliceHelper(t, m)
			},
		},
		{
			name: "booleans",
			fn: func(t *testing.T) {
				m := []bool{true, false, true}
				ptrSliceHelper(t, m)
			},
		},
		{
			name: "structs",
			fn: func(t *testing.T) {
				type Example struct {
					Name  string
					Value int
				}
				m := []Example{
					{Name: "example1", Value: 1},
					{Name: "example2", Value: 2},
				}
				ptrSliceHelper(t, m)
			},
		},
		{
			name: "struct slice",
			fn: func(t *testing.T) {
				var m []struct{}
				ptrSliceHelper(t, m)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.fn(t)
		})
	}
}

func ptrSliceHelper[V any](t *testing.T, m []V) {
	t.Helper()
	got := PtrSlice(m)
	nilAssert := When(m == nil, assert.Nil, assert.NotNil)
	nilAssert(t, got)
	assert.IsType(t, &[]*V{}, got)
	if m != nil {
		assert.Len(t, *got, len(m))
	}
}

func TestStrings(t *testing.T) {
	t.Parallel()

	t.Run("multiple strings", func(t *testing.T) {
		t.Parallel()
		result := PtrStrings("echo", "Hello from Job!", "test")

		assert.NotNil(t, result)
		assert.Len(t, *result, 3)
		assert.Equal(t, "echo", *(*result)[0])
		assert.Equal(t, "Hello from Job!", *(*result)[1])
		assert.Equal(t, "test", *(*result)[2])
	})

	t.Run("single string", func(t *testing.T) {
		t.Parallel()
		result := PtrStrings("single")

		assert.NotNil(t, result)
		assert.Len(t, *result, 1)
		assert.Equal(t, "single", *(*result)[0])
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		result := PtrStrings()

		assert.Nil(t, result)
	})
}
