package service

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/crib-sdk/internal/core/common/infra"
	"github.com/smartcontractkit/crib-sdk/internal/core/port"
)

// Test types for analyzeConstructor tests

// Simple component with Apply method - no params, no return
type SimpleComponent struct{}

func NewSimpleComponent() *SimpleComponent {
	return &SimpleComponent{}
}

func (s *SimpleComponent) Apply() {}

func (s *SimpleComponent) String() string {
	return "SimpleComponent"
}

// Component with Apply method - no params, single return
type ProducerComponent struct{}

func NewProducerComponent() *ProducerComponent {
	return &ProducerComponent{}
}

func (p *ProducerComponent) Apply() string {
	return "produced_value"
}

// Component with Apply method - no params, multiple returns
type MultiReturnComponent struct{}

func NewMultiReturnComponent() *MultiReturnComponent {
	return &MultiReturnComponent{}
}

func (m *MultiReturnComponent) Apply() (string, error) {
	return "value", nil
}

// Component with Apply method - single param, no return
type ConsumerComponent struct{}

func NewConsumerComponent() *ConsumerComponent {
	return &ConsumerComponent{}
}

func (c *ConsumerComponent) Apply(input string) {}

// Component with Apply method - multiple params, no return
type MultiConsumerComponent struct{}

func NewMultiConsumerComponent() *MultiConsumerComponent {
	return &MultiConsumerComponent{}
}

func (m *MultiConsumerComponent) Apply(input1 string, input2 int) {}

// Component with Apply method - slice param, no return
type SliceConsumerComponent struct{}

func NewSliceConsumerComponent() *SliceConsumerComponent {
	return &SliceConsumerComponent{}
}

func (s *SliceConsumerComponent) Apply(inputs []string) {}

// Component with Apply method - mixed params with return
type MixedComponent struct{}

func NewMixedComponent() *MixedComponent {
	return &MixedComponent{}
}

func (m *MixedComponent) Apply(regular string, slice []int, ctx context.Context) (string, error) {
	return "result", nil
}

// Component without Apply method
type NoApplyComponent struct{}

func NewNoApplyComponent() *NoApplyComponent {
	return &NoApplyComponent{}
}

func (n *NoApplyComponent) Run() {}

// Component without fmt.Stringer implementation
type NonStringerComponent struct{}

func NewNonStringerComponent() *NonStringerComponent {
	return &NonStringerComponent{}
}

func (n *NonStringerComponent) Apply() {}

// Constructor that panics
func NewPanicComponent() *SimpleComponent {
	panic("constructor panic")
}

// Constructor with wrong return type (no returns)
func NewNoReturnComponent() {
	// This constructor returns nothing
}

// Constructor with multiple returns
func NewMultipleReturnComponent() (*SimpleComponent, error) {
	return &SimpleComponent{}, nil
}

// Component with interface parameter
type InterfaceConsumerComponent struct{}

func NewInterfaceConsumerComponent() *InterfaceConsumerComponent {
	return &InterfaceConsumerComponent{}
}

func (i *InterfaceConsumerComponent) Apply(input any) {}

// Component with complex slice type
type ComplexSliceComponent struct{}

func NewComplexSliceComponent() *ComplexSliceComponent {
	return &ComplexSliceComponent{}
}

func (c *ComplexSliceComponent) Apply(inputs []*SimpleComponent) {}

// Component with context parameter
type ContextComponent struct{}

func NewContextComponent() *ContextComponent {
	return &ContextComponent{}
}

func (c *ContextComponent) Apply(ctx context.Context) string {
	return "context_result"
}

type compositeNoImpl struct{}

func newCompositeNoImpl() *compositeNoImpl {
	return &compositeNoImpl{}
}

func (compositeNoImpl) Run() {}

type compositeImpl struct{}

func newCompositeImpl() *compositeImpl {
	return &compositeImpl{}
}

func (compositeImpl) Run() {}

func (compositeImpl) String() string {
	return "My Composite"
}

func Test_analyzeConstructor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc            string
		constructor     any
		wantErr         assert.ErrorAssertionFunc
		errContains     string
		wantName        string
		wantProduces    reflect.Type
		wantConsumes    []reflect.Type
		wantIsSliceType bool
	}{
		// Valid cases
		{
			desc:        "simple component with Apply method - no params, no return",
			constructor: NewSimpleComponent,
			wantErr:     assert.NoError,
			wantName:    "SimpleComponent",
		},
		{
			desc:         "producer component - no params, single return",
			constructor:  NewProducerComponent,
			wantErr:      assert.NoError,
			wantName:     "service.NewProducerComponent",
			wantProduces: reflect.TypeOf(""),
		},
		{
			desc:         "multi-return component - no params, multiple returns",
			constructor:  NewMultiReturnComponent,
			wantErr:      assert.NoError,
			wantName:     "service.NewMultiReturnComponent",
			wantProduces: reflect.TypeOf(""),
		},
		{
			desc:         "consumer component - single param, no return",
			constructor:  NewConsumerComponent,
			wantErr:      assert.NoError,
			wantName:     "service.NewConsumerComponent",
			wantConsumes: []reflect.Type{reflect.TypeOf("")},
		},
		{
			desc:         "multi-consumer component - multiple params, no return",
			constructor:  NewMultiConsumerComponent,
			wantErr:      assert.NoError,
			wantName:     "service.NewMultiConsumerComponent",
			wantConsumes: []reflect.Type{reflect.TypeOf(""), reflect.TypeOf(0)},
		},
		{
			desc:            "slice consumer component - slice param, no return",
			constructor:     NewSliceConsumerComponent,
			wantErr:         assert.NoError,
			wantName:        "service.NewSliceConsumerComponent",
			wantConsumes:    []reflect.Type{reflect.TypeOf([]string{})},
			wantIsSliceType: true,
		},
		{
			desc:            "mixed component - regular and slice params with return",
			constructor:     NewMixedComponent,
			wantErr:         assert.NoError,
			wantName:        "service.NewMixedComponent",
			wantProduces:    reflect.TypeOf(""),
			wantConsumes:    []reflect.Type{reflect.TypeOf(""), reflect.TypeOf([]int{}), reflect.TypeOf((*context.Context)(nil)).Elem()},
			wantIsSliceType: true,
		},
		{
			desc:        "non-stringer component",
			constructor: NewNonStringerComponent,
			wantErr:     assert.NoError,
			wantName:    "service.NewNonStringerComponent",
		},
		{
			desc:         "interface consumer component",
			constructor:  NewInterfaceConsumerComponent,
			wantErr:      assert.NoError,
			wantName:     "service.NewInterfaceConsumerComponent",
			wantConsumes: []reflect.Type{reflect.TypeOf((*any)(nil)).Elem()},
		},
		{
			desc:            "complex slice component",
			constructor:     NewComplexSliceComponent,
			wantErr:         assert.NoError,
			wantName:        "service.NewComplexSliceComponent",
			wantConsumes:    []reflect.Type{reflect.TypeOf([]*SimpleComponent{})},
			wantIsSliceType: true,
		},
		{
			desc:         "context component",
			constructor:  NewContextComponent,
			wantErr:      assert.NoError,
			wantName:     "service.NewContextComponent",
			wantProduces: reflect.TypeOf(""),
			wantConsumes: []reflect.Type{reflect.TypeOf((*context.Context)(nil)).Elem()},
		},

		// Error cases from isCallable
		{
			desc:        "nil constructor",
			constructor: nil,
			wantErr:     assert.Error,
			errContains: "cannot register nil component",
		},
		{
			desc:        "non-function constructor",
			constructor: "not a function",
			wantErr:     assert.Error,
			errContains: "component must be a callable function",
		},
		{
			desc:        "function with parameters",
			constructor: func(param string) *SimpleComponent { return &SimpleComponent{} },
			wantErr:     assert.Error,
			errContains: "non-zero required arguments",
		},

		// Error cases specific to analyzeConstructor
		{
			desc:        "component without Apply method",
			constructor: NewNoApplyComponent,
			wantErr:     assert.Error,
			errContains: "missing Apply method",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := analyzeConstructor(tc.constructor)

			if !tc.wantErr(t, err) {
				return
			}

			if err != nil {
				assert.ErrorContains(t, err, tc.errContains)
				return
			}

			// Validate successful results
			assert.Equal(t, tc.wantName, got.name)
			assert.Equal(t, tc.wantProduces, got.produces)
			assert.Equal(t, tc.wantConsumes, got.consumes)
			assert.Equal(t, tc.wantIsSliceType, got.isSliceType)
			assert.NotNil(t, got.component)
			assert.NotZero(t, got.runMethod)
		})
	}
}

