package awsv1

import (
	"github.com/smartcontractkit/crib-sdk/crib"

	awsv1 "github.com/smartcontractkit/crib-sdk/crib/composite/bootstrap/aws/v1"
)

// Plan demonstrates how to use the AWS bootstrap composite component
// to perform AWS SSO login, switch cluster context, and switch namespace.
func Plan() *crib.Plan {
	return crib.NewPlan(
		"bootstrap-awsv1",
		crib.ComponentSet(
			awsv1.Component(&awsv1.Props{
				Profile:   "sand",
				Cluster:   "main-sand-cluster",
				Namespace: "default",
			}),
		),
	)
}
