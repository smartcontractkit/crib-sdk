package chainlinknodev1

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
)

// TestComponent verifies that the ChainlinkNode component is properly constructed
// with all required charts and configurations.
func TestComponent(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)

	is := assert.New(t)

	// Setup test environment
	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

	// Define test properties
	testProps := &Props{
		Namespace:       "test-namespace",
		AppInstanceName: "test-chainlink",
		Image:           "chainlink/chainlink:latest",
		ImagePullPolicy: "IfNotPresent",
		Config:          "LogLevel = \"info\"",
		ConfigOverrides: map[string]string{
			"override1.toml": "Override1 = \"value1\"",
			"override2.toml": "Override2 = \"value2\"",
		},
		SecretsOverrides: map[string]string{
			"secrets-override1.toml": "SecretOverride1 = \"secret1\"",
			"secrets-override2.toml": "SecretOverride2 = \"secret2\"",
		},
		DatabaseURL: "postgresql://user:password@db-host:5432/chainlink",
		Replicas:    1,
		EnvVars: map[string]string{
			"TEST_ENV": "test-value",
		},
		Resources: ResourceRequirements{
			Limits:   map[string]string{"cpu": "500m", "memory": "1Gi"},
			Requests: map[string]string{"cpu": "250m", "memory": "512Mi"},
		},
	}

	// Create and validate component
	component, err := Component(testProps)(ctx)
	is.NoError(err, "Component creation should not return an error")
	is.NotNil(component, "Component should not be nil")

	// Cast to Result type and test new fields
	result, ok := component.(Result)
	is.True(ok, "Component should be of type Result")
	is.Equal("http://test-chainlink.test-namespace.svc.cluster.local:6688", result.APIUrl(), "APIUrl should be constructed correctly")

	// Verify chart structure using the embedded Component
	gotCharts := lo.Map(*app.Charts(), func(c cdk8s.Chart, _ int) string {
		return crib.ExtractResource(c.Node().Id())
	})
	wantCharts := []string{
		"TestingApp",
		ComponentName,
		"sdk.ConfigMapV1",
		"sdk.SecretV1",
		"sdk.ServiceV1",
	}
	is.ElementsMatch(wantCharts, gotCharts, "Expected charts should match actual charts")

	// Find specific charts for testing
	var configmap, secret, service, chainlinkChart cdk8s.Chart
	for _, c := range *app.Charts() {
		if !dry.FromPtr(cdk8s.Chart_IsChart(c)) {
			continue
		}

		switch crib.ExtractResource(c.Node().Id()) {
		case "sdk.ConfigMapV1":
			configmap = c
		case "sdk.SecretV1":
			secret = c
		case "sdk.ServiceV1":
			service = c
		case ComponentName:
			chainlinkChart = c
		}
	}

	// Verify required charts exist
	is.NotNil(configmap, "ConfigMap chart should be present")
	is.NotNil(secret, "Secret chart should be present")
	is.NotNil(service, "Service chart should be present")
	is.NotNil(chainlinkChart, "ChainlinkNode chart should be present")

	// Test Secret configuration
	t.Run("Secret", func(t *testing.T) {
		var obj cdk8s.ApiObject
		is.NotPanics(func() {
			obj = cdk8s.ApiObject_Of(secret.Node().DefaultChild())
		}, "Should not panic when getting default child")
		is.NotNil(obj, "Secret object should not be nil")

		// Verify object metadata
		is.Equal("v1", *obj.ApiVersion(), "API version should match")
		is.Equal("Secret", *obj.Kind(), "Kind should be Secret")
		is.Equal("test-namespace", *obj.Metadata().Namespace(), "Namespace should match")

		// Verify object has required string data
		json := dry.As[map[string]any](obj.ToJson())
		is.NotNil(json, "JSON representation should not be nil")
		stringData := dry.As[map[string]any](json["stringData"])
		is.NotNil(stringData, "StringData should not be nil")
		is.Contains(stringData, "apicredentials", "Should contain apicredentials")
		is.Contains(stringData, "secrets.toml", "Should contain secrets.toml")
		is.Contains(stringData, "secrets-override1.toml", "Should contain secrets-override1.toml")
		is.Contains(stringData, "secrets-override2.toml", "Should contain secrets-override2.toml")
		is.Len(stringData, 4, "Should have exactly four secret files")
	})

	// Test ConfigMap configuration
	t.Run("ConfigMap", func(t *testing.T) {
		var obj cdk8s.ApiObject
		is.NotPanics(func() {
			obj = cdk8s.ApiObject_Of(configmap.Node().DefaultChild())
		}, "Should not panic when getting default child")
		is.NotNil(obj, "ConfigMap object should not be nil")

		// Verify object metadata
		is.Equal("v1", *obj.ApiVersion(), "API version should match")
		is.Equal("ConfigMap", *obj.Kind(), "Kind should be ConfigMap")
		is.Equal("test-namespace", *obj.Metadata().Namespace(), "Namespace should match")

		// Verify object has config data
		json := dry.As[map[string]any](obj.ToJson())
		is.NotNil(json, "JSON representation should not be nil")
		data := dry.As[map[string]any](json["data"])
		is.NotNil(data, "Data should not be nil")
		is.Contains(data, "config.toml", "Should contain config.toml")
		is.Contains(data, "override1.toml", "Should contain override1.toml")
		is.Contains(data, "override2.toml", "Should contain override2.toml")
		is.Len(data, 3, "Should have exactly three config files")
	})

	// Test Service configuration
	t.Run("Service", func(t *testing.T) {
		var obj cdk8s.ApiObject
		is.NotPanics(func() {
			obj = cdk8s.ApiObject_Of(service.Node().DefaultChild())
		}, "Should not panic when getting default child")
		is.NotNil(obj, "Service object should not be nil")

		// Verify object metadata
		is.Equal("v1", *obj.ApiVersion(), "API version should match")
		is.Equal("Service", *obj.Kind(), "Kind should be Service")
		is.Equal("test-namespace", *obj.Metadata().Namespace(), "Namespace should match")

		// Verify service spec
		json := dry.As[map[string]any](obj.ToJson())
		is.NotNil(json, "JSON representation should not be nil")
		spec := dry.As[map[string]any](json["spec"])
		is.NotNil(spec, "Spec should not be nil")
		is.Equal("ClusterIP", spec["type"], "Service type should be ClusterIP")

		// Verify ports (should have default api and p2pv2 ports)
		ports := dry.As[[]any](spec["ports"])
		is.NotNil(ports, "Ports should not be nil")
		is.Len(ports, 2, "Should have exactly two ports")

		// Verify api port
		apiPort := dry.As[map[string]any](ports[0])
		is.Equal("api", apiPort["name"], "First port name should be api")
		is.Equal(float64(6688), apiPort["port"], "First port should be 6688")
		is.Equal("TCP", apiPort["protocol"], "First port protocol should be TCP")

		// Verify p2pv2 port
		p2pPort := dry.As[map[string]any](ports[1])
		is.Equal("p2pv2", p2pPort["name"], "Second port name should be p2pv2")
		is.Equal(float64(5001), p2pPort["port"], "Second port should be 5001")
		is.Equal("TCP", p2pPort["protocol"], "Second port protocol should be TCP")

		// Verify selector
		selector := dry.As[map[string]any](spec["selector"])
		is.NotNil(selector, "Selector should not be nil")
		is.Equal("chainlink", selector["app.kubernetes.io/name"], "Selector name should match")
		is.Equal("test-chainlink", selector["app.kubernetes.io/instance"], "Selector instance should match")
	})

	// Test Deployment configuration
	t.Run("Deployment", func(t *testing.T) {
		// Find the deployment within the chainlink chart
		var deploymentObj cdk8s.ApiObject
		for _, child := range *chainlinkChart.Node().Children() {
			if obj := cdk8s.ApiObject_Of(child); obj != nil && *obj.Kind() == "Deployment" {
				deploymentObj = obj
				break
			}
		}
		is.NotNil(deploymentObj, "Deployment object should be found")

		// Verify object metadata
		is.Equal("apps/v1", *deploymentObj.ApiVersion(), "API version should match")
		is.Equal("Deployment", *deploymentObj.Kind(), "Kind should be Deployment")
		is.Equal("test-namespace", *deploymentObj.Metadata().Namespace(), "Namespace should match")

		// Verify deployment spec
		json := dry.As[map[string]any](deploymentObj.ToJson())
		is.NotNil(json, "JSON representation should not be nil")
		spec := dry.As[map[string]any](json["spec"])
		is.NotNil(spec, "Spec should not be nil")
		is.Equal(float64(1), spec["replicas"], "Replicas should be 1")

		// Verify template spec
		template := dry.As[map[string]any](spec["template"])
		is.NotNil(template, "Template should not be nil")
		podSpec := dry.As[map[string]any](template["spec"])
		is.NotNil(podSpec, "Pod spec should not be nil")

		// Verify container configuration
		containers := dry.As[[]any](podSpec["containers"])
		is.NotNil(containers, "Containers should not be nil")
		is.Len(containers, 1, "Should have exactly one container")

		container := dry.As[map[string]any](containers[0])
		is.Equal("chainlink", container["name"], "Container name should be chainlink")
		is.Equal("chainlink/chainlink:latest", container["image"], "Container image should match")
		is.Equal("IfNotPresent", container["imagePullPolicy"], "ImagePullPolicy should match")

		// Verify command and args are built correctly
		command := dry.As[[]any](container["command"])
		is.NotNil(command, "Command should not be nil")
		is.Len(command, 1, "Should have exactly one command element")
		is.Equal("chainlink", command[0], "Command should be chainlink")

		args := dry.As[[]any](container["args"])
		is.NotNil(args, "Args should not be nil")
		expectedArgs := []string{
			"-c", "/chainlink/config/config.toml",
			"-c", "/chainlink/config/override1.toml",
			"-c", "/chainlink/config/override2.toml",
			"-s", "/chainlink/secrets/secrets.toml",
			"-s", "/chainlink/secrets/secrets-override1.toml",
			"-s", "/chainlink/secrets/secrets-override2.toml",
			"node", "start",
			"-a", "/chainlink/secrets/apicredentials",
		}
		is.Len(args, len(expectedArgs), "Should have correct number of args")
		for i, expectedArg := range expectedArgs {
			is.Equal(expectedArg, args[i], "Arg %d should match expected value", i)
		}

		// Verify container ports (should have default api and p2pv2 ports)
		ports := dry.As[[]any](container["ports"])
		is.NotNil(ports, "Container ports should not be nil")
		is.Len(ports, 2, "Should have exactly two container ports")

		// Verify api port
		apiPort := dry.As[map[string]any](ports[0])
		is.Equal("api", apiPort["name"], "First container port name should be api")
		is.Equal(float64(6688), apiPort["containerPort"], "First container port should be 6688")
		is.Equal("TCP", apiPort["protocol"], "First container port protocol should be TCP")

		// Verify p2pv2 port
		p2pPort := dry.As[map[string]any](ports[1])
		is.Equal("p2pv2", p2pPort["name"], "Second container port name should be p2pv2")
		is.Equal(float64(5001), p2pPort["containerPort"], "Second container port should be 5001")
		is.Equal("TCP", p2pPort["protocol"], "Second container port protocol should be TCP")

		// Verify volumes
		volumes := dry.As[[]any](podSpec["volumes"])
		is.NotNil(volumes, "Volumes should not be nil")
		is.Len(volumes, 2, "Should have exactly two volumes")

		volumeNames := make([]string, len(volumes))
		for i, vol := range volumes {
			volume := dry.As[map[string]any](vol)
			volumeNames[i] = volume["name"].(string)
		}
		is.ElementsMatch([]string{"config", "secrets"}, volumeNames, "Volume names should match expected")
	})
}

