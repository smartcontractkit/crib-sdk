package dry

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func ExampleAs() {
	// Create a generic map.
	var m any = map[string]any{
		"key": "value",
	}
	fmt.Printf("Type: %T\n", m)

	// Use As to assert the type of the map.
	mm := As[map[string]any](m)
	fmt.Printf("Type: %T\n", mm)

	// Further assert the value.
	k := As[string](mm["key"])
	fmt.Printf("Type: %T\n", k)
	fmt.Printf("Value: %q\n", k)

	// An unsafe assertion that would normally panic if not properly handled.
	u := As[map[string]bool](nil)
	fmt.Printf("Type: %T\n", u)
	fmt.Printf("Value: %v\n", u["key"])

	// Output:
	// Type: map[string]interface {}
	// Type: map[string]interface {}
	// Type: string
	// Value: "value"
	// Type: map[string]bool
	// Value: false
}

func TestAs(t *testing.T) {
	t.Run("Basic types", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			v := As[int](1)
			assert.Equal(t, 1, v, "As() = %v; want 1", v)
		})
		t.Run("Failure", func(t *testing.T) {
			v := As[int]("string")
			assert.Equal(t, 0, v, "As() = %v; want 0", v)
		})
	})
	t.Run("Types fun", func(t *testing.T) {
		t.Run("Aliasing", func(t *testing.T) {
			t.Run("Success with string", func(t *testing.T) {
				type MyString = string
				const want = "string"
				v := As[MyString](want)
				assert.Equal(t, want, v, "As() = %[1]q (%[1]T); want %[2]q (%[2]T)", v, want)
			})
			t.Run("Success with MyString", func(t *testing.T) {
				type MyString = string
				const want MyString = "string"
				v := As[MyString](want)
				assert.Equal(t, want, v, "As() = %[1]q (%[1]T); want %[2]q (%[2]T)", v, want)
			})
			t.Run("Failure", func(t *testing.T) {
				type MyString = string
				v := As[MyString](1)
				assert.Equal(t, "", v, "As() = %[1]q (%[1]T); want %[2]q (%[2]T)", v, "")
			})
		})
		t.Run("Definition", func(t *testing.T) {
			t.Run("Success", func(t *testing.T) {
				type MyString string
				const want MyString = "string"
				v := As[MyString](want)
				assert.Equal(t, want, v, "As() = %[1]v (%[1]T); want %[2]q (%[2]T)", v, want)
			})
			t.Run("Failure", func(t *testing.T) {
				type MyString string
				v := As[MyString](1)
				assert.Equal(t, MyString(""), v, "As() = %[1]q (%[1]T); want %[2]q (%[2]T)", v, "")
			})
		})
		t.Run("Complex", func(t *testing.T) {
			type complexType struct {
				A int
				B string
			}
			t.Run("Non-Pointer", func(t *testing.T) {
				t.Run("Success", func(t *testing.T) {
					want := complexType{A: 1, B: "string"}
					v := As[complexType](want)
					assert.Equal(t, want, v, "As() = %[1]v (%[1]T); want %[2]v (%[2]T)", v, want)
				})
				t.Run("Failure", func(t *testing.T) {
					v := As[complexType](1)
					assert.Equal(t, complexType{}, v, "As() = %[1]v (%[1]T); want %[2]v (%[2]T)", v, complexType{})
				})
			})
			t.Run("Pointer", func(t *testing.T) {
				t.Run("Success", func(t *testing.T) {
					want := &complexType{A: 1, B: "string"}
					v := As[*complexType](want)
					assert.Equal(t, want, v, "As() = %[1]v (%[1]T); want %[2]v (%[2]T)", v, want)
				})
				t.Run("Failure", func(t *testing.T) {
					v := As[*complexType](1)
					assert.Nil(t, v, "As() = %v; want nil", v)
				})
			})
		})
	})
}

func TestMustAs(t *testing.T) {
	assert.Panics(t, func() {
		var i int
		_ = MustAs[float64](i)
	})
}
