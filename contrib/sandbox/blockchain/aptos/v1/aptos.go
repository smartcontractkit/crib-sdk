// Package aptosv1 provides a functional sample CRIB-SDK Plan for Aptos blockchain.
package aptosv1

import (
	"github.com/smartcontractkit/crib-sdk/crib"

	chainsv1 "github.com/smartcontractkit/crib-sdk/crib/composite/sandbox/blockchain/chains/v1"
	blockchainnodev1 "github.com/smartcontractkit/crib-sdk/crib/composite/sandbox/blockchain/node/v1"
)

const (
	cribNamespace = "crib-local"
)

// Plan is a sample CRIB-SDK Plan for Aptos blockchain. It demonstrates how to use the SDK to create a Plan with a single Aptos instance.
func Plan() *crib.Plan {
	return crib.NewPlan(
		"sbx-blockchain-aptosv1",
		crib.Namespace(cribNamespace),
		crib.ComponentSet(
			chainsv1.Component([]*chainsv1.BlockchainConfig{
				{
					NodeSet: []*blockchainnodev1.Props{
						{
							Type:        blockchainnodev1.BlockchainTypeAptos,
							Namespace:   cribNamespace,
							ReleaseName: "aptos",
							ChainID:     "1",
							Values:      make(map[string]any),
							ValuesPatches: [][]string{
								{"containers[0].image", "ghcr.io/friedemannf/aptos-tools:latest"}, // image for local kind on arm64
							},
						},
					},
				},
			}),
		),
	)
}
