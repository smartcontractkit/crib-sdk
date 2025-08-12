package service

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"path"
	"reflect"
	"runtime"
	"sync"
	"weak"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/samber/lo"
	"go.uber.org/fx"

	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/infra"
	"github.com/smartcontractkit/crib-sdk/internal/core/port"
)

// voidValue is the [reflect.Value] for an empty struct.
var voidValue = reflect.ValueOf(struct{}{})

type (
	AutoComponent struct {
		component   any
		name        string         // The name of the component, derived from the constructor or component type.
		applyMethod reflect.Method // The Apply method of the component.
		produces    reflect.Type
		consumes    []reflect.Type
		isSliceType bool // for collecting multiple instances like []GroupResult
	}

	// Constructor is a resolved Component constructor that can be used to create a new instance of a component.
	// For example, the constructor is the result of `func() *MyComponent` or `func() (*MyComponent, error)`.
	Constructor struct {
		component any                      // The actual component instance created by the constructor.
		namer     func(name string) string // The name of the component, derived from the constructor or component type.
		cType     reflect.Type             // The type of the component, used for reflection.
	}

	// ComponentExecutor defines the interface for executing individual components.
	ComponentExecutor interface {
		// ExecuteComponent executes a single AutoComponent and handles its dependencies.
		// It is responsible for providing the necessary parameters to the component's Apply method.
		ExecuteComponent(comp *AutoComponent) error
	}

	// Composite represents a set of Scalar Components that are executed in a dependency graph.
	// It is the main entry point for the Composite pattern.
	Composite struct {
		executor     ComponentExecutor
		results      map[reflect.Type]any
		sliceResults map[reflect.Type][]any
		components   []*AutoComponent
		mu           sync.RWMutex
	}

	// CompositeSet contains fx Options and is used by a Plan to execute a Composite.
	CompositeSet struct {
		fxOpts []fx.Option
	}

	// chartContext is a composite builtin that injects a method to fetch a context.Context that is used to create a cdk8s.Chart instance.
	// The provided context contains the full root chart context and can be used to create other constructs within the chart.
	chartContext struct {
		instanceCtx func() context.Context
	}

	// ChartFactory provides a clean interface for components to create charts without boilerplate.
	// The framework automatically provides an implementation that handles context and resource ID generation.
	//
	// Example usage in a component:
	//   func (c *MyComponent) Apply(factory ChartFactory) *MyResult {
	//       chart := factory.CreateChart(c)
	//       // Use chart to create Kubernetes resources...
	//       return &MyResult{Chart: chart}
	//   }
	//
	// This eliminates the need for these repetitive lines in every component:
	//   parent := internal.ConstructFromContext(ctx)
	//   chart := cdk8s.NewChart(parent, crib.ResourceID("MyComponent", props), nil)
	ChartFactory interface {
		// CreateChart creates a new cdk8s.Chart instance with the given resource name and props.
		CreateChart(v any) cdk8s.Chart
	}

	// chartFactory implements ChartFactory and is automatically injected by the framework.
	chartFactory struct {
		instanceCtx func() context.Context
	}

	// CompositeResult satisfies the port.Component interface and contains the result of applying a Composite.
	// TODO(polds): This is inaccurate. It only contains the root cdk8s.Chart instance.
	// 	Still need a way to fetch into a composite and get their results.
	CompositeResult struct {
		port.Component
	}

	// constructorRefs exists as a way to allow the composite with their constructors to implement the propsValidator
	// interface so that a Resource ID can be generated for the Composite. We take a weak reference to the constructor
	// so that we can json encode the Composite without having to worry about the constructors being garbage collected.
	constructorRefs struct {
		Refs []weak.Pointer[any]
	}
)

func newChartContext(ctx context.Context) func() *chartContext {
	return func() *chartContext {
		return &chartContext{
			instanceCtx: func() context.Context {
				return ctx
			},
		}
	}
}

func (c *chartContext) Apply() context.Context {
	return c.instanceCtx()
}

func (c *chartContext) Name() string {
	return "sdk.composite.builtin.chartContext"
}

func newChartFactory() *chartFactory {
	return &chartFactory{}
}

