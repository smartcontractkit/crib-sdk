package iresolver

import (
	"cmp"
	"slices"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/samber/lo"

	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/infra"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
)

// NameResolver is a default resolver function that ensures that CDK8s produces a valid DNS-1035 label for the
// metadata.name field of a Kubernetes resource.
func NameResolver(ctx cdk8s.ResolutionContext) {
	keys := lo.Map(*ctx.Key(), func(k *string, _ int) string {
		return dry.FromPtr(k)
	})
	if dry.FromPtr(ctx.Replaced()) {
		return // Skip if already replaced.
	}
	// Only inspect the metadata.name field.
	if !slices.Equal(keys, []string{"metadata", "name"}) {
		return
	}
	currentName := dry.As[string](ctx.Value())

	// Skip if the name is a magic cdk8s string.
	magicResource := cmp.Or(
		currentName == domain.CDK8sDefault,
		currentName == domain.CDK8sUnknown,
		currentName == domain.CDK8sResource,
	)
	if magicResource {
		return // Skip magic strings.
	}

	newName := infra.ToRFC1123(currentName) // Convert to RFC1123 format.
	if currentName == newName {
		return // No change needed, return early.
	}
	ctx.ReplaceValue(newName)
}
