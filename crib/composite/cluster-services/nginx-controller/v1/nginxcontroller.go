package nginxcontrollerv1

import (
	"context"
	"fmt"
	"strings"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"

	clientsideapply "github.com/smartcontractkit/crib-sdk/crib/scalar/clientsideapply/v1"
	namespace "github.com/smartcontractkit/crib-sdk/crib/scalar/k8s/namespace/v1"
	remoteapply "github.com/smartcontractkit/crib-sdk/crib/scalar/remoteapply/v1"
)

const manifestURI = "https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml"

type Props struct{}

type Result struct {
	crib.Component

	Namespace string
}

// Validate validates the props. This is a no-op for this component.
func (p *Props) Validate(context.Context) error {
	return nil
}

// Component returns a new NginxController composite component.
func Component() crib.ComponentFunc {
	props := new(Props)
	return func(ctx context.Context) (crib.Component, error) {
		if err := props.Validate(ctx); err != nil {
			return nil, err
		}
		c, err := nginxController(ctx, props)
		return c, err
	}
}

// nginxController creates and returns a new NginxController composite component. This component simply fetches the referenced
// manifest and includes it in the generated manifest.
func nginxController(ctx context.Context, props crib.Props) (crib.Component, error) {
	parent := internal.ConstructFromContext(ctx)
	chart := cdk8s.NewChart(parent, crib.ResourceID("sdk.NginxController", props), nil)
	ctx = internal.ContextWithConstruct(ctx, chart)

	i, err := remoteapply.New(ctx, &remoteapply.Props{
		URL: manifestURI,
	})
	if err != nil {
		return nil, err
	}

	// Delete the tolerations on the Deployment to allow it to run on any node.
	for _, obj := range *i.Node().DefaultChild().Node().Children() {
		if !*cdk8s.ApiObject_IsApiObject(obj) {
			continue
		}
		o := cdk8s.ApiObject_Of(obj)
		if *o.ApiVersion() == "apps/v1" && *o.Kind() == "Deployment" {
			o.AddJsonPatch(cdk8s.JsonPatch_Remove(dry.ToPtr("/spec/template/spec/tolerations")))
			o.AddJsonPatch(cdk8s.JsonPatch_Remove(dry.ToPtr("/spec/template/spec/nodeSelector")))
		}
	}

	var ns string
	for _, obj := range *i.Node().DefaultChild().Node().Children() {
		if !*cdk8s.ApiObject_IsApiObject(obj) {
			continue
		}
		o := cdk8s.ApiObject_Of(obj)
		if *o.ApiVersion() == "v1" && *o.Kind() == "Namespace" {
			ns = *o.Metadata().Name()
			break
		}
	}

	nsConstruct, err := namespace.New(ctx, &namespace.Props{
		Namespace: ns,
	})
	if err != nil {
		return nil, err
	}

	labels := []string{
		"app.kubernetes.io/component=controller",
		"app.kubernetes.io/instance=ingress-nginx",
		"app.kubernetes.io/name=ingress-nginx",
	}
	waitFor, err := clientsideapply.New(ctx, &clientsideapply.Props{
		Namespace: ns,
		OnFailure: "abort",
		Action:    "kubectl",
		Args: []string{
			"wait",
			"-n", ns,
			"--for=condition=ready",
			"pod",
			fmt.Sprintf("-l=%s", strings.Join(labels, ",")),
			"--timeout=600s",
		},
	})
	if err != nil {
		return nil, err
	}

	i.Node().AddDependency(nsConstruct)
	waitFor.Node().AddDependency(i)
	return Result{
		Component: chart,
		Namespace: ns,
	}, nil
}