// CreateChart implements ChartFactory by creating a chart with the provided resource.
// It handles the boilerplate of getting the parent construct, generating resource IDs, and creating the chart.
func (c *chartFactory) CreateChart(v any) cdk8s.Chart {
	parent := internal.ConstructFromContext(c.instanceCtx())

	var name string
	if strer, ok := v.(fmt.Stringer); ok {
		name = strer.String()
	} else {
		// If the value does not implement fmt.Stringer, use its type name as a fallback.
		name = reflect.TypeOf(v).Name()
	}

	return cdk8s.NewChart(parent, infra.ResourceID(name, v), nil)
}

// Apply implements the component interface for chartFactory, making it injectable.
func (c *chartFactory) Apply(ctx context.Context) ChartFactory {
	c.instanceCtx = func() context.Context { return ctx }
	return c
}

func (c *chartFactory) Name() string {
	return "sdk.composite.builtin.chartFactory"
}

// NewCompositeSet initializes a CompositeSet with the base Fx options.
// It sets up a lifecycle hook to apply the composite when the application starts.
// The returned CompositeSet can be used to apply components defined in the Composite.
func NewCompositeSet() *CompositeSet {
	cs := &CompositeSet{
		fxOpts: []fx.Option{
			fx.Invoke(
				func(lc fx.Lifecycle, composite *Composite) {
					lc.Append(fx.Hook{
						OnStart: func(ctx context.Context) error {
							return composite.Apply(ctx)
						},
					})
				},
			),
		},
	}
	return cs
}

func (c *CompositeSet) Apply(ctx context.Context, ctors ...any) (port.Component, error) {
	id := infra.ResourceID("sdk.composite", newConstructorRefs(ctors...))
	parent := internal.ConstructFromContext(ctx)
	chart := cdk8s.NewChart(parent, id, nil)
	ctx = internal.ContextWithConstruct(ctx, chart)
	ctors = append(
		[]any{
			newChartContext(ctx),
			newChartFactory,
		},
		ctors...,
	)

	opts := append(
		[]fx.Option{registerComponents(ctors...)},
		c.fxOpts...,
	)

	app := fx.New(opts...)
	return CompositeResult{
			Component: chart,
		},
		dry.FirstError(
			dry.Wrapf(app.Start(ctx), "starting composite application"),
			dry.Wrapf(app.Stop(ctx), "cleaning up composite application"),
		)
}

func (c *Composite) Apply(context.Context) error {
	graph, err := c.dependencyGraph()
	if err != nil {
		return fmt.Errorf("building dependency graph: %w", err)
	}
	return c.executeGraph(graph)
}

func registerComponents(components ...any) fx.Option {
	c := &Composite{
		results:      make(map[reflect.Type]any),
		sliceResults: make(map[reflect.Type][]any),
	}
	// Set the executor to the composite itself by default
	c.executor = c

	var registrationErrs error
	for component, err := range Components(components...) {
		if err != nil {
			registrationErrs = errors.Join(registrationErrs, err)
			continue
		}
		c.components = append(c.components, component)
	}
	if registrationErrs != nil {
		return fx.Error(registrationErrs)
	}
	return fx.Provide(func() *Composite {
		return c
	})
}

// Components returns an iterable sequence of AutoComponent instances based on the provided constructors.
//
// Use:
//
//	for comp, err := range Components(ctors...) {
//		if err != nil {
//			// Handle error
//			continue
//		}
//		// Use comp, which is an *AutoComponent instance
//	}
func Components(ctors ...any) iter.Seq2[*AutoComponent, error] {
	nameFn := func(idx int) func(string) string {
		return func(s string) string {
			return fmt.Sprintf("%d::%s", idx, s)
		}
	}
	return func(yield func(*AutoComponent, error) bool) {
		for idx, ctor := range ctors {
			autoComp, err := dry.FirstError2(
				func() (*AutoComponent, error) {
					return nil, isCallable(ctor)
				},
				func() (*AutoComponent, error) {
					return constructor(ctor, nameFn(idx)).Analyze()
				},
			)
			if !yield(autoComp, err) {
				return
			}
		}
	}
}

// isCallable checks if the provided argument is a callable function with no required parameters.
//
// Examples of valid constructors:
//   - func() *MyComponent
//   - func() (*MyComponent, error)
//   - func() func(v any) func() *MyComponent // closure with no required parameters
//   - func() func(v any) func() (*MyComponent, error) // closure with no required parameters
func isCallable(ctor any) error {
	// check if the constructor is truly nil without a known type.
	// If the nil has a type it will fall through to the next check.
	if ctor == nil {
		return errors.New("cannot register nil component")
	}

	ctorType := reflect.TypeOf(ctor)
	if ctorType.Kind() != reflect.Func {
		return fmt.Errorf("cannot register component of type %T, component must be a callable function", ctor)
	}
	if ctorType.NumIn() > 0 {
		return fmt.Errorf("cannot register component of type %T with non-zero required arguments (is the component a closure?)", ctor)
	}
	return nil
}

