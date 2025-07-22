package telepresencev1

import (
	"context"
	"fmt"
	"strings"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2/k8s"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"

	telepresence "github.com/smartcontractkit/crib-sdk/crib/scalar/charts/telepresence/v1"
	clientsideapply "github.com/smartcontractkit/crib-sdk/crib/scalar/clientsideapply/v1"
	helmchart "github.com/smartcontractkit/crib-sdk/crib/scalar/helmchart/v1"
	rolev1 "github.com/smartcontractkit/crib-sdk/crib/scalar/k8s/role/v1"
	rolebindingv1 "github.com/smartcontractkit/crib-sdk/crib/scalar/k8s/rolebinding/v1"
	serviceaccountv1 "github.com/smartcontractkit/crib-sdk/crib/scalar/k8s/serviceaccount/v1"
)

const ComponentName = "sdk.composite.telepresence.v1"

type Props struct {
	Namespace string `validate:"required"`
	// QuitBeforeRunning quits existing telepresence processes on the user machine
	QuitBeforeRunning bool
}

// Validate validates the props.
func (p *Props) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(p)
}

// Component returns a new telepresence composite component.
func Component(props *Props) crib.ComponentFunc {
	return func(ctx context.Context) (crib.Component, error) {
		if err := props.Validate(ctx); err != nil {
			return nil, err
		}
		return telepresenceComposite(ctx, props)
	}
}

// telepresenceComposite creates and returns a new telepresence composite component.
// This component installs telepresence and connects to it.
func telepresenceComposite(ctx context.Context, props crib.Props) (crib.Component, error) {
	telepresenceProps := dry.MustAs[*Props](props)

	parent := internal.ConstructFromContext(ctx)
	resourceID := crib.ResourceID(ComponentName, props)
	chart := cdk8s.NewChart(parent, resourceID, nil)
	ctx = internal.ContextWithConstruct(ctx, chart)

	// since we're disabling managerRBAC: false, we need to generate RBACs ourselves
	err := CreateNamespaceLocalRBACs(ctx, telepresenceProps)
	if err != nil {
		return nil, dry.Wrapf(err, "failed to create custom RBACs for traffic-manager")
	}

	// Create telepresence component
	telepresenceComponent, err := telepresence.Component(&helmchart.ChartProps{
		Namespace:   telepresenceProps.Namespace,
		ReleaseName: "traffic-manager",
		Flags:       []string{"--skip-tests", "--no-hooks"},
		Values: map[string]any{
			// Deploying individual traffic manager per namespace, to improve reliability and make RBAC management easier.
			"namespaces": []string{telepresenceProps.Namespace},
			// Disable manageRBAC, in the current version of telepresence chart 2.23.3
			// it creates ClusterRole and ClusterRoleBinding, which are not necessary, for simple one way routing
			// and would require cluster admin permissions
			"managerRbac": map[string]any{
				"create": false,
			},
			// disabling agentInjector as it is not required for exposing services to dev machines
			"agentInjector": map[string]any{
				"enabled": false,
			},
		},
	})(ctx)
	if err != nil {
		return nil, err
	}

	labels := []string{
		"app=traffic-manager",
		"telepresence=manager",
	}
	waitForTelepresence, err := clientsideapply.New(ctx, &clientsideapply.Props{
		Namespace: telepresenceProps.Namespace,
		OnFailure: "abort",
		Action:    "kubectl",
		Args: []string{
			"wait",
			"-n", telepresenceProps.Namespace,
			"--for=condition=available",
			"deployment",
			fmt.Sprintf("-l=%s", strings.Join(labels, ",")),
			"--timeout=600s",
		},
	})
	if err != nil {
		return nil, dry.Wrapf(err, "failed to wait for telepresence")
	}

	if telepresenceProps.QuitBeforeRunning {
		// Create client side apply to quit existing telepresence
		quitTelepresenceClientSideApply, quitErr := clientsideapply.New(ctx, &clientsideapply.Props{
			Namespace: telepresenceProps.Namespace,
			OnFailure: "abort",
			Action:    "telepresence",
			Args:      []string{"quit", "--stop-daemons"},
		})
		if quitErr != nil {
			return nil, dry.Wrapf(err, "failed to quit telepresence")
		}

		telepresenceComponent.Node().AddDependency(crib.ComponentState[*clientsideapply.Result](quitTelepresenceClientSideApply).Component)
	}

	// client side apply to connect to telepresence
	connectTelepresence, err := clientsideapply.New(ctx, &clientsideapply.Props{
		Namespace: "default",
		OnFailure: "abort",
		Action:    "telepresence",
		Args: []string{
			"connect",
			"--namespace",
			telepresenceProps.Namespace,
			"--mapped-namespaces",
			telepresenceProps.Namespace,
			"--manager-namespace",
			telepresenceProps.Namespace,
		},
	})
	if err != nil {
		return nil, dry.Wrapf(err, "failed to connect telepresence")
	}

	// Set up dependencies: telepresence -> wait -> connect
	waitForTelepresence.Node().AddDependency(telepresenceComponent)
	connectTelepresence.Node().AddDependency(crib.ComponentState[*clientsideapply.Result](waitForTelepresence).Component)

	return connectTelepresence, nil
}

