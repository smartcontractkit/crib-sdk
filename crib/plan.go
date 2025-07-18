package crib

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/samber/lo"

	"github.com/smartcontractkit/crib-sdk/internal/adapter/filehandler"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
	"github.com/smartcontractkit/crib-sdk/internal/core/port"
	"github.com/smartcontractkit/crib-sdk/internal/core/service"
	"github.com/smartcontractkit/crib-sdk/internal/core/service/iresolver"
)

type (
	// Plans represents a set of Plans, usually Child Plans, that will be applied
	// as part of the parent plan.
	Plans = []port.Planner

	ComponentFuncs = []port.ComponentFunc

	// A Plan is created by a call to NewPlan and is used to create an application release plan.
	// It represents an intention to apply a set of resources in the prescribed order.
	//
	// The Plan represents a directed acyclic graph (DAG) of components and child plans.
	// When ready to Build, the Plan can be built by calling the Build() method, which resolves
	// the DAG and returns a new plan with the child plans resolved.
	Plan struct {
		// name is the name of this plan. The name must be globally unique within Plans known to cribctl.
		// The name is used in the Plan Registry so that it can be applied with cribctl via
		// `cribctl plan apply <plan-name>`.
		//
		// When including other plans as dependencies, the caller can include it either by name
		// contrib.Plan("plan-name"), or by its function that returns the plan, e.g. examplev1.Plan().
		name string
		// namespace is the primary target Kubernetes namespace for the plan on the target cluster.
		// The namespace will NOT be created if it does not exist.
		namespace string
		// components is a list of components that are part of the plan.
		components []ComponentFunc
		// childPlans is a list of dependent plans that are part of the plan.
		childPlans []*Plan
		// childFuncs is only used internally before DAG resolution.
		childFuncs []func() *Plan
		// resolvers is a list of resolvers that are part of the plan.
		resolvers []cdk8s.IResolver
	}

	PlanState struct {
		results *service.PlanState
	}

	// PlanOpt is a function that modifies a Plan, resolved during the Build process.
	PlanOpt func(*Plan)
)

// NewPlan creates a new intent to actuate a set of resources in a specific order.
// The name of a plan must be globally unique so that it can be applied by cribctl.
// Plans may depend on other plans and can contain any number of Composite or Scalar components.
// A Plan represents a set of resources that are intended to be applied in a specific order.
//
// Child plans are executed before the parent plan. Components within a child are applied in their defined order.
// Recommended use cases for Child plans are things like Bootstrapping services and features.
//
// Example:
//
//	p := crib.NewPlan("my-plan",
//		crib.Namespace("my-ns"),
//		crib.AddPlan(bootstrapv1.Plan), // Directly including a Plan.
//		crib.AddPlan(contrib.Plan("examplev1")), // Importing a plan from the registry.
//		crib.ComponentSet(
//			anvilv1.Component(anvilv1.Props{ChainID: 50}, anvilv1.UseIngress),
//		),
//	)
//	err := p.Apply(context.Background())
func NewPlan(name string, opts ...PlanOpt) *Plan {
	plan := &Plan{name: name}
	for _, opt := range opts {
		opt(plan)
	}
	if plan.namespace == "" {
		plan.namespace = domain.DefaultNamespace // Ensure a default namespace is set.
	}
	return plan
}

// Plan returns a function that returns the plan itself. This is useful for passing the plan
// as a dependency to other plans or for use in the Plan Registry.
func (p *Plan) Plan() func() *Plan {
	return func() *Plan {
		return p
	}
}

// Name returns the name of the plan. The name must be globally unique within Plans known to cribctl.
// The name is used in the Plan Registry so that it can be applied with cribctl via
// `cribctl plan apply <plan-name>`.
//
// When including other plans as dependencies, the caller can include it either by name
// contrib.Plan("plan-name"), or by its function that returns the plan, e.g. examplev1.Plan().
func (p *Plan) Name() string {
	return p.name
}

