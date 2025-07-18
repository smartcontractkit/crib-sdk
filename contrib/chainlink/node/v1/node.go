package nodev1

import (
	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/crib/scalar/helmchart/v1"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"

	chainlinknodev1 "github.com/smartcontractkit/crib-sdk/crib/composite/chainlink/node/v1"
	postgresv1 "github.com/smartcontractkit/crib-sdk/crib/scalar/charts/postgres/v1"
)

const (
	cribNamespace = "crib-local"
)

// Plan is a sample CRIB-SDK Plan. It is a functional example of how to use the SDK to create a Plan.
func Plan() *crib.Plan {
	return crib.NewPlan(
		"chainlink-nodev1",
		crib.Namespace(cribNamespace),
		crib.ComponentSet(
			// PostgreSQL database component using Bitnami PostgreSQL chart
			postgresv1.Component(&helmchart.ChartProps{
				Name:        "postgres",
				Namespace:   cribNamespace,
				ReleaseName: "node-db-0",
				Values: map[string]any{
					"fullnameOverride": "node-db-0",
					"auth": map[string]any{
						"enablePostgresUser": true,
						"postgresPassword":   "postgres",
						"username":           "chainlink1",
						"password":           "simplelongpassword",
						"database":           "chainlink",
					},
					"primary": map[string]any{
						"persistence": map[string]any{
							"enabled": false,
						},
					},
				},
			}),
			// Chainlink node component with proper database connection
			// Uses default ports: api (6688), p2pv2 (5001)
			// To customize ports, add: Ports: []chainlinknodev1.ContainerPort{{Name: "api", ContainerPort: 8080, Protocol: "TCP"}}
			chainlinknodev1.Component(&chainlinknodev1.Props{
				Namespace:       cribNamespace,
				AppInstanceName: "node-0",
				Image:           "public.ecr.aws/chainlink/chainlink:2.24.0",
				// Use config below for testing in Kind
				// Requires pushing image to kind registry
				// Image:           "localhost:5001/chainlink:nightly-20250624-plugins",
				DatabaseURL: "postgresql://chainlink1:simplelongpassword@node-db-0:5432/chainlink?sslmode=disable",
				Config: dry.RemoveIndentation(`
					[Database]
					MaxIdleConns = 20
					MaxOpenConns = 40
					MigrateOnStartup = true

					[Log]
					Level = 'info'
					JSONConsole = true

					[Log.File]
					MaxSize = '0b'

					[WebServer]
					AllowOrigins = '*'
					HTTPPort = 6688
					SecureCookies = false

					[WebServer.RateLimit]
					Authenticated = 2000
					Unauthenticated = 100

					[WebServer.TLS]
					HTTPSPort = 0

					# network.toml

					[[EVM]]
					ChainID = '1337'

					[[EVM.Nodes]]
					Name = 'primary'
					WSURL = 'ws://anvil-1337:8546'
					HTTPURL = 'http://anvil-1337:8545'
				`),
			}),
		),
	)
}
