// Package examplev1 provides a functional sample CRIB-SDK Plan.
package examplev1

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/smartcontractkit/crib-sdk/crib"

	nginxcontrollerv1 "github.com/smartcontractkit/crib-sdk/crib/composite/cluster-services/nginx-controller/v1"
	kindv1 "github.com/smartcontractkit/crib-sdk/crib/scalar/bootstrap/kind/v1"
)

// Plan is a sample CRIB-SDK Plan. It is a functional example of how to use the SDK to create a Plan.
func Plan() *crib.Plan {
	return crib.NewPlan(
		"examplev1",
		crib.Namespace("default"),
		crib.ComponentSet(
			kindv1.Component(fmt.Sprintf("example-%s", uuid.NewString())),
			nginxcontrollerv1.Component(),
		),
	)
}
