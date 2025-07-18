package main

import (
	"context"
	"fmt"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"

	nodev1 "github.com/smartcontractkit/crib-sdk/crib/composite/chainlink/node/v1"
)

// Nodeset Example demonstrates how to spin up a simple nodeset with n nodes
// by composing node/v1 components
func main() {
	componentFuncs := make([]crib.ComponentFunc, 0)

	ns := "crib-local"
	registryURI := "localhost:5001"

	for i := 0; i < 3; i++ {
		cFunc := nodev1.Component(&nodev1.Props{
			Namespace:       ns,
			AppInstanceName: fmt.Sprintf("chainlink-%d", i),
			Image:           fmt.Sprintf("%s/chainlink:nightly-20250715", registryURI),
			Config: dry.RemoveIndentation(`
					[WebServer]
					AllowOrigins = '\*'
					SecureCookies = false
					
					[WebServer.TLS]
					HTTPSPort = 0
				`),
			ConfigOverrides: map[string]string{
				"overrides.toml": dry.RemoveIndentation(`
				    [Log]
				    Level = 'debug'
				    JSONConsole = true
			    `),
			},
		})
		componentFuncs = append(componentFuncs, cFunc)
	}

	plan := crib.NewPlan(
		"nodesets",
		crib.Namespace(ns),
		crib.ComponentSet(
			componentFuncs...,
		),
	)

	planState, err := plan.Apply(context.Background())
	if err != nil {
		fmt.Printf("failed to apply plan: %v\n", err)
		return
	}

	nodeComponents := planState.ComponentByName(nodev1.ComponentName)

	var nodeResults []nodev1.Result

	for component := range nodeComponents {
		res := crib.ComponentState[nodev1.Result](component)
		nodeResults = append(nodeResults, res)
		fmt.Printf("Node API URL: %s\n", res.APIUrl())
		fmt.Printf("API Credentials: username: %s , password: %s\n", res.APICredentials.UserName, res.APICredentials.Password)
	}
}