func Test_analyzeConstructor_PanicCases(t *testing.T) {
	t.Parallel()

	t.Run("constructor that panics", func(t *testing.T) {
		assert.Panics(t, func() {
			_, _ = analyzeConstructor(NewPanicComponent)
		})
	})

	t.Run("constructor with no returns", func(t *testing.T) {
		assert.Panics(t, func() {
			_, _ = analyzeConstructor(NewNoReturnComponent)
		})
	})
}

func Test_analyzeConstructor_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("component implements fmt.Stringer", func(t *testing.T) {
		got, err := analyzeConstructor(NewSimpleComponent)
		require.NoError(t, err)
		assert.Equal(t, "SimpleComponent", got.name)
	})

	t.Run("component does not implement fmt.Stringer", func(t *testing.T) {
		got, err := analyzeConstructor(NewNonStringerComponent)
		require.NoError(t, err)
		assert.Equal(t, "service.NewNonStringerComponent", got.name)
	})

	t.Run("Apply method with multiple output parameters", func(t *testing.T) {
		got, err := analyzeConstructor(NewMultiReturnComponent)
		require.NoError(t, err)
		// Should only capture the first return type
		assert.Equal(t, reflect.TypeOf(""), got.produces)
	})

	t.Run("Apply method with no output parameters", func(t *testing.T) {
		got, err := analyzeConstructor(NewSimpleComponent)
		require.NoError(t, err)
		assert.Nil(t, got.produces)
	})

	t.Run("complex slice type analysis", func(t *testing.T) {
		got, err := analyzeConstructor(NewComplexSliceComponent)
		require.NoError(t, err)
		assert.True(t, got.isSliceType)
		expectedType := reflect.TypeOf([]*SimpleComponent{})
		assert.Equal(t, []reflect.Type{expectedType}, got.consumes)
	})

	t.Run("mixed parameter types with slice", func(t *testing.T) {
		got, err := analyzeConstructor(NewMixedComponent)
		require.NoError(t, err)
		assert.True(t, got.isSliceType)
		assert.Len(t, got.consumes, 3)
		assert.Equal(t, reflect.TypeOf(""), got.consumes[0])
		assert.Equal(t, reflect.TypeOf([]int{}), got.consumes[1])
		assert.Equal(t, reflect.TypeOf((*context.Context)(nil)).Elem(), got.consumes[2])
	})
}

func Test_componentName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc  string
		input any
		want  string
	}{
		{
			desc:  "type does not implement fmt.Stringer",
			input: newCompositeNoImpl,
			want:  "service.newCompositeNoImpl",
		},
		{
			desc:  "constructor implements fmt.Stringer",
			input: newCompositeImpl,
			want:  "My Composite",
		},
		{
			desc:  "anonymous constructor implements fmt.Stringer",
			input: func() compositeImpl { return compositeImpl{} },
			want:  "My Composite",
		},
		{
			desc:  "anonymous constructor of pointer to type implements fmt.Stringer",
			input: func() *compositeImpl { return &compositeImpl{} },
			want:  "My Composite",
		},
		{
			desc: "closure returns type that implements fmt.Stringer",
			input: func() func() *compositeImpl {
				return newCompositeImpl
			}(),
			want: "My Composite",
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			var got string
			assert.NotPanics(t, func() {
				val := reflect.ValueOf(tc.input)
				res := val.Call(nil)
				got = componentName(tc.input, reflect.TypeOf(res[0].Interface()))
			})
			assert.Equal(t, tc.want, got)
		})
	}
}

func Test_newConstructorRefs(t *testing.T) {
	t.Parallel()

	constructorSets := [][]any{
		// Kitchen-sink set of constructors and non-constructors.
		{
			newCompositeImpl,
			newCompositeNoImpl,
			"Hello, World!", // This is not a constructor, should be ignored generally.
			func() bool {
				return true // This is not a constructor, should be ignored generally.
			},
		},
		{
			newCompositeImpl,
			newCompositeNoImpl,
		},
		{
			newCompositeImpl,
		},
		{
			newCompositeNoImpl,
		},
		{
			"Hello, World!", // This is not a constructor, should be ignored generally.
		},
		{
			func() bool {
				return true // This is not a constructor, should be ignored generally.
			},
		},
	}

	seen := make(map[*string]int)
	for _, set := range constructorSets {
		refs := newConstructorRefs(set...)
		assert.NotNil(t, refs)
		assert.NotEmpty(t, refs)
		assert.Len(t, refs.Refs, len(set))
		seen[infra.ResourceID(t.Name(), refs)]++
	}
	for ref, count := range seen {
		assert.Equal(t, 1, count, "Constructor reference %s should be unique", ref)
	}
}

func Test_isCallable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc        string
		input       any
		wantErr     assert.ErrorAssertionFunc
		errContains string
	}{
		{
			desc:    "constructor function",
			input:   newCompositeImpl,
			wantErr: assert.NoError,
		},
		{
			desc:    "non-constructor function",
			input:   func() bool { return true },
			wantErr: assert.NoError,
		},
		{
			desc:        "non-callable type",
			input:       "Hello, World!",
			wantErr:     assert.Error,
			errContains: "have string",
		},
		{
			desc:        "nil input",
			wantErr:     assert.Error,
			errContains: "nil component",
		},
		{
			desc:        "nil input with type assertion",
			input:       (*compositeNoImpl)(nil), // This is a pointer to a type that is not a function.
			wantErr:     assert.Error,
			errContains: "have *service.compositeNoImpl",
		},
		{
			desc:        "function has arguments",
			input:       func(int, string) *compositeImpl { return newCompositeImpl() },
			wantErr:     assert.Error,
			errContains: "non-zero required arguments",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			got := isCallable(tc.input)
			if tc.wantErr(t, got) && got != nil {
				assert.ErrorContains(t, got, tc.errContains)
			}
		})
	}
}

