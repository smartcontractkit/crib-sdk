package iresolver

import (
	"cmp"
	"slices"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
)

const (
	// ResolutionPriorityDefault is the default priority for a resolver.
	ResolutionPriorityDefault ResolutionPriority = 10
	// ResolutionPriorityLow is the lowest priority for a resolver.
	ResolutionPriorityLow ResolutionPriority = 0
	// ResolutionPriorityHigh is the highest priority for a resolver.
	ResolutionPriorityHigh ResolutionPriority = 100
)

type (
	// ResolutionPriority is an integer type that represents the priority of a resolver.
	// Higher values indicate higher priority.
	// You may optionally use the enums as helpers:
	// - ResolutionPriorityDefault for default priority (10)
	// - ResolutionPriorityLow for low priority (0)
	// - ResolutionPriorityHigh for high priority (100)..
	ResolutionPriority int

	// ResolverFn is a function type that takes a ResolutionContext and performs
	// a cdk8s resolution on the context object.
	ResolverFn func(context cdk8s.ResolutionContext)

	// Resolver represents a series of hooks that can hook into the rendering process
	// of a manifest during the CDK8s rendering phase. Each resolver can define
	// a priority, which determines the order in which the resolvers are executed.
	// This is because CDK8s stops processing once a resolver handles a resolution.
	// Priority is an integer where larger values indicate higher priority.
	Resolver struct {
		fn       ResolverFn
		priority ResolutionPriority
	}
)

// Resolvers returns a copy of the current list of registered resolvers in their correct priority order.
func Resolvers(resolvers []cdk8s.IResolver) []cdk8s.IResolver {
	// Create a copy of the resolvers slice to avoid concurrent modification issues.
	resolversCopy := make([]cdk8s.IResolver, len(resolvers))
	copy(resolversCopy, resolvers)

	slices.SortFunc(resolversCopy, func(a, b cdk8s.IResolver) int {
		priorityA, priorityB := ResolutionPriorityDefault, ResolutionPriorityDefault
		if resolverA, ok := a.(*Resolver); ok {
			priorityA = resolverA.priority
		}
		if resolverB, ok := b.(*Resolver); ok {
			priorityB = resolverB.priority
		}
		return cmp.Compare(priorityB, priorityA)
	})
	return resolversCopy
}

// NewResolver creates a new Resolver with the given ResolutionPriority and returns it.
// If the provided ResolverFn is nil, it returns nil to skip registering a resolver.
func NewResolver(fn ResolverFn, priority ResolutionPriority) *Resolver {
	if fn == nil {
		return nil // Skip nil resolvers.
	}
	return &Resolver{
		fn:       fn,
		priority: priority,
	}
}

// Resolve implements the cdk8s.IResolver interface for the Resolver type.
func (r *Resolver) Resolve(ctx cdk8s.ResolutionContext) {
	r.fn(ctx)
}
