// Package kindv1 provides a functional sample CRIB-SDK Plan.
package kindv1

import (
	"github.com/smartcontractkit/crib-sdk/crib"

	kindv1 "github.com/smartcontractkit/crib-sdk/crib/composite/bootstrap/kind/v1"
	telepresencev1 "github.com/smartcontractkit/crib-sdk/crib/composite/cluster-services/telepresence/v1"
)

// Plan is a sample CRIB-SDK Plan. It is a functional example of how to use the SDK to create a Plan.
func Plan() *crib.Plan {
	return crib.NewPlan(
		"bootstrap-kindv1",
		crib.ComponentSet(
			kindv1.Component(&kindv1.Props{
				Name: "crib",
			}),
			telepresencev1.Component(&telepresencev1.Props{
				Namespace: "ambassador",
			}),
		),
	)
}