// constructor returns a new Constructor by resolving the constructor function.
func constructor(v any, namerFn func(v string) string) *Constructor {
	if v == nil {
		return nil
	}
	// Call the constructor to get the component instance.
	vValue := reflect.ValueOf(v)
	results := vValue.Call(nil)
	component := results[0].Interface()
	// TODO(polds): Handle error return value if it exists.
	cType := reflect.TypeOf(component)

	return &Constructor{
		component: component,
		namer:     namerFn,
		cType:     cType,
	}
}

// Name returns a human-readable name for the component.
// The name is determined by one of three methods (in order of precedence):
//  1. If the component implements fmt.Stringer, use its String() method.
//  2. If the component implements an interface with a Name() string method, use that for the name.
//  3. If neither of the above, derive a name from the component's type information and the current function context.
func (c *Constructor) Name() string {
	if c == nil {
		return ""
	}
	if c.namer == nil {
		c.namer = func(name string) string {
			return name
		}
	}

	// If the component implements fmt.Stringer, use its String() method to get a name.
	if v, ok := c.component.(fmt.Stringer); ok {
		return c.namer(v.String())
	}
	// Alternatively, if the component implements an interface that has a `Name() string` method, use that for the name.
	if v, ok := c.component.(interface{ Name() string }); ok {
		return c.namer(v.Name())
	}

	// Finally, attempt to derive a name by using the pointer, package path, and type name.
	target := fmt.Sprintf("%s.%s", c.cType.PkgPath(), c.cType.Name())
	if len(target) == 1 {
		// Get current package name.
		if pc, _, _, ok := runtime.Caller(1); ok {
			fn := runtime.FuncForPC(pc)
			if fn != nil {
				target = path.Base(fn.Name())
			}
		}
	}

	// Rough format:
	// 	0x104aa24c0#runtime.main@func() *fx.Example
	return c.namer(fmt.Sprintf("%[1]p#%[2]s@%[1]T", c.component, target))
}

// Analyze determines the inputs and outputs of the component's Apply method.
func (c *Constructor) Analyze() (*AutoComponent, error) {
	if c == nil {
		return nil, errors.New("cannot analyze nil component")
	}
	name := c.Name()
	applyMethod, exists := c.cType.MethodByName("Apply")
	if !exists {
		return nil, fmt.Errorf("cannot register component %q, does not implement Composite, missing Apply method", name)
	}

	auto := &AutoComponent{
		component:   c.component,
		name:        name,
		applyMethod: applyMethod,
	}

	// Analyze what the Apply method produces (return type).
	methodType := applyMethod.Type
	if methodType.NumOut() > 0 {
		returnType := methodType.Out(0)
		auto.produces = returnType
		// TODO(COP-1232): Use a logger.
		// fmt.Printf("Auto-detected: %s produces %s\n", name, returnType)
	}

	// Analyze what the Apply method consumes (parameters beyond receiver, ie. skip index 0)
	for i := 1; i < methodType.NumIn(); i++ {
		paramType := methodType.In(i)
		auto.consumes = append(auto.consumes, paramType)

		// Check if it's a slice type (for collecting multiple instances)
		if paramType.Kind() == reflect.Slice {
			auto.isSliceType = true
			// TODO(COP-1232): Use a logger.
			// elemType := paramType.Elem()
			//	fmt.Printf("Auto-detected: %s consumes slice of %s\n", name, elemType)
			//	} else {
			//	fmt.Printf("Auto-detected: %s consumes %s\n", name, paramType)
		}
	}
	return auto, nil
}

