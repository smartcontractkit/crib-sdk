package internal

import (
	"encoding/json"
	"fmt"
)

func Example_defaults() {
	// In this example you can see how setting defaults works with nested structures and validation.

	type (
		child struct {
			SomeVal int `default:"15" validate:"required,gte=0,lte=100"`
		}

		example struct {
			Name  string `default:"Hello, World!" validate:"required"`
			Child child
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
	//     "SomeVal": 15
	//   }
	// }
}