// Test types for dependencyGraph tests

// Shared data type for testing multiple producers
type SharedData struct {
	Value string
}

// First producer of SharedData
type ProducerA struct{}

func NewProducerA() *ProducerA {
	return &ProducerA{}
}

func (p *ProducerA) Apply() *SharedData {
	return &SharedData{Value: "from A"}
}

func (p *ProducerA) String() string {
	return "ProducerA"
}

// Second producer of SharedData
type ProducerB struct{}

func NewProducerB() *ProducerB {
	return &ProducerB{}
}

func (p *ProducerB) Apply() *SharedData {
	return &SharedData{Value: "from B"}
}

func (p *ProducerB) String() string {
	return "ProducerB"
}

// Consumer that expects a single SharedData (problematic with multiple producers)
type SingleConsumer struct{}

func NewSingleConsumer() *SingleConsumer {
	return &SingleConsumer{}
}

func (s *SingleConsumer) Apply(data *SharedData) error {
	return nil
}

func (s *SingleConsumer) String() string {
	return "SingleConsumer"
}

// Consumer that expects a slice of SharedData (correct way to handle multiple producers)
type SliceConsumer struct{}

func NewSliceConsumer() *SliceConsumer {
	return &SliceConsumer{}
}

func (s *SliceConsumer) Apply(data []*SharedData) error {
	return nil
}

func (s *SliceConsumer) String() string {
	return "SliceConsumer"
}

// Consumer with no dependencies
type NoDepsConsumer struct{}

func NewNoDepsConsumer() *NoDepsConsumer {
	return &NoDepsConsumer{}
}

func (n *NoDepsConsumer) Apply() error {
	return nil
}

func (n *NoDepsConsumer) String() string {
	return "NoDepsConsumer"
}

// Consumer that depends on itself (should be ignored)
type SelfDependentProducer struct{}

func NewSelfDependentProducer() *SelfDependentProducer {
	return &SelfDependentProducer{}
}

func (s *SelfDependentProducer) Apply(data *SharedData) *SharedData {
	return &SharedData{Value: "self-produced"}
}

func (s *SelfDependentProducer) String() string {
	return "SelfDependentProducer"
}

func Test_Composite_dependencyGraph(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc          string
		components    []AutoComponent
		wantErr       assert.ErrorAssertionFunc
		errContains   string
		wantEdgeCount map[string]int // number of dependencies for each component
	}{
		{
			desc: "valid - single producer and single consumer",
			components: []AutoComponent{
				mustAnalyzeConstructor(NewProducerA),
				mustAnalyzeConstructor(NewSingleConsumer),
			},
			wantErr: assert.NoError,
			wantEdgeCount: map[string]int{
				"ProducerA":      0,
				"SingleConsumer": 1,
			},
		},
		{
			desc: "valid - multiple producers with slice consumer",
			components: []AutoComponent{
				mustAnalyzeConstructor(NewProducerA),
				mustAnalyzeConstructor(NewProducerB),
				mustAnalyzeConstructor(NewSliceConsumer),
			},
			wantErr: assert.NoError,
			wantEdgeCount: map[string]int{
				"ProducerA":     0,
				"ProducerB":     0,
				"SliceConsumer": 2,
			},
		},
		{
			desc: "valid - consumer with no dependencies",
			components: []AutoComponent{
				mustAnalyzeConstructor(NewNoDepsConsumer),
			},
			wantErr: assert.NoError,
			wantEdgeCount: map[string]int{
				"NoDepsConsumer": 0,
			},
		},
		{
			desc: "valid - self-dependent component (dependency ignored)",
			components: []AutoComponent{
				mustAnalyzeConstructor(NewSelfDependentProducer),
			},
			wantErr: assert.NoError,
			wantEdgeCount: map[string]int{
				"SelfDependentProducer": 0,
			},
		},
		{
			desc: "error - single consumer with multiple producers",
			components: []AutoComponent{
				mustAnalyzeConstructor(NewProducerA),
				mustAnalyzeConstructor(NewProducerB),
				mustAnalyzeConstructor(NewSingleConsumer),
			},
			wantErr:     assert.Error,
			errContains: "consumes single *service.SharedData but multiple producers exist: [ProducerA ProducerB]",
		},
		{
			desc: "error - multiple single consumers with multiple producers",
			components: []AutoComponent{
				mustAnalyzeConstructor(NewProducerA),
				mustAnalyzeConstructor(NewProducerB),
				mustAnalyzeConstructor(NewSingleConsumer),
				mustAnalyzeConstructor(func() *SingleConsumer { return &SingleConsumer{} }), // Another single consumer
			},
			wantErr:     assert.Error,
			errContains: "consumes single *service.SharedData but multiple producers exist: [ProducerA ProducerB]",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			composite := &Composite{
				components:   tc.components,
				results:      make(map[reflect.Type]any),
				sliceResults: make(map[reflect.Type][]any),
			}

			edges, err := composite.dependencyGraph()

			if !tc.wantErr(t, err) {
				return
			}

			if err != nil {
				assert.ErrorContains(t, err, tc.errContains)
				return
			}

			// Validate successful results
			assert.NotNil(t, edges)

			// Check edge counts
			for componentName, expectedCount := range tc.wantEdgeCount {
				actualCount := len(edges[componentName])
				assert.Equal(t, expectedCount, actualCount,
					"component %s should have %d dependencies, got %d", componentName, expectedCount, actualCount)
			}

			// Ensure all components are present in edges
			for _, comp := range tc.components {
				assert.Contains(t, edges, comp.name, "edges should contain component %s", comp.name)
			}
		})
	}
}

func Test_Composite_dependencyGraph_ErrorMessages(t *testing.T) {
	t.Parallel()

	t.Run("error message includes correct component names and suggestion", func(t *testing.T) {
		composite := &Composite{
			components: []AutoComponent{
				mustAnalyzeConstructor(NewProducerA),
				mustAnalyzeConstructor(NewProducerB),
				mustAnalyzeConstructor(NewSingleConsumer),
			},
			results:      make(map[reflect.Type]any),
			sliceResults: make(map[reflect.Type][]any),
		}

		_, err := composite.dependencyGraph()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "SingleConsumer")
		assert.Contains(t, err.Error(), "*service.SharedData")
		assert.Contains(t, err.Error(), "[ProducerA ProducerB]")
		assert.Contains(t, err.Error(), "Consider changing \"SingleConsumer\" to consume []*service.SharedData")
		assert.Contains(t, err.Error(), "only the last produced item will be used")
	})
}