// Namespace returns the primary Kubernetes target namespace for the plan on the target cluster.
// The Namespace will NOT be created if it does not exist.
func (p *Plan) Namespace() string {
	return p.namespace
}

// Components returns a list of components that are part of the plan.
func (p *Plan) Components() ComponentFuncs {
	// Convert []ComponentFunc to []port.ComponentFunc
	return lo.Map(p.components, func(c ComponentFunc, _ int) port.ComponentFunc {
		return func(ctx context.Context) (port.Component, error) {
			return c(ctx)
		}
	})
}

// ChildPlans returns a list of dependent plans that are part of the plan.
func (p *Plan) ChildPlans() Plans {
	// Convert []*Plan to []port.Planner
	return lo.Map(p.childPlans, func(plan *Plan, _ int) port.Planner {
		return plan
	})
}

// Resolvers returns a list of resolvers that are part of the plan.
func (p *Plan) Resolvers() []cdk8s.IResolver {
	return iresolver.Resolvers(p.resolvers)
}

// AddPlan is a PlanOpt that adds a plan to the list of dependencies for the given plan.
// The childPlan can be included directly via its Plan() method or imported by name via
// the Plan Registry:
//
//	AddPlan(examplev1.Plan) // Directly via its method.
//	AddPlan(contrib.Plan("examplev1")) // Import via name.
//
// Note: A child plan may not indirectly depend on a parent plan. If the runtime
// detects a cycle, it will panic.
func AddPlan(childPlan func() *Plan) PlanOpt {
	return func(plan *Plan) {
		plan.childFuncs = append(plan.childFuncs, childPlan)
	}
}

// Namespace sets the target namespace for the plan for the target cluster. It also
// instructs the plan runtime to create the namespace if it does not exist.
// This option panics if it is called multiple times.
func Namespace(ns string) PlanOpt {
	return func(p *Plan) {
		if p.namespace != "" {
			panic("Namespace may only be declared once per Plan.")
		}
		p.namespace = ns
	}
}

// ComponentSet adds the components to add to the Plan. Components will be applied in the order they are added.
// Invoking this method multiple times will append the components to the existing list.
func ComponentSet(cs ...ComponentFunc) PlanOpt {
	return func(p *Plan) {
		p.components = append(p.components, cs...)
	}
}

// manifestResolvers appends the provided resolvers to the plan.
// Resolvers are an advanced mechanism for modifying a manifest before it is synthesized and applied.
func manifestResolvers(resolvers ...cdk8s.IResolver) PlanOpt {
	return func(p *Plan) {
		p.resolvers = append(p.resolvers, resolvers...)
	}
}

// ImagePullSecrets is a Resolver that ensures that the provided secret names are added as imagePullSecrets
// to all components that support it.
// The secrets must already exist in the cluster, or be created by the plan. The resolver
// will not create the secrets.
//
// Example:
//
//	plan := crib.NewPlan("my-plan",
//		crib.ImagePullSecrets("my-secret"),
//	)
func ImagePullSecrets(secrets ...string) PlanOpt {
	return manifestResolvers(
		iresolver.NewResolver(
			iresolver.ImagePullSecretResolver(secrets...),
			iresolver.ResolutionPriorityDefault,
		),
	)
}

// Apply applies a Plan on the target cluster. It first resolves all dependencies, finding
// any cyclic dependencies and rendering the intent to a directory. It then attempts to
// apply each intent on the cluster.
func (p *Plan) Apply(ctx context.Context) (*PlanState, error) {
	fh, err := filehandler.NewTempHandler(ctx, p.Name())
	if err != nil {
		return nil, fmt.Errorf("creating file handler: %w", err)
	}
	svc, err := service.NewPlanService(ctx, fh)
	if err != nil {
		return nil, err
	}
	intent, err := svc.CreatePlan(ctx, p.Build())
	if err != nil {
		return nil, err
	}
	state, err := intent.Apply(ctx)
	return dry.Wrap2(&PlanState{results: state}, err)
}

