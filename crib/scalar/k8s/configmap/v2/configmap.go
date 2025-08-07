// Package configmapv2 provides a Kubernetes ConfigMap component for Crib SDK.
package configmapv2

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2/k8s"
	"github.com/imdario/mergo"
	"github.com/samber/lo"

	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/port"
	"github.com/smartcontractkit/crib-sdk/internal/core/service"
)

type (
	IConfigMap interface {
		ConfigMap() *Component
	}

	Component struct {
		Name      string `validate:"required"`
		Namespace string `validate:"required"`

		Data        map[string]string
		AppName     string `validate:"required"`
		AppInstance string `validate:"required"`
	}

	identifier struct {
		Name      string
		Namespace string
	}

	// ComponentMapper is a type that allows Composites to iterate over a set of exposed Configs.
	ComponentMapper struct {
		opts []ConfigMapOpt
	}

	ConfigMapOpt func(*Component) error
)

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

// WithData is a ConfigMapOpt that sets the data of the ConfigMap component.
func WithData(data map[string]string) ConfigMapOpt {
	return func(c *Component) error {
		if data == nil {
			return errors.New("data cannot be nil")
		}
		currentLength := len(c.Data)
		err := mergo.Map(&c.Data, data, mergo.WithOverride)
		if err != nil {
			return fmt.Errorf("merging data into ConfigMap failed: %w", err)
		}
		if len(c.Data) != currentLength+len(data) {
			return fmt.Errorf("merging data into ConfigMap failed: %w", err)
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

// WithValuesLoader is a ConfigMapOpt that uses the provided ValuesLoader to populate the ConfigMap data.
func WithValuesLoader(loader port.ValuesLoader) ConfigMapOpt {
	return func(c *Component) error {
		if loader == nil {
			return errors.New("values loader cannot be nil")
		}
		if c.Data != nil {
			return errors.New("cannot use both Data and ValuesLoader at the same time")
		}
		values, err := loader.Values()
		if err != nil {
			return fmt.Errorf("loading values: %w", err)
		}
		c.Data = make(map[string]string, len(values))
		for key, value := range values {
			c.Data[key] = dry.MustAs[string](value)
		}
		return nil
	}
}

// Components creates a new Mapper for ConfigMap components. This can be used by Composites to iterate over a set of
// provided ConfigMaps. To provide a ConfigMap, a Scalar should implement the `IConfigMap` interface.
// When invoking the Apply method, the Mapper will group ConfigMaps by their name and namespace and apply them.
// If multiple ConfigMaps share a name and namespace, the values will be merged into a single ConfigMap.
// Components support `ConfigMapOpt` functions to modify the ConfigMap before it is applied, such as coercing
// the name or namespace. ConfigMapOpts are applied to all ConfigMaps found in the Composite context.
func Components(opts ...ConfigMapOpt) *ComponentMapper {
	cm := &ComponentMapper{
		opts: make([]ConfigMapOpt, len(opts)),
	}
	copy(cm.opts, opts)
	return cm
}

// String returns the name of the component mapper.
func (m *ComponentMapper) String() string {
	return "sdk.ConfigMapV2#ComponentMapper"
}

// Apply iterates over the available []ConfigMap components in the Composite context. It first applies the
// ConfigMapOpts to each ConfigMap, then groups them by their name and namespace. If any ConfigMaps share the same
// name and namespace, their data will be merged into a single ConfigMap. If, in the process of merging, the data
// gets overridden, an error will be returned. This is to ensure that the ConfigMap is deterministic and does not
// contain conflicting data.
//
// The resulting set of ConfigMaps is then applied using the Scalar ConfigMap methods.
func (m *ComponentMapper) Apply(ctx context.Context, cms []IConfigMap) error {
	if len(cms) == 0 {
		return nil // No ConfigMaps to apply.
	}
	// Create the new Components.
	ccs, err := m.createComponents(ctx, cms)
	if err != nil {
		return err
	}
	_, err = m.mapComponents(ccs)
	return err
}

// createComponents iterates over the provided ConfigMaps, applies the ConfigMapOpts to each one, and validates them.
func (m *ComponentMapper) createComponents(ctx context.Context, cms []IConfigMap) ([]*Component, error) {
	var errs error
	components := make([]*Component, 0, len(cms))
	for i, cm := range cms {
		if cm == nil {
			continue // Skip nil ConfigMaps.
		}
		component := cm.ConfigMap()
		if component == nil {
			continue // Skip nil components.
		}
		// Create a new Component with the provided options. We'll prepend the WithComponentOverride option first
		// to set the base properties of the ConfigMap, then apply the rest of the options.
		opts := append([]ConfigMapOpt{WithComponentOverride(component)}, m.opts...)
		nc := New(component.Name, opts...)()
		// Validate each component before adding it to the list.
		if err := nc.Validate(ctx); err != nil {
			errs = errors.Join(errs, fmt.Errorf("validating ConfigMap component for index %d (%q): %w", i, nc.Name, err))
			continue
		}
		components[i] = nc
	}
	// Remove nil components from the slice.
	components = lo.Compact(components)
	components = slices.Clip(components)
	return dry.Wrap2(components, errs)
}

// mapComponents takes a slice of ConfigMap components and merges them by their name and namespace.
func (*ComponentMapper) mapComponents(components []*Component) ([]*Component, error) {
	var errs error
	// Create a map to hold the components by their identifier (name and namespace).
	componentMap := map[identifier]*Component{}
	for _, c := range components {
		id := identifier{
			Name:      c.Name,
			Namespace: c.Namespace,
		}
		existing, exists := componentMap[id]
		if !exists {
			// If the component does not exist, add it to the map.
			componentMap[id] = c
			continue
		}
		// Determine if we will fail to merge the data map due to conflicts.
		if err := calculateDiff(id, existing.Data, c.Data); err != nil {
			// If there are conflicts, we cannot merge the components.
			errs = errors.Join(errs, err)
			continue
		}
		// Merge the new component's data into the existing one.
		if err := mergo.Merge(existing, c, mergo.WithOverride); err != nil {
			errs = errors.Join(errs, fmt.Errorf("merging ConfigMap %q: %w", id.String(), err))
			continue
		}
	}
	if errs != nil {
		return nil, errs // Return early if there were errors during merging.
	}
	if len(componentMap) == 0 {
		return nil, nil // No components to return.
	}

	// Iterate over the components to create a slice that mimics the original order.
	keyMap := make(map[identifier]struct{}, len(componentMap))
	for _, c := range components {
		id := identifier{
			Name:      c.Name,
			Namespace: c.Namespace,
		}
		keyMap[id] = struct{}{} // Track unique identifiers to maintain order.
	}

	// Iterate over components again to try keep the rough order of the original components.
	retComponents := make([]*Component, 0, len(componentMap))
	seen := make(map[identifier]struct{}, len(componentMap))
	for _, c := range components {
		id := identifier{
			Name:      c.Name,
			Namespace: c.Namespace,
		}
		if _, exists := seen[id]; exists {
			continue // Skip components that have already been added to the result.
		}
		seen[id] = struct{}{} // Mark this identifier as seen.
		retComponents = append(retComponents, componentMap[id])
	}
	return retComponents, nil
}

func calculateDiff(id identifier, m1, m2 map[string]string) error {
	m1Keys := slices.Collect(maps.Keys(m1))
	m2Keys := slices.Collect(maps.Keys(m2))
	slices.Sort(m1Keys)
	slices.Sort(m2Keys)
	diff := lo.Intersect(m1Keys, m2Keys)
	if len(diff) == 0 {
		return nil // No conflicts, nothing to report.
	}
	slices.Sort(diff)
	return fmt.Errorf("configmap %q cannot override conflicting keys: %q", id.String(), diff)
}

// New initializes a new ConfigMap component with the provided name and options.
// TODO(polds): Return the error returned by the ConfigMapOpt functions. The Composite API doesn't support this yet.
func New(name string, opts ...ConfigMapOpt) func() *Component {
	return func() *Component {
		c := &Component{
			Name: name,
		}
		for _, opt := range opts {
			_ = opt(c) // Ignore errors for now, as the Composite API does not support returning errors.
		}
		return c
	}
}

// String returns the name of the component.
func (c *Component) String() string {
	return "sdk.ConfigMapV2"
}

func (c *Component) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(c)
}

func (c *Component) Apply(cf service.ChartFactory) *Component {
	chart := cf.CreateChart(c.String(), c)

	k8s.NewKubeConfigMap(chart, chart.ToString(), &k8s.KubeConfigMapProps{
		Data: dry.PtrMapping(c.Data),
	})

	return &Component{
		Name: c.Name,
		Data: c.Data,
	}
}

func (i identifier) String() string {
	return fmt.Sprintf("%s/%s", i.Name, i.Namespace)
}