// TestChainlinkNodeComponentSnapshot tests the component using snapshot testing
// to capture and verify the generated Kubernetes manifests.
func TestChainlinkNodeComponentSnapshot(t *testing.T) {
	tests := []struct {
		name  string
		props *Props
	}{
		{
			name: "minimal_configuration",
			props: &Props{
				Namespace:       "test-namespace",
				AppInstanceName: "test-chainlink",
				Image:           "chainlink/chainlink:latest",
				ImagePullPolicy: "IfNotPresent",
				Config:          "LogLevel = \"info\"",
				DatabaseURL:     "postgresql://user:password@db-host:5432/chainlink",
				Replicas:        1,
			},
		},
		{
			name: "with_overrides_and_resources",
			props: &Props{
				Namespace:       "chainlink-prod",
				AppInstanceName: "chainlink-node",
				Image:           "chainlink/chainlink:2.8.0",
				ImagePullPolicy: "Always",
				Config:          "LogLevel = \"debug\"\nRootDir = \"/chainlink\"",
				ConfigOverrides: map[string]string{
					"monitoring.toml": "PrometheusPort = 9090",
					"features.toml":   "LogPoller = true",
				},
				SecretsOverrides: map[string]string{
					"api-secrets.toml": "SessionSecret = \"test-secret\"",
				},
				DatabaseURL: "postgresql://chainlink:password@postgres:5432/chainlink",
				Replicas:    3,
				EnvVars: map[string]string{
					"FEATURE_OFFCHAIN_REPORTING": "true",
					"LOG_LEVEL":                  "debug",
				},
				Resources: ResourceRequirements{
					Limits:   map[string]string{"cpu": "2", "memory": "4Gi"},
					Requests: map[string]string{"cpu": "1", "memory": "2Gi"},
				},
			},
		},
		{
			name: "with_custom_api_port",
			props: &Props{
				Namespace:       "test-namespace",
				AppInstanceName: "custom-port-node",
				Image:           "chainlink/chainlink:latest",
				ImagePullPolicy: "IfNotPresent",
				Config:          "LogLevel = \"info\"",
				DatabaseURL:     "postgresql://user:password@db-host:5432/chainlink",
				Replicas:        1,
				Ports: []ContainerPort{
					{
						Name:          "api",
						ContainerPort: 8080,
						Protocol:      "TCP",
					},
					{
						Name:          "p2pv2",
						ContainerPort: 5001,
						Protocol:      "TCP",
					},
				},
			},
		},
		{
			name: "with_automatic_postgres",
			props: &Props{
				Namespace:       "test-namespace",
				AppInstanceName: "auto-postgres-node",
				Image:           "chainlink/chainlink:latest",
				ImagePullPolicy: "IfNotPresent",
				Config:          "LogLevel = \"info\"",
				// DatabaseURL: omitted to trigger automatic postgres creation
				Replicas: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			internal.JSIIKernelMutex.Lock()
			t.Cleanup(internal.JSIIKernelMutex.Unlock)

			is := assert.New(t)

			app := internal.NewTestApp(t)
			ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

			componentFunc := Component(tt.props)
			component, err := componentFunc(ctx)
			is.NoError(err, "Component creation should not return an error")
			is.NotNil(component, "Component should not be nil")

			// Cast to Result type and test new fields
			result, ok := component.(Result)
			is.True(ok, "Component should be of type Result")

			// Test APIPort based on whether custom ports are provided
			expectedAPIPort := 6688 // default API port
			for _, port := range tt.props.Ports {
				if port.Name == "api" {
					expectedAPIPort = port.ContainerPort
					break
				}
			}

			// Test APIUrl method
			expectedAPIUrl := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", tt.props.AppInstanceName, tt.props.Namespace, expectedAPIPort)
			is.Equal(expectedAPIUrl, result.APIUrl(), "APIUrl should be constructed correctly")

			// Generate snapshots
			internal.SynthAndSnapYamls(t, app)
		})
	}
}

