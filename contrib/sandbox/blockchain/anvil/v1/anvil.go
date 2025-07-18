// Package anvilv1 provides a functional sample CRIB-SDK Plan.
package anvilv1

import (
	"fmt"

	"github.com/smartcontractkit/crib-sdk/crib"

	chainsv1 "github.com/smartcontractkit/crib-sdk/crib/composite/sandbox/blockchain/chains/v1"
	blockchainexplorerv1 "github.com/smartcontractkit/crib-sdk/crib/composite/sandbox/blockchain/explorer/v1"
	blockchainnodev1 "github.com/smartcontractkit/crib-sdk/crib/composite/sandbox/blockchain/node/v1"
)

const (
	cribNamespace = "crib-local"
)

// Plan is a sample CRIB-SDK Plan. It is a functional example of how to use the SDK to create a Plan.
func Plan() *crib.Plan {
	return crib.NewPlan(
		"sbx-blockchain-anvilv1",
		crib.Namespace(cribNamespace),
		crib.ComponentSet(
			chainsv1.Component([]*chainsv1.BlockchainConfig{
				{
					NodeSet: []*blockchainnodev1.Props{
						{
							Type:        blockchainnodev1.BlockchainTypeAnvil,
							Namespace:   cribNamespace,
							ReleaseName: "anvil-1337",
							ChainID:     "1337",
							Values: map[string]any{
								"service": map[string]any{
									"name": "anvil-1337",
								},
							},
							ValuesPatches: [][]string{
								{"containers[0].env[0].value", "1337"},
							},
						},
					},
					Explorer: &blockchainexplorerv1.Props{
						Type:        blockchainexplorerv1.ExplorerTypeOtterscan,
						Namespace:   cribNamespace,
						ReleaseName: "ots-1337",
						Values:      make(map[string]any),
						ValuesPatches: [][]string{
							{"containers[0].env[0].value", fmt.Sprintf("http://anvil-1337.%s.svc.cluster.local:8545", cribNamespace)},
						},
					},
				},
			}, chainsv1.ParallelProcessing),
		),
	)
}
