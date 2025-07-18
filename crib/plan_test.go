package crib

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChildCycle(t *testing.T) {
	t.Parallel()

	var (
		p1, p2, p3, p4 func() *Plan

		testComponent = func() ComponentFunc {
			return func(ctx context.Context) (Component, error) {
				return nil, nil
			}
		}
	)

	{
		p1 = func() *Plan {
			return NewPlan("p1",
				ComponentSet(testComponent()),
				AddPlan(p2),
			)
		}
		p2 = func() *Plan {
			return NewPlan("p2",
				ComponentSet(testComponent()),
				AddPlan(p3),
				AddPlan(p4),
			)
		}
		p3 = func() *Plan {
			return NewPlan("p3",
				ComponentSet(testComponent()),
				AddPlan(p1), // This will cause a cycle.
			)
		}
		p4 = func() *Plan {
			return NewPlan("p4",
				ComponentSet(testComponent()),
			)
		}
	}

	assert.Panics(t, func() {
		p1().Build()
	})
}

func TestBuild(t *testing.T) {
	t.Parallel()

	var (
		p1, p2, p3 func() *Plan

		testComponent = func() ComponentFunc {
			return func(ctx context.Context) (Component, error) {
				return nil, nil
			}
		}
	)

	{
		p1 = func() *Plan {
			return NewPlan("p1",
				ComponentSet(
					testComponent(),
					testComponent(),
				),
				AddPlan(p2),
				AddPlan(p3),
			)
		}
		p2 = func() *Plan {
			return NewPlan("p2",
				ComponentSet(testComponent()),
			)
		}
		p3 = func() *Plan {
			return NewPlan("p3",
				ComponentSet(testComponent()),
			)
		}
	}
	is := assert.New(t)
	must := require.New(t)

	plan := p1()
	is.Len(plan.Components(), 2, "Plan p1 should have 2 components")
	// We haven't built the plan yet, so it should not have any child plans.
	is.Len(plan.ChildPlans(), 0, "Plan p1 should not have any child plans yet")

	is.NotPanics(func() {
		plan.Build()
	})

	is.Len(plan.Components(), 2, "Plan p1 should still have 2 components after Build")
	is.Equal("p1", plan.Name())
	must.Len(plan.ChildPlans(), 2, "Plan p1 should have 2 child plans after Build")

	is.Equal("p2", plan.ChildPlans()[0].Name(), "First child plan should be p2")
	is.Equal("p3", plan.ChildPlans()[1].Name(), "Second child plan should be p3")

	is.Len(plan.ChildPlans()[0].Components(), 1, "Child plan p2 should have 1 component")
	is.Len(plan.ChildPlans()[1].Components(), 1, "Child plan p3 should have 1 component")
}