// TestResultType tests the Result type functionality specifically
func TestResultType(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	defer internal.JSIIKernelMutex.Unlock()

	tests := []struct {
		name        string
		props       *Props
		expectedAPI int32
	}{
		{
			name: "default_api_port",
			props: &Props{
				Namespace:       "test-namespace",
				AppInstanceName: "test-node",
				Image:           "chainlink/chainlink:latest",
				Config:          "LogLevel = \"info\"",
				DatabaseURL:     "postgresql://user:password@db-host:5432/chainlink",
				// No ports specified, should use defaults
			},
			expectedAPI: 6688,
		},
		{
			name: "custom_api_port",
			props: &Props{
				Namespace:       "test-namespace",
				AppInstanceName: "custom-node",
				Image:           "chainlink/chainlink:latest",
				Config:          "LogLevel = \"info\"",
				DatabaseURL:     "postgresql://user:password@db-host:5432/chainlink",
				Ports: []ContainerPort{
					{
						Name:          "api",
						ContainerPort: 9999,
						Protocol:      "TCP",
					},
				},
			},
			expectedAPI: 9999,
		},
		{
			name: "no_api_port_defined",
			props: &Props{
				Namespace:       "test-namespace",
				AppInstanceName: "no-api-node",
				Image:           "chainlink/chainlink:latest",
				Config:          "LogLevel = \"info\"",
				DatabaseURL:     "postgresql://user:password@db-host:5432/chainlink",
				Ports: []ContainerPort{
					{
						Name:          "metrics",
						ContainerPort: 9090,
						Protocol:      "TCP",
					},
				},
			},
			expectedAPI: 6688, // Should fall back to default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := assert.New(t)

			app := internal.NewTestApp(t)
			ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

			// Create component
			component, err := Component(tt.props)(ctx)
			is.NoError(err, "Component creation should not return an error")
			is.NotNil(component, "Component should not be nil")

			// Test Result type
			result, ok := component.(Result)
			is.True(ok, "Component should be of type Result")

			// Test APIUrl method
			expectedURL := fmt.Sprintf("http://%s.test-namespace.svc.cluster.local:%d", tt.props.AppInstanceName, tt.expectedAPI)
			is.Equal(expectedURL, result.APIUrl(), "APIUrl should be constructed correctly")

			// Test that the embedded Component is accessible
			is.NotNil(result.Component, "Embedded Component should not be nil")

			is.NotNil(result.APICredentials)
		})
	}
}