// Build resolves the plan DAG by using lazy DFS traversal and cycle detection. That is, cycles are only detected
// during resolution, not during plan creation. The resulting Plan has all child plans resolved and is ready for
// application. If a cycle is detected, it will panic with a human-readable error message.
func (p *Plan) Build() port.Planner {
	var (
		visited = make(map[string]struct{})
		stack   = make(map[string]bool)
		frames  []string
	)

	var resolveFn func(p *Plan) *Plan
	resolveFn = func(p *Plan) *Plan {
		if stack[p.Name()] {
			// slice copy to avoid mutation
			cycle := append([]string(nil), frames...)
			cycle = append(cycle, p.Name())
			panic(renderCycle(cycle))
		}

		if _, ok := visited[p.Name()]; ok {
			return p
		}

		stack[p.Name()] = true
		frames = append(frames, p.Name())

		for _, fn := range p.childFuncs {
			child := fn() // Resolve the child plan.
			resolved := resolveFn(child)
			p.childPlans = append(p.childPlans, resolved)
		}

		frames = frames[:len(frames)-1] // pop the current frame
		stack[p.Name()] = false
		visited[p.Name()] = struct{}{}
		p.childFuncs = nil // clear child functions after resolution
		return p
	}
	return resolveFn(p)
}

// ComponentByName returns a sequence of Components by their ID.
// Zero or more Components may be returned, depending on the ID provided.
// IDs take the form of the following (any are valid):
// - sdk.HelmChart#telepresence
// - sdk.HelmChart#telepresence-1bbec390
// - sdk.Namespace
//
// If providing an ID with a hash, the hash will be stripped and the ID will be
// extracted to match the ID of the component in the plan results.
//
// Example:
//
//	state, err := plan.Apply(ctx)
//	if err != nil { /* handle error */ }
//	for component := range state.ComponentByName("sdk.HelmChart") {
//		// Use the component.
//	}
func (s *PlanState) ComponentByName(id string) iter.Seq[Component] {
	id = ExtractResource(dry.ToPtr(id))
	return func(yield func(Component) bool) {
		for construct := range s.results.Get(id) {
			if !yield(dry.As[Component](construct.Component())) {
				return
			}
		}
	}
}

// Components returns all Components that were applied as part of the plan.
// The order of the components is guaranteed to be the order in which they were processed by the plan engine.
func (s *PlanState) Components() iter.Seq[Component] {
	return func(yield func(Component) bool) {
		for construct := range s.results.Components() {
			if !yield(dry.As[Component](construct)) {
				return
			}
		}
	}
}

// ComponentIDs returns an iterator over all component IDs in the plan state.
// The order of the IDs is guaranteed to be the order in which they were processed by the plan engine.
func (s *PlanState) ComponentIDs() iter.Seq[string] {
	return func(yield func(string) bool) {
		for construct := range s.results.Components() {
			id := ExtractResource(construct.Node().Id())
			if !yield(id) {
				return
			}
		}
	}
}

// ComponentState returns the state of a component as a specific type T.
//
// Example:
//
//	for component := range state.ComponentByName("sdk.ClientSideApply") {
//		csa := ComponentState[*clientsideapply.Result](component)
//		fmt.Println(csa.Args) // Access the Args field of the ClientSideApply component.
//	}
func ComponentState[T any](c Component) T {
	return dry.MustAs[T](c)
}

func renderCycle(frames []string) string {
	var sb strings.Builder
	sb.WriteString("Plan dependency cycle detected:\n")
	for i, frame := range frames {
		_, _ = fmt.Fprintf(&sb, " %s", frame)
		if i < len(frames)-1 {
			sb.WriteString(" ->")
		} else {
			sb.WriteString(" ⟳")
		}
	}
	return sb.String()
}