func Test_Composite_dependencyGraph_Integration(t *testing.T) {
	t.Parallel()

	t.Run("integration with Apply method - error propagation", func(t *testing.T) {
		composite := &Composite{
			components: []AutoComponent{
				mustAnalyzeConstructor(NewProducerA),
				mustAnalyzeConstructor(NewProducerB),
				mustAnalyzeConstructor(NewSingleConsumer),
			},
			results:      make(map[reflect.Type]any),
			sliceResults: make(map[reflect.Type][]any),
		}
		// Set the executor to the composite itself
		composite.executor = composite

		err := composite.Apply(t.Context())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "building dependency graph")
		assert.Contains(t, err.Error(), "consumes single *service.SharedData but multiple producers exist")
	})

	t.Run("integration with Apply method - success case", func(t *testing.T) {
		composite := &Composite{
			components: []AutoComponent{
				mustAnalyzeConstructor(NewProducerA),
				mustAnalyzeConstructor(NewSliceConsumer),
			},
			results:      make(map[reflect.Type]any),
			sliceResults: make(map[reflect.Type][]any),
		}
		// Set the executor to the composite itself
		composite.executor = composite

		err := composite.Apply(t.Context())
		// Note: This might fail at execution time due to missing dependencies,
		// but it should pass the dependency graph validation
		if err != nil {
			assert.NotContains(t, err.Error(), "failed to build dependency graph")
		}
	})
}

// mustAnalyzeConstructor is a helper function that calls analyzeConstructor and panics on error
// This is useful for test setup where we know the constructor should be valid
func mustAnalyzeConstructor(ctor any) AutoComponent {
	component, err := analyzeConstructor(ctor)
	if err != nil {
		panic(fmt.Sprintf("failed to analyze constructor: %v", err))
	}
	return component
}

// Mock component executor for testing executeGraph in isolation
type MockComponentExecutor struct {
	executionOrder []string
	executeFunc    func(comp AutoComponent) error
	mu             sync.Mutex
}

func NewMockComponentExecutor() *MockComponentExecutor {
	return &MockComponentExecutor{
		executionOrder: []string{},
		executeFunc:    func(comp AutoComponent) error { return nil },
	}
}

func (m *MockComponentExecutor) ExecuteComponent(comp AutoComponent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.executionOrder = append(m.executionOrder, comp.name)
	return m.executeFunc(comp)
}

func (m *MockComponentExecutor) GetExecutionOrder() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Return a copy to avoid race conditions
	order := make([]string, len(m.executionOrder))
	copy(order, m.executionOrder)
	return order
}

func (m *MockComponentExecutor) SetExecuteFunc(fn func(comp AutoComponent) error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executeFunc = fn
}

func Test_Composite_executeGraph(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc          string
		components    []AutoComponent
		edges         map[string][]string
		wantErr       assert.ErrorAssertionFunc
		errContains   string
		expectedOrder []string
		executorFunc  func(comp AutoComponent) error
	}{
		{
			desc:          "empty graph - no components",
			components:    []AutoComponent{},
			edges:         map[string][]string{},
			wantErr:       assert.NoError,
			expectedOrder: []string{},
		},
		{
			desc: "single component - no dependencies",
			components: []AutoComponent{
				mustAnalyzeConstructor(NewNoDepsConsumer),
			},
			edges: map[string][]string{
				"NoDepsConsumer": {},
			},
			wantErr:       assert.NoError,
			expectedOrder: []string{"NoDepsConsumer"},
		},
		{
			desc: "linear dependency chain - A -> B -> C",
			components: []AutoComponent{
				mustAnalyzeConstructor(NewProducerA),
				mustAnalyzeConstructor(NewSingleConsumer),
				mustAnalyzeConstructor(NewNoDepsConsumer),
			},
			edges: map[string][]string{
				"ProducerA":      {},
				"SingleConsumer": {"ProducerA"},
				"NoDepsConsumer": {"SingleConsumer"},
			},
			wantErr:       assert.NoError,
			expectedOrder: []string{"ProducerA", "SingleConsumer", "NoDepsConsumer"},
		},
		{
			desc: "diamond dependency - A -> B,C -> D",
			components: []AutoComponent{
				mustAnalyzeConstructor(NewProducerA),
				mustAnalyzeConstructor(NewProducerB),
				mustAnalyzeConstructor(NewSingleConsumer),
				mustAnalyzeConstructor(NewNoDepsConsumer),
			},
			edges: map[string][]string{
				"ProducerA":      {},
				"ProducerB":      {},
				"SingleConsumer": {"ProducerA"},
				"NoDepsConsumer": {"SingleConsumer", "ProducerB"},
			},
			wantErr:       assert.NoError,
			expectedOrder: []string{"ProducerA", "ProducerB", "SingleConsumer", "NoDepsConsumer"},
		},
		{
			desc: "complex graph with multiple dependency levels",
			components: []AutoComponent{
				mustAnalyzeConstructor(NewProducerA),
				mustAnalyzeConstructor(NewProducerB),
				mustAnalyzeConstructor(NewSingleConsumer),
				mustAnalyzeConstructor(NewSliceConsumer),
				mustAnalyzeConstructor(NewNoDepsConsumer),
			},
			edges: map[string][]string{
				"ProducerA":      {},
				"ProducerB":      {},
				"SingleConsumer": {"ProducerA"},
				"SliceConsumer":  {"ProducerA", "ProducerB"},
				"NoDepsConsumer": {"SingleConsumer"},
			},
			wantErr:       assert.NoError,
			expectedOrder: []string{"ProducerA", "ProducerB", "SingleConsumer", "SliceConsumer", "NoDepsConsumer"},
		},
		{
			desc: "parallel independent components",
			components: []AutoComponent{
				mustAnalyzeConstructor(NewProducerA),
				mustAnalyzeConstructor(NewProducerB),
				mustAnalyzeConstructor(NewNoDepsConsumer),
			},
			edges: map[string][]string{
				"ProducerA":      {},
				"ProducerB":      {},
				"NoDepsConsumer": {},
			},
			wantErr: assert.NoError,
			// Order can vary for independent components
			expectedOrder: []string{"ProducerA", "ProducerB", "NoDepsConsumer"},
		},
		{
			desc: "circular dependency - A -> B -> A",
			components: []AutoComponent{
				mustAnalyzeConstructor(NewProducerA),
				mustAnalyzeConstructor(NewProducerB),
			},
			edges: map[string][]string{
				"ProducerA": {"ProducerB"},
				"ProducerB": {"ProducerA"},
			},
			wantErr:     assert.Error,
			errContains: "circular dependency detected involving",
		},
		{
			desc: "circular dependency in complex graph - A -> B -> C -> A",
			components: []AutoComponent{
				mustAnalyzeConstructor(NewProducerA),
				mustAnalyzeConstructor(NewProducerB),
				mustAnalyzeConstructor(NewSingleConsumer),
			},
			edges: map[string][]string{
				"ProducerA":      {"SingleConsumer"},
				"ProducerB":      {},
				"SingleConsumer": {"ProducerB", "ProducerA"}, // This creates a cycle
			},
			wantErr:     assert.Error,
			errContains: "circular dependency detected involving",
		},
		{
			desc: "execution error from component",
			components: []AutoComponent{
				mustAnalyzeConstructor(NewProducerA),
				mustAnalyzeConstructor(NewSingleConsumer),
			},
			edges: map[string][]string{
				"ProducerA":      {},
				"SingleConsumer": {"ProducerA"},
			},
			wantErr:     assert.Error,
			errContains: "component execution failed",
			executorFunc: func(comp AutoComponent) error {
				if comp.name == "SingleConsumer" {
					return errors.New("component execution failed")
				}
				return nil
			},
		},
		{
			desc: "execution error propagates correctly",
			components: []AutoComponent{
				mustAnalyzeConstructor(NewProducerA),
				mustAnalyzeConstructor(NewProducerB),
				mustAnalyzeConstructor(NewSingleConsumer),
			},
			edges: map[string][]string{
				"ProducerA":      {},
				"ProducerB":      {"ProducerA"}, // This should fail, preventing SingleConsumer from running
				"SingleConsumer": {"ProducerB"},
			},
			wantErr:       assert.Error,
			errContains:   "dependency error",
			expectedOrder: []string{"ProducerA"}, // Only ProducerA should execute before error
			executorFunc: func(comp AutoComponent) error {
				if comp.name == "ProducerB" {
					return errors.New("dependency error")
				}
				return nil
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			// Create composite with mock executor
			mockExecutor := NewMockComponentExecutor()
			if tc.executorFunc != nil {
				mockExecutor.SetExecuteFunc(tc.executorFunc)
			}

			composite := &Composite{
				components:   tc.components,
				results:      make(map[reflect.Type]any),
				sliceResults: make(map[reflect.Type][]any),
				executor:     mockExecutor,
			}

			err := composite.executeGraph(tc.edges)

			if !tc.wantErr(t, err) {
				return
			}

			if err != nil {
				assert.ErrorContains(t, err, tc.errContains)
				return
			}

			// Validate execution order for successful cases
			actualOrder := mockExecutor.GetExecutionOrder()

			if len(tc.expectedOrder) > 0 {
				// For deterministic tests, check exact order
				if tc.desc == "parallel independent components" {
					// For parallel components, just check that all components were executed
					assert.ElementsMatch(t, tc.expectedOrder, actualOrder)
				} else {
					// For ordered dependencies, check exact order
					assert.Equal(t, tc.expectedOrder, actualOrder, "execution order should match expected")
				}
			}

			// Ensure all components were attempted to be executed (unless error occurred)
			if tc.executorFunc == nil {
				assert.Len(t, actualOrder, len(tc.components), "all components should be executed")
			}
		})
	}
}