// TestAutomaticPostgresCreation tests that a postgres component is automatically created
// when DatabaseURL is not provided.
func TestAutomaticPostgresCreation(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)

	is := assert.New(t)
	must := require.New(t)

	// Setup test environment
	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

	// Test props without DatabaseURL to trigger automatic postgres creation
	testProps := &Props{
		Namespace:       "test-namespace",
		AppInstanceName: "auto-postgres-test",
		Image:           "chainlink/chainlink:latest",
		ImagePullPolicy: "IfNotPresent",
		Config:          "LogLevel = \"info\"",
		// DatabaseURL: omitted to trigger automatic postgres creation
		Replicas: 1,
	}

	// Create component
	c := Component(testProps)
	result, err := c(ctx)
	must.NoError(err, "Component creation should not return an error")
	must.NotNil(result, "Component should not be nil")

	// Cast result to our Result type
	chainlinkResult, ok := result.(Result)
	must.True(ok, "Result should be of type Result")

	// Verify postgres information is available
	must.NotNil(chainlinkResult.Postgres, "Postgres info should be populated when auto-created")
	is.Equal("auto-postgres-test-postgres", chainlinkResult.Postgres.ReleaseName, "Postgres release name should follow convention")
	is.Equal("auto-postgres-test", chainlinkResult.Postgres.Username, "Username should match app instance name")
	is.Equal("auto-postgres-test", chainlinkResult.Postgres.Database, "Database should match app instance name")
	is.Equal("staticlongpassword", chainlinkResult.Postgres.Password, "Password should be the default password")
	is.Equal("postgres", chainlinkResult.Postgres.SuperUser, "SuperUser should be postgres")
	is.Equal("staticlongpassword", chainlinkResult.Postgres.SuperUserPassword, "SuperUserPassword should be the default password")
	is.NotEmpty(chainlinkResult.Postgres.DatabaseURL, "Database URL should be generated")

	// Verify database URL format
	expectedURL := fmt.Sprintf("postgresql://auto-postgres-test:staticlongpassword@auto-postgres-test-postgres:5432/auto-postgres-test?sslmode=disable")
	is.Equal(expectedURL, chainlinkResult.Postgres.DatabaseURL, "Database URL should follow expected format")

	// Verify that a postgres chart was created
	gotCharts := lo.Map(*app.Charts(), func(c cdk8s.Chart, _ int) string {
		return crib.ExtractResource(c.Node().Id())
	})
	is.Contains(gotCharts, "sdk.HelmChart", "Should contain postgres chart")
}

