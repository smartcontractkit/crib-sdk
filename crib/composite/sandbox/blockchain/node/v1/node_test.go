package blockchainnodev1

import (
	"testing"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

func TestComponent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode.")
	}

	testCases := []struct {
		name           string
		blockchainType BlockchainType
		wantCharts     []string
	}{
		{
			name:           "Anvil blockchain",
			blockchainType: BlockchainTypeAnvil,
			wantCharts: []string{
				"TestingApp",
				"sdk.Blockchain",
				"sdk.HelmChart#anvil",
				"sdk.Namespace",
				"sdk.HelmChart",
				"sdk.ClientSideApply",
			},
		},
		{
			name:           "Aptos blockchain",
			blockchainType: BlockchainTypeAptos,
			wantCharts: []string{
				"TestingApp",
				"sdk.Blockchain",
				"sdk.HelmChart#aptos",
				"sdk.Namespace",
				"sdk.HelmChart",
				"sdk.ClientSideApply",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			internal.JSIIKernelMutex.Lock()
			defer internal.JSIIKernelMutex.Unlock()

			is := assert.New(t)

			app := internal.NewTestApp(t)
			ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

			testProps := &Props{
				Namespace:   "test-namespace",
				ReleaseName: "test-blockchain",
				Type:        tc.blockchainType,
				ChainID:     "1337",
				Values: map[string]any{
					"key": "value",
				},
				ValuesPatches: [][]string{
					{"path", "to", "value"},
				},
			}

			c := Component(testProps)
			component, err := c(ctx)
			is.NoError(err)
			is.NotNil(component)

			// Normalize the chart names into a more readable format.
			gotCharts := lo.Map(*app.Charts(), func(c cdk8s.Chart, _ int) string {
				return crib.ExtractResource(c.Node().Id())
			})
			is.Equal(tc.wantCharts, gotCharts)
			is.Len(*app.Charts(), len(tc.wantCharts))

			// Debug: Print all chart IDs and types
			for i, c := range *app.Charts() {
				id := crib.ExtractResource(c.Node().Id())
				chartType := "<unknown>"
				if dry.FromPtr(cdk8s.Chart_IsChart(c)) {
					chartType = "Chart"
				}
				t.Logf("Chart[%d]: ID=%s, Type=%s", i, id, chartType)
			}

			var blockchain, namespace, helmChart, waitFor cdk8s.Chart
			for _, c := range *app.Charts() {
				if !dry.FromPtr(cdk8s.Chart_IsChart(c)) {
					continue
				}

				switch crib.ExtractResource(c.Node().Id()) {
				case "sdk.HelmChart#anvil", "sdk.HelmChart#aptos":
					blockchain = c
				case "sdk.Namespace":
					namespace = c
				case "sdk.HelmChart":
					helmChart = c
				case "sdk.ClientSideApply":
					waitFor = c
				}
			}

			// Assert that all expected charts are found
			is.NotNil(blockchain, "blockchain chart should not be nil")
			is.NotNil(namespace, "namespace chart should not be nil")
			is.NotNil(helmChart, "helmChart chart should not be nil")
			is.NotNil(waitFor, "waitFor chart should not be nil")

			t.Run("Namespace", func(t *testing.T) {
				children := namespace.Node().Children()
				t.Logf("namespace chart children: %d", len(*children))
				child := namespace.Node().DefaultChild()
				if child == nil {
					t.Log("namespace chart's DefaultChild is nil; skipping ApiObject_Of check")
					return
				}
				var obj cdk8s.ApiObject
				is.NotPanics(func() {
					obj = cdk8s.ApiObject_Of(child)
				})
				is.NotNil(obj)
				is.Equal("v1", *obj.ApiVersion())
				is.Equal("Namespace", *obj.Kind())
				is.Equal("test-namespace", *obj.Metadata().Name())
			})

			t.Run("Blockchain", func(t *testing.T) {
				children := blockchain.Node().Children()
				t.Logf("blockchain chart children: %d", len(*children))
				child := blockchain.Node().DefaultChild()
				if child == nil {
					t.Log("blockchain chart's DefaultChild is nil; skipping ApiObject_Of check")
					return
				}
				var obj cdk8s.ApiObject
				is.NotPanics(func() {
					obj = cdk8s.ApiObject_Of(child)
				})
				is.NotNil(obj)

				is.Equal("helm.toolkit.fluxcd.io/v2beta1", *obj.ApiVersion())
				is.Equal("HelmRelease", *obj.Kind())
				is.Equal("test-blockchain", *obj.Metadata().Name())
				is.Equal("test-namespace", *obj.Metadata().Namespace())
			})

			t.Run("HelmChart", func(t *testing.T) {
				children := helmChart.Node().Children()
				t.Logf("helmChart chart children: %d", len(*children))
				child := helmChart.Node().DefaultChild()
				if child == nil {
					t.Log("helmChart chart's DefaultChild is nil; skipping ApiObject_Of check")
					return
				}
				var obj cdk8s.ApiObject
				is.NotPanics(func() {
					obj = cdk8s.ApiObject_Of(child)
				})
				is.NotNil(obj)

				is.Equal("source.toolkit.fluxcd.io/v1beta2", *obj.ApiVersion())
				is.Equal("HelmRepository", *obj.Kind())
				is.Equal("test-blockchain", *obj.Metadata().Name())
				is.Equal("test-namespace", *obj.Metadata().Namespace())
			})

			t.Run("WaitFor", func(t *testing.T) {
				var obj cdk8s.ApiObject
				is.NotPanics(func() {
					obj = cdk8s.ApiObject_Of(waitFor.Node().DefaultChild())
				})
				is.NotNil(obj)

				is.Equal("crib.smartcontract.com/v1alpha1", *obj.ApiVersion())
				is.Equal("ClientSideApply", *obj.Kind())
				is.Equal("crib.smartcontract.com", *obj.ApiGroup())
				is.Equal("test-namespace", *obj.Metadata().Namespace())

				json := dry.As[map[string]any](obj.ToJson())
				is.NotNil(json)
				spec := dry.As[map[string]any](json["spec"])
				is.NotNil(spec)

				want := map[string]any{
					"onFailure": "abort",
					"action":    "kubectl",
					"args": []any{
						"wait",
						"-n", "test-namespace",
						"--for=condition=ready",
						"pod",
						"-l=app.kubernetes.io/component=test-blockchain,app.kubernetes.io/name=devspace-app",
						"--timeout=600s",
					},
				}
				is.Equal(want, spec)
			})
		})
	}
}
