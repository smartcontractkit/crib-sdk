package regression

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/crib-sdk/crib"
	chainlinknodev1 "github.com/smartcontractkit/crib-sdk/crib/composite/chainlink/node/v1"
	nodesetv1 "github.com/smartcontractkit/crib-sdk/crib/composite/chainlink/nodeset/v1"
	"github.com/smartcontractkit/crib-sdk/crib/scalar/charts/postgres/v1"
	helmchartv1 "github.com/smartcontractkit/crib-sdk/crib/scalar/helmchart/v1"
	namespace "github.com/smartcontractkit/crib-sdk/crib/scalar/k8s/namespace/v1"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

func TestRegressionPostgresViaChart(t *testing.T) {
	t.Setenv("CRIB_ENABLE_COMMAND_EXECUTION", "true") // Enable command execution for the test

	plan := crib.NewPlan(
		"postgres-regression-test",
		crib.Namespace("postgres-via-chart"),
		crib.ComponentSet(
			namespace.Component("postgres-via-chart"),
			postgres.Component(&helmchartv1.ChartProps{
				Namespace:   "postgres-via-chart",
				ReleaseName: "shared-postgres",
				Values: map[string]any{
					"fullnameOverride": "shared-postgres",
					"auth": map[string]any{
						"enablePostgresUser": true,
						"postgresPassword":   "postgres",
					},
					"primary": map[string]any{
						"persistence": map[string]any{
							"enabled": false,
						},
						"initdb": map[string]any{
							"scripts": map[string]string{
								"init.sql": generateInitSQL(3), // Generate SQL for 3 Chainlink nodes
							},
						},
					},
				},
			}),
		),
	)

	_, err := plan.Apply(t.Context())
	assert.NoError(t, err, "Failed to apply the regression test plan")
}

func TestRegressionPostgresViaNodeset(t *testing.T) {
	t.Setenv("CRIB_ENABLE_COMMAND_EXECUTION", "true") // Enable command execution for the test
	t.Setenv("DEBUG", "jsii*")                        // Enable JSII debugging
	t.Setenv("JSII_DEBUG_FILE", "/tmp/jsii.log")      // Set JSII debug log file

	var nodesConfigs []*chainlinknodev1.Props
	for i := range 3 {
		nodesConfigs = append(nodesConfigs, &chainlinknodev1.Props{
			AppInstanceName: fmt.Sprintf("node-%d", i),
			Image:           "public.ecr.aws/chainlink/chainlink:2.26.0-beta.0",
			ImagePullPolicy: "IfNotPresent",
			Config: dry.RemoveIndentation(`
		[[EVM]]
			LinkContractAddress = '0x1234567890abcdef1234567890abcdef12345678'
			ChainID = '%s'
			MinIncomingConfirmations = 1
			MinContractPayment = '0.0000001 link'
			FinalityDepth = %d

		[[EVM.Nodes]]
			Name = 'default'
			WsUrl = 'wss://foo'
			HttpUrl = 'https://foo'
		[Log]
			JSONConsole = true
			Level = 'debug'
		[Pyroscope]
			ServerAddress = 'http://host.docker.internal:4040'
			Environment = 'local'
		[WebServer]
			SessionTimeout = '999h0m0s'
			HTTPWriteTimeout = '3m'
			SecureCookies = false
			HTTPPort = 6688
		[WebServer.TLS]
			HTTPSPort = 0
		[WebServer.RateLimit]
			Authenticated = 5000
			Unauthenticated = 5000
		[JobPipeline]
		[JobPipeline.HTTPRequest]
			DefaultTimeout = '1m'
		[Log.File]
			MaxSize = '0b'
		[Feature]
			FeedsManager = true
			LogPoller = true
			UICSAKeys = true
		[OCR2]
			Enabled = true
			SimulateTransactions = false
			DefaultTransactionQueueDepth = 1
		[P2P.V2]
			Enabled = true
			ListenAddresses = ['0.0.0.0:6690']
`),
			Ports: []chainlinknodev1.ContainerPort{
				{
					Name:          "api",
					Protocol:      "TCP",
					ContainerPort: 6688,
				},
				{
					Name:          "p2pv2",
					Protocol:      "TCP",
					ContainerPort: 6690,
				},
			},
		})
	}

	plan := crib.NewPlan(
		"postgres-regression-test",
		crib.Namespace("postgres-via-nodeset"),
		crib.ComponentSet(
			namespace.Component("postgres-via-nodeset"),
			nodesetv1.Component(&nodesetv1.Props{
				Namespace: "postgres-via-nodeset",
				NodeProps: nodesConfigs,
				Size:      3, // Number of Chainlink nodes
			}),
		),
	)

	_, err := plan.Apply(t.Context())
	assert.NoError(t, err, "Failed to apply the regression test plan")
}

// generateInitSQL creates the SQL script to initialize databases and users for each Chainlink node.
func generateInitSQL(num int) string {
	var sqlStatements []string

	for i := 0; i < num; i++ {
		dbName := fmt.Sprintf("chainlink_node_%d", i)
		username := fmt.Sprintf("chainlink_user_%d", i)
		password := fmt.Sprintf("chainlink_pass_%d", i)

		// Create user
		//nolint:gocritic // for readability
		sqlStatements = append(sqlStatements, fmt.Sprintf(
			"CREATE USER %s WITH PASSWORD '%s';", username, password))

		// Create database
		sqlStatements = append(sqlStatements, fmt.Sprintf(
			"CREATE DATABASE %s OWNER %s;", dbName, username))

		// Grant privileges
		sqlStatements = append(sqlStatements, fmt.Sprintf(
			"GRANT ALL PRIVILEGES ON DATABASE %s TO %s;", dbName, username))
	}

	return strings.Join(sqlStatements, "\n")
}
