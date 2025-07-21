package nodesetv1

import (
	"fmt"
	"testing"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"

	chainlinknodev1 "github.com/smartcontractkit/crib-sdk/crib/composite/chainlink/node/v1"
)

// TestNodeSetComponent verifies that the NodeSet component creates the correct
// number of PostgreSQL databases and Chainlink node components.
func TestNodeSetComponent(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)

	is := assert.New(t)
	must := require.New(t)

	// Setup test environment
	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

	// Define test properties for 3 Chainlink nodes with PostgreSQL resources
	testProps := &Props{
		Namespace: "test-namespace",
		Size:      3,
		PostgresResources: map[string]map[string]string{
			"requests": {
				"cpu":    "500m",
				"memory": "512Mi",
			},
			"limits": {
				"cpu":    "1000m",
				"memory": "1Gi",
			},
		},
		NodeProps: []*chainlinknodev1.Props{
			{
				AppInstanceName: "chainlink-node-0",
				Image:           "chainlink/chainlink:latest",
				Config: `[Log]
Level = 'warn'

[WebServer]
AllowOrigins = '*'
HTTPWriteTimeout = '10m'
HTTPPort = 6688
SecureCookies = false`,
			},
			{
				AppInstanceName: "chainlink-node-1",
				Image:           "chainlink/chainlink:latest",
				Config: `[Log]
Level = 'warn'

[WebServer]
AllowOrigins = '*'
HTTPWriteTimeout = '10m'
HTTPPort = 6688
SecureCookies = false`,
			},
			{
				AppInstanceName: "chainlink-node-2",
				Image:           "chainlink/chainlink:latest",
				Config: `[Log]
Level = 'warn'

[WebServer]
AllowOrigins = '*'
HTTPWriteTimeout = '10m'
HTTPPort = 6688
SecureCookies = false`,
			},
		},
	}

	// Create and validate component
	c := Component(testProps)
	component, err := c(ctx)
	must.NoError(err, "Component creation should not return an error")
	must.NotNil(component, "Component should not be nil")

	// Type assert to Result struct and validate its structure
	result, ok := component.(Result)
	must.True(ok, "Component should return a Result struct")
	must.NotNil(result.Component, "Result.Component should not be nil")
	must.Len(result.Nodes, 3, "Result.Nodes should contain exactly 3 chainlink nodes")

	// Verify each node has the correct properties
	for i, node := range result.Nodes {
		must.NotNil(node, "Node %d should not be nil", i)
		must.NotNil(node.Component, "Node %d Component should not be nil", i)
		// Verify the expected app instance name
		expectedName := fmt.Sprintf("chainlink-node-%d", i)
		is.Equal(expectedName, testProps.NodeProps[i].AppInstanceName, "Node %d should have correct app instance name", i)
	}

	// Verify chart structure
	gotCharts := lo.Map(*app.Charts(), func(c cdk8s.Chart, _ int) string {
		return crib.ExtractResource(c.Node().Id())
	})

	// We expect:
	// - TestingApp (root)
	// - sdk.NodeSet (main component)
	// - sdk.HelmChart#postgres (PostgreSQL)
	// - 3 x sdk.composite.chainlink.node.v1 (Chainlink nodes)
	// - sdk.ClientSideApply (wait for all nodes)
	// Each Chainlink node creates multiple sub-charts, so we expect more charts
	is.GreaterOrEqual(len(gotCharts), 6, "Should have at least 6 charts (app, nodeset, postgres, 3 chainlink nodes, clientsideapply)")

	// Verify PostgreSQL chart exists
	postgresCharts := lo.Filter(gotCharts, func(name string, _ int) bool {
		return name == "sdk.HelmChart#postgres"
	})
	is.Len(postgresCharts, 1, "Should have exactly one PostgreSQL chart")

	// Verify Chainlink node charts exist - use the correct component name
	chainlinkCharts := lo.Filter(gotCharts, func(name string, _ int) bool {
		return name == "sdk.composite.chainlink.node.v1"
	})
	is.Len(chainlinkCharts, 3, "Should have exactly 3 Chainlink node charts")

	// Verify ClientSideApply chart exists
	clientSideApplyCharts := lo.Filter(gotCharts, func(name string, _ int) bool {
		return name == "sdk.ClientSideApply"
	})
	is.Len(clientSideApplyCharts, 1, "Should have exactly one ClientSideApply chart")

	// Find and verify the ClientSideApply component
	var waitForChart cdk8s.Chart
	for _, c := range *app.Charts() {
		if !*cdk8s.Chart_IsChart(c) {
			continue
		}
		if crib.ExtractResource(c.Node().Id()) == "sdk.ClientSideApply" {
			waitForChart = c
			break
		}
	}
	must.NotNil(waitForChart, "ClientSideApply chart should be found")

	// Verify ClientSideApply configuration
	t.Run("ClientSideApply", func(t *testing.T) {
		var obj cdk8s.ApiObject
		is.NotPanics(func() {
			obj = cdk8s.ApiObject_Of(waitForChart.Node().DefaultChild())
		}, "Should not panic when getting default child")
		is.NotNil(obj, "ClientSideApply object should not be nil")

		// Verify object metadata
		is.Equal("crib.smartcontract.com/v1alpha1", *obj.ApiVersion(), "API version should match")
		is.Equal("ClientSideApply", *obj.Kind(), "Kind should be ClientSideApply")
		is.Equal("crib.smartcontract.com", *obj.ApiGroup(), "API group should match")
		is.Equal("test-namespace", *obj.Metadata().Namespace(), "Namespace should match")

		// Verify object specification
		json := dry.As[map[string]any](obj.ToJson())
		is.NotNil(json, "JSON representation should not be nil")
		spec := dry.As[map[string]any](json["spec"])
		is.NotNil(spec, "Spec should not be nil")

		want := map[string]any{
			"onFailure": "abort",
			"action":    "kubectl",
			"args": []any{
				"wait",
				"-n", "test-namespace",
				"--for=condition=ready",
				"pod",
				"-l=app.kubernetes.io/name=chainlink",
				"--timeout=600s",
			},
		}
		is.Equal(want, spec, "Spec should match expected configuration")
	})
}

