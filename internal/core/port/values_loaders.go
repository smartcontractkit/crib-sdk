package port

import "io"

// ValuesLoader is an interface that defines a method to load Helm chart values.
type ValuesLoader interface {
	// Values loads the values for a Helm chart.
	Values() (values map[string]any, err error)
}

// ValuesParser is an interface that extends ValuesLoader to include a method for parsing values from an io.Reader.
type ValuesParser interface {
	ValuesLoader

	// Parse reads from the provided io.Reader and parses the values.
	Parse(r io.Reader) error
}