// CreateNamespaceLocalRBACs provides RBACs based on the output from helm template for the 2.23 version of the chart.
func CreateNamespaceLocalRBACs(ctx context.Context, telepresenceProps *Props) error {
	_, err := rolev1.New(ctx, &rolev1.Props{
		Namespace:   telepresenceProps.Namespace,
		AppName:     "traffic-manager",
		AppInstance: "traffic-manager",
		Name:        "traffic-manager",
		Rules: []*k8s.PolicyRule{
			{
				ApiGroups: dry.PtrSlice([]string{""}),
				Resources: dry.PtrSlice([]string{"services"}),
				Verbs:     dry.PtrSlice([]string{"create", "update"}),
			},
			{
				ApiGroups: dry.PtrSlice([]string{""}),
				Resources: dry.PtrSlice([]string{"services", "pods"}),
				Verbs:     dry.PtrSlice([]string{"list", "get", "watch"}),
			},
			{
				ApiGroups: dry.PtrSlice([]string{""}),
				Resources: dry.PtrSlice([]string{"pods/log"}),
				Verbs:     dry.PtrSlice([]string{"get"}),
			},
			{
				ApiGroups:     dry.PtrSlice([]string{""}),
				Resources:     dry.PtrSlice([]string{"configmaps"}),
				Verbs:         dry.PtrSlice([]string{"list", "get", "watch"}),
				ResourceNames: dry.PtrSlice([]string{"traffic-manager"}),
			},
			{
				ApiGroups: dry.PtrSlice([]string{"apps"}),
				Resources: dry.PtrSlice([]string{"deployments", "replicasets", "statefulsets"}),
				Verbs:     dry.PtrSlice([]string{"get", "list", "watch"}),
			},
			{
				ApiGroups: dry.PtrSlice([]string{"events.k8s.io"}),
				Resources: dry.PtrSlice([]string{"events"}),
				Verbs:     dry.PtrSlice([]string{"get", "watch"}),
			},
			{
				ApiGroups:     dry.PtrSlice([]string{""}),
				Resources:     dry.PtrSlice([]string{"namespaces"}),
				ResourceNames: dry.PtrSlice([]string{telepresenceProps.Namespace}),
				Verbs:         dry.PtrSlice([]string{"get"}),
			},
		},
	})
	if err != nil {
		return dry.Wrapf(err, "failed to create Role")
	}

	_, err = rolebindingv1.New(ctx, &rolebindingv1.Props{
		Namespace:   telepresenceProps.Namespace,
		AppName:     "traffic-manager",
		AppInstance: "traffic-manager",
		Name:        "traffic-manager",
		RoleRef: &k8s.RoleRef{
			ApiGroup: dry.ToPtr("rbac.authorization.k8s.io"),
			Kind:     dry.ToPtr("Role"),
			Name:     dry.ToPtr("traffic-manager"),
		},
		Subjects: []*k8s.Subject{
			{
				Kind:      dry.ToPtr("ServiceAccount"),
				Name:      dry.ToPtr("traffic-manager"),
				Namespace: dry.ToPtr(telepresenceProps.Namespace),
			},
		},
	})
	if err != nil {
		return dry.Wrapf(err, "failed to create RoleBinding")
	}

	_, err = serviceaccountv1.New(ctx, &serviceaccountv1.Props{
		Namespace:   telepresenceProps.Namespace,
		AppName:     "traffic-manager",
		AppInstance: "traffic-manager",
		Name:        "traffic-manager",
	})
	if err != nil {
		return dry.Wrapf(err, "failed to create ServiceAccount")
	}
	return nil
}
