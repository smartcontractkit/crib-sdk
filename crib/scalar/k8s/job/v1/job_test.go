package jobv1

import (
	"context"
	"testing"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"

	kplus "github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2"
)

func TestNewJob(t *testing.T) {
	t.Parallel()
	is := assert.New(t)

	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

	k8sResourceName := dry.ToPtr("test-job")

	// Create job props with container specification
	testProps := &Props{
		JobProps: &kplus.JobProps{
			Metadata: &cdk8s.ApiObjectMetadata{
				Name:      k8sResourceName,
				Namespace: dry.ToPtr("test-namespace"),
				Labels: &map[string]*string{
					"app": k8sResourceName,
				},
			},
		},
	}

	component, err := New(ctx, testProps)
	is.NoError(err)
	is.NotNil(component)
	is.Implements((*kplus.Job)(nil), component)

	job := dry.As[kplus.Job](component)

	// Add a container to the job
	job.AddContainer(&kplus.ContainerProps{
		Name:    dry.ToPtr("test-container"),
		Image:   dry.ToPtr("busybox:latest"),
		Command: dry.PtrStrings("echo", "Hello from Job!"),
	})
	internal.SynthAndSnapYamls(t, app)
}

func TestJobComponent(t *testing.T) {
	ctx := context.Background()

	// Create validator and add to context
	v, err := internal.NewValidator()
	require.NoError(t, err)
	ctx = internal.ContextWithValidator(ctx, v)

	// Create a test app and chart for the component
	app := cdk8s.NewApp(nil)
	chart := cdk8s.NewChart(app, dry.ToPtr("test-chart"), nil)
	ctx = internal.ContextWithConstruct(ctx, chart)

	// Create job props with simple configuration
	jobProps := &kplus.JobProps{
		Metadata: &cdk8s.ApiObjectMetadata{
			Name: dry.ToPtr("test-job"),
		},
	}

	props := &Props{
		JobProps: jobProps,
	}

	// Test validation
	err = props.Validate(ctx)
	require.NoError(t, err)

	// Test component creation
	component, err := New(ctx, props)
	require.NoError(t, err)
	require.NotNil(t, component)

	// Verify the component is a Job
	job, ok := component.(kplus.Job)
	require.True(t, ok)
	require.NotNil(t, job)
}

func TestJobComponentValidation(t *testing.T) {
	ctx := context.Background()

	// Create validator and add to context
	v, err := internal.NewValidator()
	require.NoError(t, err)
	ctx = internal.ContextWithValidator(ctx, v)

	// Test with nil JobProps (should fail validation)
	props := &Props{
		JobProps: nil,
	}

	err = props.Validate(ctx)
	require.Error(t, err)
}