// dependencyGraph builds a directed graph of dependencies between components.
// It also validates that consumers expecting single items don't have multiple producers,
// which would be confusing as only the last produced item would be used.
func (c *Composite) dependencyGraph() (map[string][]string, error) {
	edges := make(map[string][]string)

	// Initialize edges
	for i := range c.components {
		edges[c.components[i].name] = []string{}
	}

	// Build dependency edges and validate single vs multiple producer scenarios
	for i := range c.components {
		consumer := c.components[i] // Capture range variable
		for _, needsType := range consumer.consumes {
			// Handle slice types - need all producers of element type
			if needsType.Kind() == reflect.Slice {
				elemType := needsType.Elem()
				for i := range c.components {
					producer := c.components[i] // Capture range variable
					if producer.name == consumer.name {
						continue
					}
					if producer.produces == elemType {
						edges[consumer.name] = append(edges[consumer.name], producer.name)
						// TODO(COP-1232): Use a logger.
						// fmt.Printf("Dependency: %s needs []%s (collects from %s)\n",
						//	consumer.name, elemType, producer.name)
					}
				}
				continue
			}

			// Handle regular types - but first check for multiple producers
			var matchingProducers []string
			for i := range c.components {
				producer := c.components[i] // Capture range variable
				if producer.name == consumer.name {
					continue
				}
				if producer.produces == needsType {
					matchingProducers = append(matchingProducers, producer.name)
					edges[consumer.name] = append(edges[consumer.name], producer.name)
				}
			}

			// Validate single consumer doesn't have multiple producers
			if len(matchingProducers) > 1 {
				return nil, fmt.Errorf(
					"component %q consumes single %s but multiple producers exist: %v. "+
						"This is confusing because only the last produced item will be used. "+
						"Consider changing %q to consume []%s to collect all results",
					consumer.name, needsType, matchingProducers, consumer.name, needsType)
			}

			// TODO(COP-1232): Use a logger.
			// Log the dependency if we have exactly one producer
			// if len(matchingProducers) == 1 {
			//	fmt.Printf("Dependency: %s needs %s (from %s)\n",
			//	consumer.name, needsType, matchingProducers[0])
			// }
		}
	}

	return edges, nil
}

// executeGraph runs all components in the Composite in the correct order based on their dependencies.
func (c *Composite) executeGraph(edges map[string][]string) error {
	executed := make(map[string]bool)
	visiting := make(map[string]bool)
	componentMap := make(map[string]*AutoComponent)

	for i := range c.components {
		comp := c.components[i] // Capture range variable
		componentMap[comp.name] = comp
	}

	var visit func(string) error
	visit = func(name string) error {
		if executed[name] {
			return nil
		}
		if visiting[name] {
			return fmt.Errorf("circular dependency detected involving %s", name)
		}

		visiting[name] = true

		// Execute dependencies first
		for _, depName := range edges[name] {
			if err := visit(depName); err != nil {
				return err
			}
		}

		visiting[name] = false

		// Only execute if component exists in our component map
		if comp, exists := componentMap[name]; exists {
			// TODO(COP-1232): Use a logger.
			// fmt.Printf("Executing: %s\n", name)
			if err := c.executor.ExecuteComponent(comp); err != nil {
				return err
			}
		}

		executed[name] = true
		return nil
	}

	// Execute all components
	for i := range c.components {
		comp := c.components[i] // Capture range variable
		if !executed[comp.name] {
			if err := visit(comp.name); err != nil {
				return err
			}
		}
	}

	return nil
}

// ExecuteComponent implements the ComponentExecutor interface and is the default implementation to
// execute components.
func (c *Composite) ExecuteComponent(comp *AutoComponent) error {
	// Prepare arguments for Run method
	methodType := comp.applyMethod.Type
	args := []reflect.Value{reflect.ValueOf(comp.component)} // receiver

	// Add parameters automatically, start at index 1 to skip the receiver.
	for i := 1; i < methodType.NumIn(); i++ {
		paramType := methodType.In(i)
		paramValue, err := c.valueForType(paramType)
		if err != nil {
			return fmt.Errorf("failed to provide params for %s: %w", comp.name, err)
		}
		args = append(args, paramValue)
	}

	// Execute the Apply method
	results := comp.applyMethod.Func.Call(args)

	// Try to find a returned error, if it exists. If it's not nil, return it.
	for _, res := range results {
		if res.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			if err, ok := res.Interface().(error); ok && err != nil {
				return fmt.Errorf("executing component %s: %w", comp.name, err)
			}
		}
	}

	// Store result if component produces something
	if len(results) > 0 && comp.produces != nil {
		// Note: Possible bug / confusing user experience.
		// If the component produces multiple values, we only store the first one.
		// If, for example, a component author returns 'err, bool' instead of 'bool, error'
		// then a nil error will be stored and the user will not know that the component produced an error.
		result := results[0].Interface()

		c.mu.Lock()
		c.results[comp.produces] = result

		// Also add to slice collection for slice consumers
		if c.sliceResults[comp.produces] == nil {
			c.sliceResults[comp.produces] = []any{}
		}
		c.sliceResults[comp.produces] = append(c.sliceResults[comp.produces], result)
		c.mu.Unlock()

		// TODO(COP-1232): Use a logger.
		// fmt.Printf("  -> Stored %s for future consumption\n", comp.produces)
	}

	return nil
}

