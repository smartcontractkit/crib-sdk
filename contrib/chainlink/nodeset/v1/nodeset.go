package v1

import (
	"fmt"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"

	chainlinknodev1 "github.com/smartcontractkit/crib-sdk/crib/composite/chainlink/node/v1"
	nodesetv1 "github.com/smartcontractkit/crib-sdk/crib/composite/chainlink/nodeset/v1"
)

const (
	cribNamespace = "crib-local"
)

// Plan is a sample CRIB-SDK Plan demonstrating how to use the nodeset v1 component
// to create multiple Chainlink nodes with a shared PostgreSQL database.
func Plan() *crib.Plan {
	config := dry.RemoveIndentation(`
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
				`)

	propsSlice := make([]*chainlinknodev1.Props, 0)
	for i := 0; i < 5; i++ {
		props := &chainlinknodev1.Props{
			// Namespace is intentionally left empty - nodeset component will set it
			Image:           "localhost:5001/chainlink:nightly-20250624-plugins",
			AppInstanceName: fmt.Sprintf("%s-%d", "chainlink", i),
			// passing as config not as override
			Config: config,
			// todo test with secret overrides
			// SecretsOverrides: map[string]string{
			//	"overrides": *secrets,
			// },
		}
		propsSlice = append(propsSlice, props)
	}

	return crib.NewPlan(
		"chainlink-nodesetv1",
		crib.Namespace(cribNamespace),
		crib.ComponentSet(
			// NodeSet component that creates multiple Chainlink nodes with shared PostgreSQL
			nodesetv1.Component(&nodesetv1.Props{
				Namespace: cribNamespace,
				Size:      len(propsSlice), // Create 3 Chainlink nodes
				NodeProps: propsSlice,
			}),
		),
	)
}
