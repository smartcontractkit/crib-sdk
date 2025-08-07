package configmapv2

import (
	"errors"
	"fmt"
	"github.com/imdario/mergo"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/port"
)

// ConfigMapOpt defines a function type that takes a pointer to a Component and returns an error.
// It is applied before the Component is created, allowing for configuration of the Component's properties.
type ConfigMapOpt func(*Component) error

// WithComponentOverride is a ConfigMapOpt that merges the existing Component with the provided one.
func WithComponentOverride(component *Component) ConfigMapOpt {
	return func(c *Component) error {
		return mergo.Merge(c, component, mergo.WithOverride)
	}
}

// WithName is a ConfigMapOpt that sets the name of the ConfigMap component.
func WithName(name string) ConfigMapOpt {
	return func(c *Component) error {
		if name == "" {
			return errors.New("name cannot be empty")
		}
		c.Name = name
		return nil
	}
}

// WithNamespace is a ConfigMapOpt that sets the namespace of the ConfigMap component.
func WithNamespace(namespace string) ConfigMapOpt {
	return func(c *Component) error {
		if namespace == "" {
			return errors.New("namespace cannot be empty")
		}
		c.Namespace = namespace
		return nil
	}
}

// WithData is a ConfigMapOpt that merges the data of the ConfigMap component.
func WithData(data map[string]string) ConfigMapOpt {
	return func(c *Component) error {
		if data == nil {
			return errors.New("data cannot be nil")
		}
		return dry.Wrapf(mergo.Map(&c.Data, data, mergo.WithOverride), "merging data into ConfigMap %q failed", c.Name)
	}
}

// WithValuesLoader is a ConfigMapOpt that uses the provided ValuesLoader to populate the ConfigMap data.
func WithValuesLoader(loader port.ValuesLoader) ConfigMapOpt {
	return func(c *Component) error {
		if loader == nil {
			return errors.New("values loader cannot be nil")
		}
		values, err := loader.Values()
		if err != nil {
			return fmt.Errorf("loading values: %w", err)
		}
		// Create a map if it doesn't exist yet.
		if c.Data == nil {
			c.Data = make(map[string]string, len(values))
		}
		for key, value := range values {
			c.Data[key] = dry.MustAs[string](value)
		}
		return nil
	}
}

// WithAppName is a ConfigMapOpt that sets the application name of the ConfigMap component.
func WithAppName(appName string) ConfigMapOpt {
	return func(c *Component) error {
		if appName == "" {
			return errors.New("app name cannot be empty")
		}
		c.AppName = appName
		return nil
	}
}

// WithAppInstance is a ConfigMapOpt that sets the application instance of the ConfigMap component.
func WithAppInstance(appInstance string) ConfigMapOpt {
	return func(c *Component) error {
		if appInstance == "" {
			return errors.New("app instance cannot be empty")
		}
		c.AppInstance = appInstance
		return nil
	}
}