func Test_Composite_executeGraph_TopologicalSort(t *testing.T) {
	t.Parallel()

	t.Run("verifies correct topological ordering", func(t *testing.T) {
		// Create a complex dependency graph to test topological sorting
		components := []AutoComponent{
			mustAnalyzeConstructor(NewProducerA),      // Level 0
			mustAnalyzeConstructor(NewProducerB),      // Level 0
			mustAnalyzeConstructor(NewSingleConsumer), // Level 1 (depends on ProducerA)
			mustAnalyzeConstructor(NewSliceConsumer),  // Level 1 (depends on ProducerA, ProducerB)
			mustAnalyzeConstructor(NewNoDepsConsumer), // Level 2 (depends on SingleConsumer)
		}

		edges := map[string][]string{
			"ProducerA":      {},
			"ProducerB":      {},
			"SingleConsumer": {"ProducerA"},
			"SliceConsumer":  {"ProducerA", "ProducerB"},
			"NoDepsConsumer": {"SingleConsumer"},
		}

		mockExecutor := NewMockComponentExecutor()
		composite := &Composite{
			components:   components,
			results:      make(map[reflect.Type]any),
			sliceResults: make(map[reflect.Type][]any),
			executor:     mockExecutor,
		}

		err := composite.executeGraph(edges)
		require.NoError(t, err)

		order := mockExecutor.GetExecutionOrder()

		// Verify topological constraints
		positionMap := make(map[string]int)
		for i, name := range order {
			positionMap[name] = i
		}

		// Check that dependencies come before dependents
		for component, deps := range edges {
			componentPos := positionMap[component]
			for _, dep := range deps {
				depPos := positionMap[dep]
				assert.Less(t, depPos, componentPos,
					"dependency %s should come before %s in execution order", dep, component)
			}
		}
	})
}

func Test_Composite_executeGraph_ErrorHandling(t *testing.T) {
	t.Parallel()

	t.Run("stops execution on first error", func(t *testing.T) {
		components := []AutoComponent{
			mustAnalyzeConstructor(NewProducerA),
			mustAnalyzeConstructor(NewProducerB),
			mustAnalyzeConstructor(NewSingleConsumer),
		}

		edges := map[string][]string{
			"ProducerA":      {},
			"ProducerB":      {},
			"SingleConsumer": {"ProducerA"},
		}

		mockExecutor := NewMockComponentExecutor()
		var executionCount int
		mockExecutor.SetExecuteFunc(func(comp AutoComponent) error {
			executionCount++
			if comp.name == "ProducerA" {
				return errors.New("execution failed")
			}
			return nil
		})

		composite := &Composite{
			components:   components,
			results:      make(map[reflect.Type]any),
			sliceResults: make(map[reflect.Type][]any),
			executor:     mockExecutor,
		}

		err := composite.executeGraph(edges)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "execution failed")

		// Should have attempted only the failing component
		assert.Equal(t, 1, executionCount)
	})

	t.Run("reports circular dependency with component name", func(t *testing.T) {
		components := []AutoComponent{
			mustAnalyzeConstructor(NewProducerA),
			mustAnalyzeConstructor(NewProducerB),
		}

		edges := map[string][]string{
			"ProducerA": {"ProducerB"},
			"ProducerB": {"ProducerA"},
		}

		mockExecutor := NewMockComponentExecutor()
		composite := &Composite{
			components:   components,
			results:      make(map[reflect.Type]any),
			sliceResults: make(map[reflect.Type][]any),
			executor:     mockExecutor,
		}

		err := composite.executeGraph(edges)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "circular dependency detected involving")
		// Should mention one of the components in the cycle
		assert.True(t,
			strings.Contains(err.Error(), "ProducerA") || strings.Contains(err.Error(), "ProducerB"),
			"error should mention component involved in circular dependency")
	})
}

