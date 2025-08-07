package configmapv2

import (
	"context"
	"errors"
	"fmt"
	"github.com/smartcontractkit/crib-sdk/internal/core/service"
	"maps"
	"slices"

	"github.com/imdario/mergo"
	"github.com/samber/lo"
)

type (
	// IConfigMap is an interface that should be implemented by any Scalar that provides a ConfigMap config.
	IConfigMap interface {
		// ConfigMap returns a Component that represents the ConfigMap to be applied.
		ConfigMap() *Component
	}

	// ComponentMapper is a type that allows Composites to iterate over a set of exposed Configs.
	ComponentMapper struct {
		opts []ConfigMapOpt
	}

	identifier struct {
		Name      string
		Namespace string
	}
)

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
func (m *ComponentMapper) Apply(ctx context.Context, cf service.ChartFactory, cms []IConfigMap) error {
	if len(cms) == 0 {
		return nil // No ConfigMaps to apply.
	}

	// Create the new Components.
	ccs := m.createComponents(cms)
	mapped, err := m.mapComponents(ccs)
	if err != nil {
		return fmt.Errorf("mapping ConfigMaps: %w", err)
	}

	// Validate and Apply each ConfigMap.
	for _, c := range mapped {
		if err := c.Validate(ctx); err != nil {
			return err
		}
		if _, err := c.Apply(ctx, cf); err != nil {
			return err
		}
	}
	return nil
}

// createComponents iterates over the provided ConfigMaps, applies the ConfigMapOpts to each one, and validates them.
func (m *ComponentMapper) createComponents(cms []IConfigMap) []*Component {
	components := make([]*Component, len(cms))
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
		nc := newScalar(component.Name, opts...)
		components[i] = nc
	}
	// Remove nil components from the slice.
	components = lo.Compact(components)
	components = slices.Clip(components)
	// Explicitly return nil if there are no components vs returning an empty slice.
	if len(components) == 0 {
		return nil // No valid components to return.
	}
	return components
}

// mapComponents takes a slice of ConfigMap components and merges them by their name and namespace.
func (*ComponentMapper) mapComponents(components []*Component) ([]*Component, error) {
	var errs error
	// Create a map to hold the components by their identifier (name and namespace).
	componentMap := make(map[identifier]*Component)
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

func (i identifier) String() string {
	return fmt.Sprintf("%s/%s", i.Name, i.Namespace)
}
