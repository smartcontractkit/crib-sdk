package jdv1

import (
	"github.com/smartcontractkit/crib-sdk/crib"

	jdv1 "github.com/smartcontractkit/crib-sdk/crib/composite/sandbox/chainlink/jd/v1"
)

const (
	cribNamespace = "crib-local"
)

// Plan is a sample CRIB-SDK Plan. It is a functional example of how to use the SDK to create a Plan.
func Plan() *crib.Plan {
	return crib.NewPlan(
		"sbx-chainlink-jdv1",
		crib.Namespace(cribNamespace),
		crib.ComponentSet(
			jdv1.Component(&jdv1.Props{
				JD: jdv1.JDProps{
					Namespace:   cribNamespace,
					ReleaseName: "jd",
				},
				DB: jdv1.DBProps{
					Namespace:   cribNamespace,
					ReleaseName: "jd-db",
					Values: map[string]any{
						"fullnameOverride": "jd-db",
					},
				},
			}),
		),
	)
}
