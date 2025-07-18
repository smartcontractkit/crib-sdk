package internal

import "fmt"

// ExampleValidator demonstrates a simple struct that can be validated.
// All available validation tags can be found at https://pkg.go.dev/github.com/go-playground/validator/v10.
type ExampleValidator struct {
	Name            string `validate:"required"`
	AppInstanceName string `default:"test-app"`
	Config          ValidationConfig
}

type ValidationConfig struct {
	SomeVal int `validate:"required,gte=0,lte=100"`
}

func Example_validator() {
	v, err := NewValidator()
	if err != nil {
		fmt.Println("Failed to create validator:", err)
		return
	}

	invalid := &ExampleValidator{
		Name: "", // Missing required field
		Config: ValidationConfig{
			SomeVal: -1, // Invalid value
		},
	}
	fmt.Println(v.Struct(invalid))

	valid := &ExampleValidator{
		Name: "Valid Name",
		Config: ValidationConfig{
			SomeVal: 50, // Valid value
		},
	}
	err = v.Struct(valid)
	if err != nil {
		fmt.Println("Failed to validate struct:", err)
	}
	fmt.Printf("AppInstanceName is set by default: %s\n", invalid.AppInstanceName)
	fmt.Println(err)

	// Output:
	// Key: 'ExampleValidator.Name' Error:Field validation for 'Name' failed on the 'required' tag
	// Key: 'ExampleValidator.Config.SomeVal' Error:Field validation for 'SomeVal' failed on the 'gte' tag
	// AppInstanceName is set by default: test-app
	// <nil>
}