func (c *Composite) valueForType(paramType reflect.Type) (reflect.Value, error) {
	switch paramType.Kind() {
	case reflect.Interface:
		implementations := c.findImplementations(paramType)
		if len(implementations) == 0 {
			return voidValue, fmt.Errorf("missing dependency %s (no concrete type found that implements this interface)", paramType)
		}
		return implementations[0], nil

	case reflect.Slice:
		elemType := paramType.Elem()
		implementations := c.findImplementations(elemType)
		return c.createSliceFromValues(implementations, paramType), nil

	default:
		// Handle regular concrete type parameters
		c.mu.RLock()
		value, exists := c.results[paramType]
		c.mu.RUnlock()

		if !exists {
			return voidValue, fmt.Errorf("no registered component provides missing dependency %s", paramType)
		}
		return reflect.ValueOf(value), nil
	}
}

// findImplementations finds all instances that satisfy the given type.
// For interface types, it searches both results and sliceResults for implementations.
// For concrete types, it gets instances from sliceResults.
func (c *Composite) findImplementations(targetType reflect.Type) []reflect.Value {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var implementations []reflect.Value
	seen := make(map[string]bool)

	if targetType.Kind() == reflect.Interface {
		// For interface types, search both maps for implementations
		c.findInterfaceImplementations(targetType, &implementations, seen)
		return implementations
	}

	// For concrete types, get instances from sliceResults
	if instances, exists := c.sliceResults[targetType]; exists {
		for _, instance := range instances {
			val := reflect.ValueOf(instance)
			key := c.createUniqueKey(instance)
			if !seen[key] {
				implementations = append(implementations, val)
				seen[key] = true
			}
		}
	}
	return implementations
}

// findInterfaceImplementations searches both results and sliceResults for interface implementations.
func (c *Composite) findInterfaceImplementations(targetType reflect.Type, implementations *[]reflect.Value, seen map[string]bool) {
	// Search main results map
	for _, resultValue := range c.results {
		if reflect.TypeOf(resultValue).Implements(targetType) {
			val := reflect.ValueOf(resultValue)
			key := c.createUniqueKey(resultValue)
			if !seen[key] {
				*implementations = append(*implementations, val)
				seen[key] = true
			}
		}
	}

	// Search slice results map
	for _, sliceInstances := range c.sliceResults {
		for _, instance := range sliceInstances {
			if reflect.TypeOf(instance).Implements(targetType) {
				val := reflect.ValueOf(instance)
				key := c.createUniqueKey(instance)
				if !seen[key] {
					*implementations = append(*implementations, val)
					seen[key] = true
				}
			}
		}
	}
}

// createUniqueKey generates a unique key for deduplication based on type and pointer/value.
func (c *Composite) createUniqueKey(value any) string {
	val := reflect.ValueOf(value)
	if val.Kind() == reflect.Ptr {
		return fmt.Sprintf("%s:%x", reflect.TypeOf(value), val.Pointer())
	}
	return fmt.Sprintf("%s:%v", reflect.TypeOf(value), value)
}

// createSliceFromValues creates a slice of the specified type from the given values.
func (c *Composite) createSliceFromValues(values []reflect.Value, sliceType reflect.Type) reflect.Value {
	sliceValue := reflect.MakeSlice(sliceType, len(values), len(values))
	for i, val := range values {
		sliceValue.Index(i).Set(val)
	}
	return sliceValue
}

func newConstructorRefs(ctors ...any) constructorRefs {
	return constructorRefs{
		Refs: lo.Map(ctors, func(ctor any, _ int) weak.Pointer[any] {
			return weak.Make(dry.ToPtr(ctor))
		}),
	}
}

func (constructorRefs) Validate(context.Context) error {
	// This is a no-op, as the CompositeSet does not have any props to validate
	return nil
}