// TestExternalDatabaseUsage tests that no postgres component is created
// when DatabaseURL is provided.
func TestExternalDatabaseUsage(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)

	is := assert.New(t)
	must := require.New(t)

	// Setup test environment
	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

	// Test props with explicit DatabaseURL
	testProps := &Props{
		Namespace:       "test-namespace",
		AppInstanceName: "external-db-test",
		Image:           "chainlink/chainlink:latest",
		ImagePullPolicy: "IfNotPresent",
		Config:          "LogLevel = \"info\"",
		DatabaseURL:     "postgresql://user:password@external-db:5432/chainlink",
		Replicas:        1,
	}

	// Create component
	c := Component(testProps)
	result, err := c(ctx)
	must.NoError(err, "Component creation should not return an error")
	must.NotNil(result, "Component should not be nil")

	// Cast result to our Result type
	chainlinkResult, ok := result.(Result)
	must.True(ok, "Result should be of type Result")

	// Verify postgres information is NOT available
	is.Nil(chainlinkResult.Postgres, "Postgres info should be nil when external database is used")

	// Verify that NO postgres chart was created
	gotCharts := lo.Map(*app.Charts(), func(c cdk8s.Chart, _ int) string {
		return crib.ExtractResource(c.Node().Id())
	})
	is.NotContains(gotCharts, "sdk.HelmChart", "Should NOT contain postgres chart")
}
