package main

import (
	"context"
	"fmt"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"

	nodev1 "github.com/smartcontractkit/crib-sdk/crib/composite/chainlink/node/v1"
	nodesetv1 "github.com/smartcontractkit/crib-sdk/crib/composite/chainlink/nodeset/v1"
)

// NodeSetExample demonstrates how to use the NodeSet composite component
// to create multiple Chainlink nodes with a shared PostgreSQL database.
func main() {
	ns := "crib-local"

	// when deploying to kind it requires pre-pulling and pushing images to private registry
	image := "localhost:5001/chainlink:nightly-20250715"
	nodeProps := []*nodev1.Props{
		// Node 0 - Standard configuration
		{
			AppInstanceName: "chainlink-node-0",
			Config: dry.RemoveIndentation(`
				[Log]
				Level = 'warn'
			
				[WebServer]
				AllowOrigins = '\*'
				HTTPWriteTimeout = '10m'
				HTTPPort = 6688
				SecureCookies = false
			
				[WebServer.TLS]
				HTTPSPort = 0
			
				[[EVM]]
				ChainID = '1337'
			
				[[EVM.Nodes]]
				Name = 'primary'
				WSURL = 'ws://anvil:8546'
				HTTPURL = 'http://anvil:8545'
			`),
			Image: image,
		},
		// Node 1 - With custom ports
		{
			AppInstanceName: "chainlink-node-1",
			Image:           image,
			Config: dry.RemoveIndentation(`
				[Log]
				Level = 'info'
				
				[WebServer]
				AllowOrigins = '*'
				HTTPWriteTimeout = '10m'
				HTTPPort = 6689
				SecureCookies = false
				
				[WebServer.TLS]
				HTTPSPort = 0
				
				[[EVM]]
				ChainID = '1337'
				
				[[EVM.Nodes]]
				Name = 'primary'
				WSURL = 'ws://anvil:8546'
				HTTPURL = 'http://anvil:8545'`),
			Ports: []nodev1.ContainerPort{
				{
					Name:          "api",
					ContainerPort: 6689,
					Protocol:      "TCP",
				},
				{
					Name:          "p2pv2",
					ContainerPort: 5002,
					Protocol:      "TCP",
				},
			},
		},
		// Node 2 - With resource limits and custom configuration
		{
			AppInstanceName: "chainlink-node-2",
			Image:           image,
			Config: dry.RemoveIndentation(`
				[Log]
				Level = 'debug'
				
				[WebServer]
				AllowOrigins = '*'
				HTTPWriteTimeout = '15m'
				HTTPPort = 6690
				SecureCookies = false
				
				[WebServer.TLS]
				HTTPSPort = 0
				
				[[EVM]]
				ChainID = '1337'
				
				[[EVM.Nodes]]
				Name = 'primary'
				WSURL = 'ws://anvil:8546'
				HTTPURL = 'http://anvil:8545'`),
			Ports: []nodev1.ContainerPort{
				{
					Name:          "api",
					ContainerPort: 6690,
					Protocol:      "TCP",
				},
				{
					Name:          "p2pv2",
					ContainerPort: 5003,
					Protocol:      "TCP",
				},
			},
			Resources: nodev1.ResourceRequirements{
				Limits: map[string]string{
					"cpu":    "2",
					"memory": "4096Mi",
				},
				Requests: map[string]string{
					"cpu":    "1",
					"memory": "512Mi",
				},
			},
			ConfigOverrides: map[string]string{
				"debug.toml": dry.RemoveIndentation(`
					[Log]
					Level = 'debug'
					JSONConsole = true`),
			},
			SecretsOverrides: map[string]string{
				"additional-secrets.toml": dry.RemoveIndentation(`
					[EVM]
					[[EVM.Keys]]
					JSON = '{"address":"4c59bc9ef776e68ec9eb4b07ba678e853c72b23a","crypto":{"cipher":"aes-128-ctr","ciphertext":"3223b4061de6eb7f2c0034ce4973b06838e41d7789bac4961d5762504486055d","cipherparams":{"iv":"1853b2ef246c32c56cff44c73d2bb1ce"},"kdf":"scrypt","kdfparams":{"dklen":32,"n":262144,"p":1,"r":8,"salt":"53275663a60f32fa17ba95cde36f908aec32adc17447971cebf3fb09ef0da85d"},"mac":"4c0a57eb999f4f91eccd78ba202796f7db6f03b2175e9e54902df11a2b4ea6c4"},"id":"00000000-0000-0000-0000-000000000000","version":3}'
					Password = ''
					ID = 1337
				`),
			},
		},
	}

	// Create a NodeSet with 3 Chainlink nodes and a shared PostgreSQL database
	component := nodesetv1.Component(&nodesetv1.Props{
		Namespace: ns,
		NodeProps: nodeProps,
		Size:      len(nodeProps),
	})

	plan := crib.NewPlan(
		"nodeset",
		crib.Namespace(ns),
		crib.ComponentSet(
			component,
		),
	)

	planState, err := plan.Apply(context.Background())
	if err != nil {
		fmt.Printf("failed to apply plan: %v\n", err)
		return
	}

	nodesetComponent := planState.ComponentByName(nodesetv1.ComponentName)

	for c := range nodesetComponent {
		res := crib.ComponentState[nodesetv1.Result](c)

		for _, node := range res.Nodes {
			fmt.Printf("Node API URL: %s\n", node.APIUrl())
			fmt.Printf("API Credentials: username: %s , password: %s\n", node.APICredentials.UserName, node.APICredentials.Password)
		}
	}

	for c := range nodesetComponent {
		fmt.Printf("nodeset with id: %d\n", c.Node().Id())
	}
}
