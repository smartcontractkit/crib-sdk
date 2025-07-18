// Package anvilv1 provides a functional sample CRIB-SDK Plan.
package anvilv1

import (
	"github.com/smartcontractkit/crib-sdk/crib"

	anvilv1 "github.com/smartcontractkit/crib-sdk/crib/composite/blockchain/anvil/v1"
)

// Plan is a sample CRIB-SDK Plan. It is a functional example of how to use the SDK to create a Plan.
func Plan() *crib.Plan {
	namespace := "crib-local"
	return crib.NewPlan(
		"blockchain-anvilv1",
		crib.Namespace(namespace),
		crib.ComponentSet(
			anvilv1.Component(&anvilv1.Props{
				Namespace: namespace,
				ChainID:   "1337",
			}),
			anvilv1.Component(&anvilv1.Props{
				Namespace: namespace,
				ChainID:   "2337",
			}),
		),
	)
}