func Test_Composite_executeGraph_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("handles component not in edges map", func(t *testing.T) {
		components := []AutoComponent{
			mustAnalyzeConstructor(NewProducerA),
			mustAnalyzeConstructor(NewProducerB), // This won't be in edges
		}

		edges := map[string][]string{
			"ProducerA": {},
			// ProducerB is missing from edges
		}

		mockExecutor := NewMockComponentExecutor()
		composite := &Composite{
			components:   components,
			results:      make(map[reflect.Type]any),
			sliceResults: make(map[reflect.Type][]any),
			executor:     mockExecutor,
		}

		err := composite.executeGraph(edges)
		require.NoError(t, err)

		// All components should still be executed
		order := mockExecutor.GetExecutionOrder()
		assert.ElementsMatch(t, []string{"ProducerA", "ProducerB"}, order)
	})

	t.Run("handles dependencies on non-existent components", func(t *testing.T) {
		components := []AutoComponent{
			mustAnalyzeConstructor(NewProducerA),
		}

		edges := map[string][]string{
			"ProducerA": {"NonExistentComponent"},
		}

		mockExecutor := NewMockComponentExecutor()
		composite := &Composite{
			components:   components,
			results:      make(map[reflect.Type]any),
			sliceResults: make(map[reflect.Type][]any),
			executor:     mockExecutor,
		}

		err := composite.executeGraph(edges)
		require.NoError(t, err) // Should not fail, just skip non-existent dependency

		order := mockExecutor.GetExecutionOrder()
		assert.Equal(t, []string{"ProducerA"}, order)
	})
}

func Test_Composite_executeGraph_Integration(t *testing.T) {
	t.Parallel()

	t.Run("integration with real ComponentExecutor", func(t *testing.T) {
		// Test that executeGraph works with the real Composite.ExecuteComponent
		components := []AutoComponent{
			mustAnalyzeConstructor(NewNoDepsConsumer),
		}

		edges := map[string][]string{
			"NoDepsConsumer": {},
		}

		composite := &Composite{
			components:   components,
			results:      make(map[reflect.Type]any),
			sliceResults: make(map[reflect.Type][]any),
		}
		// Set the executor to the composite itself after initialization
		composite.executor = composite

		// This should work without mocking
		err := composite.executeGraph(edges)
		assert.NoError(t, err)
	})
}

// Test types for ExecuteComponent tests

// Interface for testing interface parameter handling
type TestService interface {
	GetValue() string
}

// Concrete implementation of TestService
type ConcreteTestService struct {
	value string
}

func NewConcreteTestService() *ConcreteTestService {
	return &ConcreteTestService{value: "concrete_service"}
}

func (c *ConcreteTestService) Apply() TestService {
	return c
}

func (c *ConcreteTestService) GetValue() string {
	return c.value
}

func (c *ConcreteTestService) String() string {
	return "ConcreteTestService"
}

// Component that consumes an interface for ExecuteComponent tests
type TestInterfaceConsumerComponent struct{}

func NewTestInterfaceConsumerComponent() *TestInterfaceConsumerComponent {
	return &TestInterfaceConsumerComponent{}
}

func (i *TestInterfaceConsumerComponent) Apply(service TestService) string {
	return "consumed: " + service.GetValue()
}

func (i *TestInterfaceConsumerComponent) String() string {
	return "TestInterfaceConsumerComponent"
}

// Component that returns an error
type ErrorProducerComponent struct {
	shouldError bool
}

func NewErrorProducerComponent(shouldError bool) func() *ErrorProducerComponent {
	return func() *ErrorProducerComponent {
		return &ErrorProducerComponent{shouldError: shouldError}
	}
}

func (e *ErrorProducerComponent) Apply() error {
	if e.shouldError {
		return assert.AnError
	}
	return nil
}

func (e *ErrorProducerComponent) String() string {
	return "ErrorProducerComponent"
}

// Component with multiple return values
type MultiReturnProducerComponent struct{}

func NewMultiReturnProducerComponent() *MultiReturnProducerComponent {
	return &MultiReturnProducerComponent{}
}

func (m *MultiReturnProducerComponent) Apply() (string, int, error) {
	return "multi", 42, nil
}

func (m *MultiReturnProducerComponent) String() string {
	return "MultiReturnProducerComponent"
}

// Component that produces multiple types
type (
	TypeAComponent struct{}
	TypeBComponent struct{}
)

func NewTypeAComponent() *TypeAComponent {
	return &TypeAComponent{}
}

func (t *TypeAComponent) Apply() string {
	return "typeA"
}

func (t *TypeAComponent) String() string {
	return "TypeAComponent"
}

func NewTypeBComponent() *TypeBComponent {
	return &TypeBComponent{}
}

func (t *TypeBComponent) Apply() int {
	return 123
}

func (t *TypeBComponent) String() string {
	return "TypeBComponent"
}

// Component that consumes multiple types
type MultiTypeConsumerComponent struct{}

func NewMultiTypeConsumerComponent() *MultiTypeConsumerComponent {
	return &MultiTypeConsumerComponent{}
}

func (m *MultiTypeConsumerComponent) Apply(s string, i int) string {
	return fmt.Sprintf("consumed: %s, %d", s, i)
}

func (m *MultiTypeConsumerComponent) String() string {
	return "MultiTypeConsumerComponent"
}

