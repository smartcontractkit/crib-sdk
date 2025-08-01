package main

import (
	"context"
	"fmt"

	"github.com/smartcontractkit/crib-sdk/crib"
	anvilv1 "github.com/smartcontractkit/crib-sdk/crib/composite/blockchain/anvil/v1"
)

// Anvil Persistence Example demonstrates how to use the anvil component with persistence enabled
func main() {
	ns := "crib-local"

	// Example 1: Basic anvil with persistence enabled
	anvilWithPersistence := anvilv1.Component(&anvilv1.Props{
		Namespace: ns,
		ChainID:   "1337",
	}, anvilv1.UsePersistence)

	// Example 2: Anvil with custom persistence configuration
	anvilWithCustomPersistence := anvilv1.Component(&anvilv1.Props{
		Namespace: ns,
		ChainID:   "31337",
	}, anvilv1.UsePersistenceWithConfig("5Gi", "fast-ssd"))

	// Example 3: Anvil with persistence and ingress
	anvilWithPersistenceAndIngress := anvilv1.Component(&anvilv1.Props{
		Namespace: ns,
		ChainID:   "9999",
	}, anvilv1.UsePersistence, anvilv1.UseIngress)

	plan := crib.NewPlan(
		"anvil-persistence-example",
		crib.Namespace(ns),
		crib.ComponentSet(
			anvilWithPersistence,
			anvilWithCustomPersistence,
			anvilWithPersistenceAndIngress,
		),
	)

	planState, err := plan.Apply(context.Background())
	if err != nil {
		fmt.Printf("failed to apply plan: %v\n", err)
		return
	}

	anvilComponents := planState.ComponentByName(anvilv1.ComponentName)

	var anvilResults []anvilv1.Result

	for component := range anvilComponents {
		res := crib.ComponentState[anvilv1.Result](component)
		anvilResults = append(anvilResults, res)
		fmt.Printf("Anvil Instance: %s\n", res.AppInstanceName())
		fmt.Printf("  HTTP RPC URL: %s\n", res.RPCHTTPURL())
		fmt.Printf("  WebSocket RPC URL: %s\n", res.RPCWebsocketURL())
		fmt.Printf("  Namespace: %s\n", res.Namespace())
		fmt.Println()
	}

	fmt.Printf("Successfully deployed %d anvil instances with persistence\n", len(anvilResults))
}