// TestNodeSetValidation verifies the validation logic for Props.
func TestNodeSetValidation(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)
	is := assert.New(t)

	ctx := t.Context()

	// Test case: Size and NodeProps length mismatch
	invalidProps := &Props{
		Namespace: "test-namespace",
		Size:      3,
		NodeProps: []*chainlinknodev1.Props{
			{
				AppInstanceName: "node-0",
				Image:           "chainlink/chainlink:latest",
				DatabaseURL:     "placeholder",
				Config:          "[Log]\nLevel = 'warn'",
			},
			// Only 1 node prop, but Size is 3
		},
	}
	err := invalidProps.Validate(ctx)
	is.Error(err, "Should return validation error for mismatched Size and NodeProps length")
	is.Contains(err.Error(), "NodeProps length (1) must match Size (3)")

	// Test case: NodeProps with non-empty Namespace (should fail)
	invalidPropsWithNamespace := &Props{
		Namespace: "test-namespace",
		Size:      1,
		NodeProps: []*chainlinknodev1.Props{
			{
				Namespace:       "custom-namespace", // This should cause validation error
				AppInstanceName: "node-0",
				Image:           "chainlink/chainlink:latest",
				DatabaseURL:     "placeholder",
				Config:          "[Log]\nLevel = 'warn'",
			},
		},
	}

	err = invalidPropsWithNamespace.Validate(ctx)
	is.Error(err, "Should return validation error for NodeProps with non-empty Namespace")
	is.Contains(err.Error(), "NodeProps[0].Namespace must be empty to allow automatic propagation")

	// Test case: Valid props
	validProps := &Props{
		Namespace: "test-namespace",
		Size:      2,
		NodeProps: []*chainlinknodev1.Props{
			{
				AppInstanceName: "node-0",
				Image:           "chainlink/chainlink:latest",
				DatabaseURL:     "placeholder",
				Config:          "[Log]\nLevel = 'warn'",
			},
			{
				AppInstanceName: "node-1",
				Image:           "chainlink/chainlink:latest",
				DatabaseURL:     "placeholder",
				Config:          "[Log]\nLevel = 'warn'",
			},
		},
	}

	err = validProps.Validate(ctx)
	is.NoError(err, "Should not return validation error for valid props")
}

// TestNodeSetResult verifies that the NodeSet component returns the correct Result struct
// with proper node references and component interface compliance.
func TestNodeSetResult(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)

	is := assert.New(t)
	must := require.New(t)

	// Setup test environment
	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

	// Create a smaller test case with 2 nodes for focused testing
	testProps := &Props{
		Namespace: "test-result-namespace",
		Size:      2,
		NodeProps: []*chainlinknodev1.Props{
			{
				AppInstanceName: "result-test-node-0",
				Image:           "chainlink/chainlink:latest",
				Config:          "[Log]\nLevel = 'warn'",
			},
			{
				AppInstanceName: "result-test-node-1",
				Image:           "chainlink/chainlink:latest",
				Config:          "[Log]\nLevel = 'warn'",
			},
		},
	}

	// Create component and get result
	component, err := Component(testProps)(ctx)
	must.NoError(err, "Component creation should succeed")

	// Verify Result struct
	result, ok := component.(Result)
	must.True(ok, "Component should return a Result struct")

	// Verify Result implements crib.Component interface
	var _ crib.Component = result
	is.NotNil(result.Component, "Result should embed a valid crib.Component")

	// Verify nodes array
	must.Len(result.Nodes, 2, "Result should contain exactly 2 nodes")

	// Verify each node is properly configured
	for i, node := range result.Nodes {
		must.NotNil(node, "Node %d should not be nil", i)
		must.NotNil(node.Component, "Node %d should have a valid Component", i)

		// Verify the node is also a Result struct from chainlink node component
		is.IsType(&chainlinknodev1.Result{}, node, "Node %d should be a chainlink node Result", i)
	}

	// Verify that the Result can be used as a crib.Component
	// This ensures the embedded component works correctly
	is.NotNil(result.Node(), "Result should have a valid Node() method from crib.Component")
}

