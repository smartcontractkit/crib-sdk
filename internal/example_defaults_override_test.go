package internal

import (
	"encoding/json"
	"fmt"
)

func Example_defaultsParentOverride() {
	// This example demonstrates how to override default values in a parent struct
	// when the child struct has its own default values defined.

	type (
		child struct {
			SomeVal int `default:"15" validate:"required,gte=0,lte=100"`
		}

		example struct {
			Name string `default:"Hello, World!" validate:"required"`
			// Example is the parent of child and overrides its default value.
			Child child `default:"{ \"SomeVal\": 10 }" validate:"required"`
		}
	)

	v, _ := NewValidator()
	e := &example{}
	fmt.Println(v.Struct(e))
	raw, _ := json.MarshalIndent(e, "", "  ")
	fmt.Println(string(raw))

	// Output:
	// <nil>
	// {
	//   "Name": "Hello, World!",
	//   "Child": {
	//     "SomeVal": 10
	//   }
	// }
}