func Test_Composite_ExecuteComponent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc              string
		component         AutoComponent
		prePopulateTypes  map[reflect.Type]any
		prePopulateSlices map[reflect.Type][]any
		wantErr           assert.ErrorAssertionFunc
		errContains       string
		wantResult        any
		wantResultType    reflect.Type
		checkSliceResults bool
		expectedSliceLen  int
	}{
		{
			desc:      "component with no parameters - no return",
			component: mustAnalyzeConstructor(NewNoDepsConsumer),
			wantErr:   assert.NoError,
		},
		{
			desc:           "component with no parameters - single return",
			component:      mustAnalyzeConstructor(NewProducerA),
			wantErr:        assert.NoError,
			wantResult:     &SharedData{Value: "from A"},
			wantResultType: reflect.TypeOf(&SharedData{}),
		},
		{
			desc:      "component with single parameter",
			component: mustAnalyzeConstructor(NewSingleConsumer),
			prePopulateTypes: map[reflect.Type]any{
				reflect.TypeOf(&SharedData{}): &SharedData{Value: "test"},
			},
			wantErr: assert.NoError,
		},
		{
			desc:      "component with multiple parameters",
			component: mustAnalyzeConstructor(NewMultiTypeConsumerComponent),
			prePopulateTypes: map[reflect.Type]any{
				reflect.TypeOf(""): "test_string",
				reflect.TypeOf(0):  42,
			},
			wantErr:        assert.NoError,
			wantResult:     "consumed: test_string, 42",
			wantResultType: reflect.TypeOf(""),
		},
		{
			desc:      "component with slice parameter",
			component: mustAnalyzeConstructor(NewSliceConsumer),
			prePopulateSlices: map[reflect.Type][]any{
				reflect.TypeOf(&SharedData{}): {
					&SharedData{Value: "item1"},
					&SharedData{Value: "item2"},
				},
			},
			wantErr: assert.NoError,
		},
		{
			desc:      "component with slice parameter - empty slice",
			component: mustAnalyzeConstructor(NewSliceConsumer),
			wantErr:   assert.NoError,
		},
		{
			desc:      "component with interface parameter",
			component: mustAnalyzeConstructor(NewTestInterfaceConsumerComponent),
			prePopulateTypes: map[reflect.Type]any{
				reflect.TypeOf(&ConcreteTestService{}): &ConcreteTestService{value: "interface_test"},
			},
			wantErr:        assert.NoError,
			wantResult:     "consumed: interface_test",
			wantResultType: reflect.TypeOf(""),
		},
		{
			desc:        "component that returns error",
			component:   mustAnalyzeConstructor(NewErrorProducerComponent(true)),
			wantErr:     assert.Error,
			errContains: "ErrorProducerComponent: assert.AnError",
		},
		{
			desc:           "component with multiple return values",
			component:      mustAnalyzeConstructor(NewMultiReturnProducerComponent),
			wantErr:        assert.NoError,
			wantResult:     "multi",
			wantResultType: reflect.TypeOf(""),
		},
		{
			desc:        "missing dependency error",
			component:   mustAnalyzeConstructor(NewSingleConsumer),
			wantErr:     assert.Error,
			errContains: "no registered component provides missing dependency *service.SharedData",
		},
		{
			desc:        "missing interface dependency error",
			component:   mustAnalyzeConstructor(NewTestInterfaceConsumerComponent),
			wantErr:     assert.Error,
			errContains: "TestInterfaceConsumerComponent: missing dependency service.TestService",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			composite := &Composite{
				results:      make(map[reflect.Type]any),
				sliceResults: make(map[reflect.Type][]any),
			}

			// Pre-populate results if specified
			if tc.prePopulateTypes != nil {
				for typ, value := range tc.prePopulateTypes {
					composite.results[typ] = value
				}
			}

			// Pre-populate slice results if specified
			if tc.prePopulateSlices != nil {
				for typ, values := range tc.prePopulateSlices {
					composite.sliceResults[typ] = values
				}
			}

			err := composite.ExecuteComponent(tc.component)
			if !tc.wantErr(t, err) {
				return
			}

			if err != nil {
				assert.ErrorContains(t, err, tc.errContains)
				return
			}

			// Check stored results
			if tc.wantResultType != nil {
				storedValue, exists := composite.results[tc.wantResultType]
				assert.True(t, exists, "result should be stored for type %s", tc.wantResultType)
				assert.Equal(t, tc.wantResult, storedValue)

				// Also check slice results
				sliceValues, sliceExists := composite.sliceResults[tc.wantResultType]
				assert.True(t, sliceExists, "slice result should be stored for type %s", tc.wantResultType)
				assert.Contains(t, sliceValues, tc.wantResult)
			}

			if tc.checkSliceResults {
				assert.Len(t, composite.sliceResults, tc.expectedSliceLen)
			}
		})
	}
}

func Test_Composite_ExecuteComponent_ThreadSafety(t *testing.T) {
	t.Parallel()

	t.Run("concurrent access to results", func(t *testing.T) {
		composite := &Composite{
			results:      make(map[reflect.Type]any),
			sliceResults: make(map[reflect.Type][]any),
		}

		// Pre-populate some data
		composite.results[reflect.TypeOf(&SharedData{})] = &SharedData{Value: "concurrent"}

		var wg sync.WaitGroup
		numGoroutines := 10

		// Run multiple components concurrently
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				component := mustAnalyzeConstructor(NewSingleConsumer)
				err := composite.ExecuteComponent(component)
				assert.NoError(t, err)
			}()
		}

		wg.Wait()
	})
}

func Test_Composite_ExecuteComponent_InterfaceHandling(t *testing.T) {
	t.Parallel()

	t.Run("finds correct concrete implementation for interface", func(t *testing.T) {
		composite := &Composite{
			results:      make(map[reflect.Type]any),
			sliceResults: make(map[reflect.Type][]any),
		}

		// Store a concrete implementation
		concreteService := &ConcreteTestService{value: "interface_impl"}
		composite.results[reflect.TypeOf(concreteService)] = concreteService

		// Execute component that needs the interface
		component := mustAnalyzeConstructor(NewTestInterfaceConsumerComponent)
		err := composite.ExecuteComponent(component)

		require.NoError(t, err)

		// Check that the result was stored
		result, exists := composite.results[reflect.TypeOf("")]
		assert.True(t, exists)
		assert.Equal(t, "consumed: interface_impl", result)
	})

	t.Run("multiple implementations - uses first found", func(t *testing.T) {
		composite := &Composite{
			results:      make(map[reflect.Type]any),
			sliceResults: make(map[reflect.Type][]any),
		}

		// Store implementation
		impl1 := &ConcreteTestService{value: "impl1"}

		// Note: In a real scenario, multiple different types would implement the same interface
		// but for this test, we'll use one concrete type
		composite.results[reflect.TypeOf(impl1)] = impl1

		// Execute component that needs the interface
		component := mustAnalyzeConstructor(NewTestInterfaceConsumerComponent)
		err := composite.ExecuteComponent(component)

		require.NoError(t, err)

		// Should use the first implementation found
		result, exists := composite.results[reflect.TypeOf("")]
		assert.True(t, exists)
		assert.Contains(t, result, "consumed: impl1")
	})
}

func Test_Composite_ExecuteComponent_ErrorHandling(t *testing.T) {
	t.Parallel()

	t.Run("component method with invalid arguments", func(t *testing.T) {
		// Create a component with a valid method but no produces type
		// This simulates a component that might have issues during execution
		component := mustAnalyzeConstructor(NewNoDepsConsumer)

		// Modify the component to have invalid state
		invalidComponent := AutoComponent{
			component: component.component,
			name:      "InvalidComponent",
			runMethod: component.runMethod,
			produces:  nil, // This is valid - some components don't produce anything
		}

		composite := &Composite{
			results:      make(map[reflect.Type]any),
			sliceResults: make(map[reflect.Type][]any),
		}

		// This should execute successfully even with nil produces
		err := composite.ExecuteComponent(invalidComponent)
		assert.NoError(t, err)
	})

	t.Run("error in return value", func(t *testing.T) {
		component := mustAnalyzeConstructor(NewErrorProducerComponent(true))
		composite := &Composite{
			results:      make(map[reflect.Type]any),
			sliceResults: make(map[reflect.Type][]any),
		}

		err := composite.ExecuteComponent(component)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ErrorProducerComponent")
		assert.Contains(t, err.Error(), "assert.AnError")
	})

	t.Run("no error when error return is nil", func(t *testing.T) {
		component := mustAnalyzeConstructor(NewErrorProducerComponent(false))
		composite := &Composite{
			results:      make(map[reflect.Type]any),
			sliceResults: make(map[reflect.Type][]any),
		}

		err := composite.ExecuteComponent(component)
		assert.NoError(t, err)
	})
}

