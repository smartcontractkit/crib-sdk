package service_test

import (
	"context"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/adapter/filehandler"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
	"github.com/smartcontractkit/crib-sdk/internal/core/service"

	anvilv1 "github.com/smartcontractkit/crib-sdk/crib/composite/blockchain/anvil/v1"
	nginxcontroller "github.com/smartcontractkit/crib-sdk/crib/composite/cluster-services/nginx-controller/v1"
)

func TestCreatePlan(t *testing.T) {
	t.Parallel()
	is := assert.New(t)
	must := require.New(t)
	ctx := t.Context()
	fh, err := filehandler.New(t.Context(), t.TempDir())
	must.NoError(err)

	var (
		p1, p2        func() *crib.Plan
		testComponent = func() crib.ComponentFunc {
			return func(ctx context.Context) (crib.Component, error) {
				return nil, nil
			}
		}
	)

	{
		p1 = func() *crib.Plan {
			return crib.NewPlan("p1",
				crib.ComponentSet(
					testComponent(),
					testComponent(),
				),
				crib.AddPlan(p2),
			)
		}

		p2 = func() *crib.Plan {
			return crib.NewPlan("p2",
				crib.ComponentSet(testComponent()),
			)
		}
	}

	ps, err := service.NewPlanService(ctx, fh)
	must.NoError(err)

	var appPlan *service.AppPlan
	must.NotPanics(func() {
		appPlan, err = ps.CreatePlan(ctx, p1().Build())
	})
	must.NoError(err)
	must.NotNil(appPlan)

	raw := *appPlan.App.SynthYaml()
	is.Empty(raw)
	for manifest, err := range domain.UnmarshalDocument([]byte(raw)) {
		if !is.NoError(err) {
			continue
		}
		snaps.MatchYAML(t, manifest)
	}

	is.Len(appPlan.RootPlan.Components(), 2)
	must.Len(appPlan.RootPlan.ChildPlans(), 1)
	is.Len(appPlan.RootPlan.ChildPlans()[0].Components(), 1)
}

func TestE2ECreatePlan(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)

	ctx := t.Context()
	is := assert.New(t)
	must := require.New(t)
	fh, err := filehandler.New(ctx, t.TempDir())
	must.NoError(err)

	rawPlan := crib.NewPlan("e2e-create-plan",
		crib.Namespace("e2e-create-plan"),
		crib.ComponentSet(
			anvilv1.Component(
				&anvilv1.Props{
					Namespace: "e2e-create-plan",
					ChainID:   "e2e-create-plan",
				},
				anvilv1.UseIngress,
			),
		),
		crib.AddPlan(
			crib.NewPlan("anvil-plan",
				crib.ComponentSet(nginxcontroller.Component()),
			).Plan(),
		),
	)

	ps, err := service.NewPlanService(ctx, fh)
	must.NoError(err)
	must.NotNil(ps)

	plan, err := ps.CreatePlan(ctx, rawPlan)
	must.NoError(err)
	must.NotNil(plan)

	raw := *plan.App.SynthYaml()
	is.NotEmpty(raw)
	var c int
	for manifest, err := range domain.UnmarshalDocument([]byte(raw)) {
		c++
		if !is.NoError(err) {
			continue
		}
		snaps.MatchStandaloneYAML(t, manifest)
	}
	is.Greater(c, 0, "expected at least one manifest in the plan")
}

func TestImagePullSecret(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)
	ctx := t.Context()
	must := require.New(t)
	fh, err := filehandler.New(ctx, t.TempDir())
	must.NoError(err)

	rawPlan := crib.NewPlan("e2e-create-plan",
		crib.Namespace("e2e-create-plan"),
		crib.ComponentSet(
			anvilv1.Component(
				&anvilv1.Props{
					Namespace: "e2e-create-plan",
					ChainID:   "e2e-create-plan",
				},
				anvilv1.UseIngress,
			),
		),
		crib.ImagePullSecrets("image-pull-secret"),
	)

	ps, err := service.NewPlanService(ctx, fh)
	must.NoError(err)
	must.NotNil(ps)

	plan, err := ps.CreatePlan(ctx, rawPlan)
	must.NoError(err)
	must.NotNil(plan)

	internal.SynthAndSnapYamls(t, &internal.TestApp{App: plan.App})
}
