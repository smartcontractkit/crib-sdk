package iresolver

import (
	"slices"
	"unique"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/samber/lo"

	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

// ImagePullSecretResolver is a resolver function that sets image pull secrets to the Kubernetes resources that
// support it.
//
// Due to the way CDK8s works, this resolver overwrites the image pull secrets for the specified resources.
func ImagePullSecretResolver(secrets ...string) func(ctx cdk8s.ResolutionContext) {
	changedEntries := make(map[unique.Handle[string]]struct{})
	return func(ctx cdk8s.ResolutionContext) {
		if len(secrets) == 0 {
			return // No-op if no secret is provided.
		}
		keys := lo.Map(*ctx.Key(), func(k *string, _ int) string {
			return dry.FromPtr(k)
		})
		obj := ctx.Obj()
		objRef := unique.Make(*obj.ToString())

		// If the object has already been changed, we skip it.
		if _, changed := changedEntries[objRef]; changed {
			return
		}

		var path string
		switch {
		case slices.Equal(keys, []string{"spec"}):
			supportedKinds := []string{
				"Pod",
				"Job",
				"CronJob",
				"Workflow", // ArgoCD
			}
			if !slices.Contains(supportedKinds, *obj.Kind()) {
				return
			}

			path = "/spec/imagePullSecrets"
		case slices.Equal(keys, []string{"spec", "template", "spec"}):
			supportedKinds := []string{
				"Deployment",
				"StatefulSet",
				"DaemonSet",
				"DeploymentConfig",
			}
			if !slices.Contains(supportedKinds, *obj.Kind()) {
				return
			}

			path = "/spec/template/spec/imagePullSecrets"
		default:
			return // No-op if the keys do not match the expected patterns.
		}

		changedEntries[objRef] = struct{}{}
		secretObj := lo.Map(secrets, func(secret string, _ int) map[string]string {
			return map[string]string{"name": secret}
		})
		obj.AddJsonPatch(cdk8s.JsonPatch_Add(dry.ToPtr(path), secretObj))
		ctx.SetReplacedValue(obj.ToJson())
	}
}
