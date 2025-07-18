package iresolver

import (
	"slices"
	"testing"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2/k8s"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
)

type testResolver struct {
	resolver func(ctx cdk8s.ResolutionContext)
	kind     string
	keys     []string
	rctx     cdk8s.ResolutionContext
}

func (r *testResolver) Resolve(ctx cdk8s.ResolutionContext) {
	keys := lo.Map(*ctx.Key(), func(k *string, _ int) string {
		return dry.FromPtr(k)
	})
	if r.kind == *ctx.Obj().Kind() && slices.Equal(r.keys, keys) {
		r.rctx = ctx // Capture the context for assertions.
	}
	r.resolver(ctx)
}

func TestImagePullSecrets(t *testing.T) {
	tests := []struct {
		desc      string
		kind      string
		keys      []string
		resource  func(chart cdk8s.Chart)
		assertion func(t *testing.T, got any)
	}{
		{
			desc: "Match - Pod",
			kind: "Pod",
			keys: []string{"spec"},
			resource: func(chart cdk8s.Chart) {
				k8s.NewKubePod(chart, dry.ToPtr("test-pod"), &k8s.KubePodProps{
					Spec: &k8s.PodSpec{
						Containers: dry.PtrSlice([]k8s.Container{
							{Name: dry.ToPtr("test-container")},
						}),
					},
				})
			},
			assertion: func(t *testing.T, got any) {
				spec, ok := dry.As[map[string]any](got)["spec"]
				require.True(t, ok, "Expected spec to be present in the resolved value")
				specMap := dry.As[map[string]any](spec)
				secrets, ok := specMap["imagePullSecrets"]
				require.True(t, ok, "Expected imagePullSecrets to be present in the spec")
				assert.NotNil(t, secrets, "Expected imagePullSecrets to be non-nil")
			},
		},
		{
			desc: "Match - Deployment",
			kind: "Deployment",
			keys: []string{"spec", "template", "spec"},
			resource: func(chart cdk8s.Chart) {
				k8s.NewKubeDeployment(chart, dry.ToPtr("test-deployment"), &k8s.KubeDeploymentProps{
					Spec: &k8s.DeploymentSpec{
						Selector: &k8s.LabelSelector{
							MatchLabels: dry.PtrMapping(map[string]string{"app": "test"}),
						},
						Template: &k8s.PodTemplateSpec{
							Spec: &k8s.PodSpec{
								Containers: dry.PtrSlice([]k8s.Container{
									{Name: dry.ToPtr("test-container")},
								}),
							},
						},
					},
				})
			},
			assertion: func(t *testing.T, got any) {
				spec, ok := dry.As[map[string]any](got)["spec"]
				require.True(t, ok, "Expected spec to be present in the resolved value")

				specMap := dry.As[map[string]any](spec)
				secrets, ok := specMap["imagePullSecrets"]
				require.False(t, ok, "Expected imagePullSecrets to be present in the spec")

				templateMap, ok := specMap["template"]
				require.True(t, ok, "Expected template to be present in the spec")

				templateSpec, ok := dry.As[map[string]any](templateMap)["spec"]
				require.True(t, ok, "Expected template spec to be present in the template")

				secrets, ok = dry.As[map[string]any](templateSpec)["imagePullSecrets"]
				require.True(t, ok, "Expected imagePullSecrets to be present in the template spec")
				assert.NotNil(t, secrets, "Expected imagePullSecrets to be non-nil")
			},
		},
		{
			desc: "No match - Kind",
			kind: "Service",
			keys: []string{"spec"},
			resource: func(chart cdk8s.Chart) {
				k8s.NewKubeService(chart, dry.ToPtr("test-service"), &k8s.KubeServiceProps{
					Spec: &k8s.ServiceSpec{
						ExternalName: dry.ToPtr("test-service"),
					},
				})
			},
			assertion: func(t *testing.T, got any) {
				spec, ok := dry.As[map[string]any](got)["spec"]
				require.True(t, ok, "Expected spec to be present in the resolved value")
				specMap := dry.As[map[string]any](spec)
				_, ok = specMap["imagePullSecrets"]
				assert.False(t, ok, "Expected imagePullSecrets to be absent in the spec")
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			r := &testResolver{
				resolver: ImagePullSecretResolver("test-secret"),
				kind:     tc.kind,
				keys:     tc.keys,
			}
			app := cdk8s.Testing_App(&cdk8s.AppProps{
				YamlOutputType: cdk8s.YamlOutputType_FILE_PER_APP,
				Resolvers:      dry.ToPtr([]cdk8s.IResolver{r}),
			})
			chart := cdk8s.NewChart(app, dry.ToPtr("TestChart"), nil)
			tc.resource(chart)
			// Trigger the resolver.
			SynthAndSnapYAMLs(t, app)
			tc.assertion(t, r.rctx.Obj().ToJson())
		})
	}
}

// SynthAndSnapYAMLs calls SynthYaml and unmarshal results into separate files.
// Note: This is copied from internal/testing_utils.go - which cannot be imported
// due to a circular dependency.
func SynthAndSnapYAMLs(t *testing.T, app cdk8s.App) {
	raw := *app.SynthYaml()
	for obj, err := range domain.UnmarshalDocument([]byte(raw)) {
		if err != nil {
			require.NoError(t, err)
		}
		snaps.MatchStandaloneYAML(t, obj)
	}
}
