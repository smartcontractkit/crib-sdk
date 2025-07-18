package jdv1

import (
	"github.com/smartcontractkit/crib-sdk/crib"

	jdv1 "github.com/smartcontractkit/crib-sdk/crib/composite/chainlink/jd/v1"
)

const (
	cribNamespace = "crib-local"
)

// Plan is a sample CRIB-SDK Plan. It is a functional example of how to use the SDK to create a Plan.
func Plan() *crib.Plan {
	return crib.NewPlan(
		"chainlink-jdv1",
		crib.Namespace(cribNamespace),
		crib.ComponentSet(
			jdv1.Component(&jdv1.Props{
				Namespace:      cribNamespace,
				WaitForRollout: true,
				JD: jdv1.JDProps{
					// Use local registry image
					Image:            "localhost:5001/job-distributor:0.12.7",
					CSAEncryptionKey: "d1093c0060d50a3c89c189b2e485da5a3ce57f3dcb38ab7e2c0d5f0bb2314a44",
				},
			}),
		),
	)
}