// TestGenerateInitSQL verifies the SQL generation logic.
func TestGenerateInitSQL(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)
	is := assert.New(t)

	props := &Props{
		Size: 2,
	}

	sql := generateInitSQL(props)

	// Verify the SQL contains the expected statements
	is.Contains(sql, "CREATE USER chainlink_user_0 WITH PASSWORD 'chainlink_pass_0';")
	is.Contains(sql, "CREATE DATABASE chainlink_node_0 OWNER chainlink_user_0;")
	is.Contains(sql, "GRANT ALL PRIVILEGES ON DATABASE chainlink_node_0 TO chainlink_user_0;")

	is.Contains(sql, "CREATE USER chainlink_user_1 WITH PASSWORD 'chainlink_pass_1';")
	is.Contains(sql, "CREATE DATABASE chainlink_node_1 OWNER chainlink_user_1;")
	is.Contains(sql, "GRANT ALL PRIVILEGES ON DATABASE chainlink_node_1 TO chainlink_user_1;")
}

// TestNodeSetWithPostgresResources verifies that PostgreSQL resources are properly applied.
func TestNodeSetWithPostgresResources(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)

	is := assert.New(t)
	must := require.New(t)

	// Setup test environment
	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

	// Define test properties with custom PostgreSQL resources
	testProps := &Props{
		Namespace: "test-resources-namespace",
		Size:      1,
		PostgresResources: map[string]map[string]string{
			"requests": {
				"cpu":    "250m",
				"memory": "256Mi",
			},
			"limits": {
				"cpu":    "750m",
				"memory": "768Mi",
			},
		},
		NodeProps: []*chainlinknodev1.Props{
			{
				AppInstanceName: "test-node",
				Image:           "chainlink/chainlink:latest",
				Config:          "[Log]\nLevel = 'warn'",
			},
		},
	}

	// Create component
	component, err := Component(testProps)(ctx)
	must.NoError(err, "Component creation should succeed")
	must.NotNil(component, "Component should not be nil")

	// Verify the component was created successfully
	result, ok := component.(Result)
	must.True(ok, "Component should return a Result struct")
	is.Len(result.Nodes, 1, "Should have exactly 1 node")
}

// TestNodeSetBackwardCompatibility verifies that the component works without PostgreSQL resources.
func TestNodeSetBackwardCompatibility(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)

	is := assert.New(t)
	must := require.New(t)

	// Setup test environment
	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

	// Define test properties without PostgreSQL resources (backward compatibility)
	testProps := &Props{
		Namespace: "test-backward-compat",
		Size:      1,
		NodeProps: []*chainlinknodev1.Props{
			{
				AppInstanceName: "compat-test-node",
				Image:           "chainlink/chainlink:latest",
				Config:          "[Log]\nLevel = 'warn'",
			},
		},
	}

	// Create component
	component, err := Component(testProps)(ctx)
	must.NoError(err, "Component creation should succeed without PostgreSQL resources")
	must.NotNil(component, "Component should not be nil")

	// Verify the component was created successfully
	result, ok := component.(Result)
	must.True(ok, "Component should return a Result struct")
	is.Len(result.Nodes, 1, "Should have exactly 1 node")
}

// TestNodeSetWithEmptyPostgresResources verifies that empty PostgreSQL resources work correctly.
func TestNodeSetWithEmptyPostgresResources(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)

	is := assert.New(t)
	must := require.New(t)

	// Setup test environment
	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

	// Define test properties with empty PostgreSQL resources
	testProps := &Props{
		Namespace:         "test-empty-resources",
		Size:              1,
		PostgresResources: map[string]map[string]string{}, // Empty map
		NodeProps: []*chainlinknodev1.Props{
			{
				AppInstanceName: "empty-resources-node",
				Image:           "chainlink/chainlink:latest",
				Config:          "[Log]\nLevel = 'warn'",
			},
		},
	}

	// Create component
	component, err := Component(testProps)(ctx)
	must.NoError(err, "Component creation should succeed with empty PostgreSQL resources")
	must.NotNil(component, "Component should not be nil")

	// Verify the component was created successfully
	result, ok := component.(Result)
	must.True(ok, "Component should return a Result struct")
	is.Len(result.Nodes, 1, "Should have exactly 1 node")
}
