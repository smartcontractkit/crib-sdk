package deploymentv1

import (
	"testing"

	"github.com/aws/jsii-runtime-go"
	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2/k8s"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"

	workloadv1 "github.com/smartcontractkit/crib-sdk/crib/composite/workload/v1"
)

func TestDeploymentComponent(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)

	is := assert.New(t)

	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

	const (
		k8sResourceName = "blockchain-1234"
		appInstanceName = "blockchain-1234"
		appName         = "blockchain"
	)

	container := &domain.Container{
		Name:    "blockchain",
		Image:   "example-image:latest",
		Command: []string{"blockchain"},
		Args: []string{
			"start",
			"--mode",
			"simulated",
		},
		Ports: []domain.ContainerPort{
			{
				Name:          "http",
				ContainerPort: 8080,
				Protocol:      "TCP",
			},
		},
		Env: []domain.EnvVar{
			{
				Name:  "FOO",
				Value: "BAR",
			},
		},
		Resources: &domain.Resources{
			Limits: map[string]string{
				"cpu":    "500m",
				"memory": "512Mi",
			},
			Requests: map[string]string{
				"cpu":    "100m",
				"memory": "128Mi",
			},
		},
		UserID:  1000,
		GroupID: 1000,
	}

	testProps := &Props{
		Name:        k8sResourceName,
		AppInstance: appInstanceName,
		AppName:     appName,
		Namespace:   "test-ns",
		Containers:  []*domain.Container{container},
	}

	c, err := Component(testProps)(ctx)
	is.NoError(err)
	is.NotNil(c)
	deployment := dry.MustAs[*DeploymentComposite](c)

	t.Run("List of charts", func(t *testing.T) {
		// Normalize the chart names into a more readable format.
		gotCharts := lo.Map(*app.Charts(), func(c cdk8s.Chart, _ int) string {
			return crib.ExtractResource(c.Node().Id())
		})
		wantCharts := []string{
			"TestingApp",
			"sdk.DeploymentV1",
		}
		is.Equal(wantCharts, gotCharts)
		is.Len(*app.Charts(), len(wantCharts))
	})

	t.Run("Exposing Via Ingress", func(t *testing.T) {
		service, err := deployment.ExposeViaIngress("/", &workloadv1.ExposeViaIngressProps{
			Name:             "",
			IngressClassName: "example-ingress",
			Ports: []*k8s.ServicePort{
				{
					Port:       jsii.Number(8545),
					Protocol:   dry.ToPtr("TCP"),
					TargetPort: k8s.IntOrString_FromNumber(jsii.Number(8545)),
				},
			},
		})

		is.NoError(err)
		is.NotNil(service)
	})

	t.Run("WaitFor", func(t *testing.T) {
		err = deployment.WaitForRollout(ctx)
		is.NoError(err)
	})

	t.Run("Synth", func(t *testing.T) {
		internal.SynthAndSnapYamls(t, app)
	})
}
