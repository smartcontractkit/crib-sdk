package internal

import (
	"encoding/json"
	"fmt"
)

// CustomResource represents a type that satisfies the defaults Setter interface.
type CustomResource struct {
	A string `validate:"required"`
	B string `validate:"required"`
}

func (c *CustomResource) SetDefaults() {
	// CanUpdate verifies that the current value is neither the default nor zero value.
	if CanUpdate(c.A) {
		c.A = "Default value for A"
	}
	if CanUpdate(c.B) {
		c.B = "Default value for B"
	}
}

func Example_customDefaults() {
	// In this example we implement the defaults Setter interface which allows
	// us to perform complex logic to set a default value.

	v, _ := NewValidator()
	c := &CustomResource{
		A: "Custom value for A",
		// Don't set a value for B.
	}
	fmt.Println(v.Struct(c))
	raw, _ := json.MarshalIndent(c, "", " ")
	fmt.Println(string(raw))

	// Output:
	// <nil>
	// {
	//  "A": "Custom value for A",
	//  "B": "Default value for B"
	// }
}