func Test_Composite_ExecuteComponent_ResultStorage(t *testing.T) {
	t.Parallel()

	t.Run("stores result in both results and sliceResults", func(t *testing.T) {
		component := mustAnalyzeConstructor(NewProducerA)
		composite := &Composite{
			results:      make(map[reflect.Type]any),
			sliceResults: make(map[reflect.Type][]any),
		}

		err := composite.ExecuteComponent(component)
		require.NoError(t, err)

		expectedType := reflect.TypeOf(&SharedData{})

		// Check regular results
		result, exists := composite.results[expectedType]
		assert.True(t, exists)
		assert.Equal(t, "from A", result.(*SharedData).Value)

		// Check slice results
		sliceResult, sliceExists := composite.sliceResults[expectedType]
		assert.True(t, sliceExists)
		assert.Len(t, sliceResult, 1)
		assert.Equal(t, "from A", sliceResult[0].(*SharedData).Value)
	})

	t.Run("appends to existing slice results", func(t *testing.T) {
		composite := &Composite{
			results:      make(map[reflect.Type]any),
			sliceResults: make(map[reflect.Type][]any),
		}

		expectedType := reflect.TypeOf(&SharedData{})

		// Pre-populate slice results
		composite.sliceResults[expectedType] = []any{
			&SharedData{Value: "existing"},
		}

		// Execute component that produces the same type
		component := mustAnalyzeConstructor(NewProducerA)
		err := composite.ExecuteComponent(component)
		require.NoError(t, err)

		// Check that it was appended
		sliceResult := composite.sliceResults[expectedType]
		assert.Len(t, sliceResult, 2)
		assert.Equal(t, "existing", sliceResult[0].(*SharedData).Value)
		assert.Equal(t, "from A", sliceResult[1].(*SharedData).Value)
	})

	t.Run("no storage when component produces nothing", func(t *testing.T) {
		// Use SimpleComponent which has Apply() with no return value
		component := mustAnalyzeConstructor(NewSimpleComponent)
		composite := &Composite{
			results:      make(map[reflect.Type]any),
			sliceResults: make(map[reflect.Type][]any),
		}

		err := composite.ExecuteComponent(component)
		require.NoError(t, err)

		// Should have no stored results because Apply() returns nothing
		assert.Empty(t, composite.results)
		assert.Empty(t, composite.sliceResults)
	})
}

// Test types for ChartFactory tests

// Props implementation for testing ChartFactory
type TestChartProps struct {
	Name string
}

func (p *TestChartProps) Validate(ctx context.Context) error {
	if p.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

// Component that uses ChartFactory
type ChartFactoryConsumerComponent struct{}

func NewChartFactoryConsumerComponent() *ChartFactoryConsumerComponent {
	return &ChartFactoryConsumerComponent{}
}

func (c *ChartFactoryConsumerComponent) Apply(factory ChartFactory) string {
	props := &TestChartProps{Name: "test-chart"}
	chart := factory.CreateChart("TestComponent", props)
	// In a real scenario, you'd use the chart to create resources
	_ = chart
	return "chart created successfully"
}

func (c *ChartFactoryConsumerComponent) String() string {
	return "ChartFactoryConsumerComponent"
}

// Component that provides a props object
type PropsProviderComponent struct{}

func NewPropsProviderComponent() *PropsProviderComponent {
	return &PropsProviderComponent{}
}

func (p *PropsProviderComponent) Apply() *TestChartProps {
	return &TestChartProps{Name: "provided-props"}
}

func (p *PropsProviderComponent) String() string {
	return "PropsProviderComponent"
}

func Test_ChartFactory(t *testing.T) {
	t.Parallel()

	t.Run("ChartFactory interface is properly defined", func(t *testing.T) {
		// Test that the interface has the expected methods
		ctx := t.Context()
		factory := &chartFactory{}

		// Ensure it implements ChartFactory interface
		var _ ChartFactory = factory

		// Test Apply method returns the factory itself
		returnedFactory := factory.Apply(ctx)
		assert.Equal(t, factory, returnedFactory)
	})

	t.Run("ChartFactory is injected correctly", func(t *testing.T) {
		// Test that the chartFactory can be analyzed and used in DI

		component, err := analyzeConstructor(newChartFactory)
		require.NoError(t, err)

		// Should produce ChartFactory interface type
		expectedType := reflect.TypeOf((*ChartFactory)(nil)).Elem()
		assert.Equal(t, expectedType, component.produces)
		assert.Equal(t, "sdk.composite.builtin.chartFactory", component.name)
	})
}

func Test_ChartFactory_Integration(t *testing.T) {
	t.Parallel()

	t.Run("component can declare ChartFactory dependency", func(t *testing.T) {
		// Test that components can declare ChartFactory as a dependency
		component, err := analyzeConstructor(NewChartFactoryConsumerComponent)
		require.NoError(t, err)

		// Should consume ChartFactory interface
		expectedType := reflect.TypeOf((*ChartFactory)(nil)).Elem()
		assert.Contains(t, component.consumes, expectedType)
		assert.Equal(t, "ChartFactoryConsumerComponent", component.name)
	})

	t.Run("ChartFactory dependency injection works end-to-end", func(t *testing.T) {
		// Create a mock factory that doesn't actually create charts
		mockFactory := &mockChartFactory{}

		// Create components that demonstrate the ChartFactory usage
		components := []AutoComponent{
			// Provider of ChartFactory
			{
				component: mockFactory,
				name:      "MockChartFactory",
				runMethod: reflect.TypeOf(mockFactory).Method(0), // Apply method
				produces:  reflect.TypeOf((*ChartFactory)(nil)).Elem(),
			},
			mustAnalyzeConstructor(NewChartFactoryConsumerComponent),
		}

		composite := &Composite{
			components:   components,
			results:      make(map[reflect.Type]any),
			sliceResults: make(map[reflect.Type][]any),
		}
		composite.executor = composite

		// Execute the factory provider first
		factoryComponent := components[0]
		err := composite.ExecuteComponent(factoryComponent)
		require.NoError(t, err)

		// Verify ChartFactory was stored
		factoryResult, exists := composite.results[reflect.TypeOf((*ChartFactory)(nil)).Elem()]
		assert.True(t, exists)
		assert.NotNil(t, factoryResult)

		// Execute the consumer component
		consumerComponent := components[1]
		err = composite.ExecuteComponent(consumerComponent)
		require.NoError(t, err)

		// Verify the consumer successfully used the factory
		result, exists := composite.results[reflect.TypeOf("")]
		assert.True(t, exists)
		assert.Equal(t, "chart created successfully", result)

		// Verify the mock factory was called
		assert.True(t, mockFactory.createChartCalled)
		assert.Equal(t, "TestComponent", mockFactory.lastResourceName)
	})
}

// mockChartFactory implements ChartFactory for testing without requiring CDK8s context
type mockChartFactory struct {
	createChartCalled bool
	lastResourceName  string
	lastProps         port.Validator
}

func (m *mockChartFactory) CreateChart(resourceName string, props port.Validator) cdk8s.Chart {
	m.createChartCalled = true
	m.lastResourceName = resourceName
	m.lastProps = props
	// Return a mock chart - in real usage this would be a proper CDK8s chart
	return nil // This is fine for testing the DI mechanism
}

func (m *mockChartFactory) Apply() ChartFactory {
	return m
}
