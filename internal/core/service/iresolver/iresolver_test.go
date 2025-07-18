package iresolver

import (
	"testing"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolvers(t *testing.T) {
	// Do not run this test in parallel as it mutates package-level state.
	is := assert.New(t)

	sampleResolver := func(ctx cdk8s.ResolutionContext) {
		// Sample resolver function that does nothing.
	}

	resolvers := []cdk8s.IResolver{
		NewResolver(sampleResolver, ResolutionPriorityDefault),
		NewResolver(sampleResolver, ResolutionPriorityLow),
		NewResolver(sampleResolver, ResolutionPriorityHigh),
		NewResolver(sampleResolver, ResolutionPriorityHigh),
	}

	got := Resolvers(resolvers)
	is.Len(got, 4, "Expected four resolvers to be returned")

	// Check that the resolvers are sorted by priority.
	is.NotEqual(got, resolvers)
	want := []ResolutionPriority{
		ResolutionPriorityHigh,
		ResolutionPriorityHigh,
		ResolutionPriorityDefault,
		ResolutionPriorityLow,
	}
	priorities := lo.Map(got, func(r cdk8s.IResolver, _ int) ResolutionPriority {
		resolver, ok := r.(*Resolver)
		require.True(t, ok, "Expected resolver to be of type *Resolver")
		return resolver.priority
	})
	is.Equal(want, priorities, "Expected resolvers to be sorted by priority")
}
