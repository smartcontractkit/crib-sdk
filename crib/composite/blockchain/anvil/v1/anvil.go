package anvilv1

import (
	"context"
	"fmt"

	"github.com/aws/jsii-runtime-go"
	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2/k8s"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"

	deploymentv1 "github.com/smartcontractkit/crib-sdk/crib/composite/deployment/v1"
	statefulsetv1 "github.com/smartcontractkit/crib-sdk/crib/composite/statefulset/v1"
	workloadv1 "github.com/smartcontractkit/crib-sdk/crib/composite/workload/v1"
)

const (
	ComponentName = "sdk.composite.blockchain.anvil.v1"
	// In anvil the same servicePort is used for both Http and Websocket connections
	// The protocol (HTTP vs WebSocket) is determined by the type of client request.
	servicePort = 8545
)

type (
	// Props contains the configuration for a blockchain component.
	Props struct {
		Namespace   string
		ChainID     string
		ingress     IngressProps
		persistence PersistenceProps
	}

	// IngressProps contains the configuration for ingress.
	IngressProps struct {
		enabled bool
	}

	// PersistenceProps contains the configuration for persistence.
	PersistenceProps struct {
		enabled bool
	}

	PropOpt func(p *Props)

	Result struct {
		crib.Component
		namespace       string
		appInstanceName string
	}
)

// UseIngress enables an Ingress for the deployed Anvil instance. The default
// behavior is to expose the Anvil instance via a Kubernetes Service.
func UseIngress(p *Props) {
	p.ingress.enabled = true
}

// UsePersistence enables persistence for anvil instance. It deploys a Statefulset with mount PVCs for storage
// The default behavior is to use Deployment resource without persistence storage.
// Enabling persistence can only work in cloud as kind doesn't provide persistence.
func UsePersistence(p *Props) {
	p.persistence.enabled = true
}

// Validate validates the props. This is a no-op for this component.
func (p *Props) Validate(ctx context.Context) error {
	return nil
}

// RPCWebsocketURL returns cluster local websocket URL.
func (r *Result) RPCWebsocketURL() string {
	return domain.ClusterLocalServiceURL("ws", r.appInstanceName, r.namespace, servicePort)
}

// RPCHTTPURL returns cluster local http URL.
func (r *Result) RPCHTTPURL() string {
	return domain.ClusterLocalServiceURL("http", r.appInstanceName, r.namespace, servicePort)
}

// Component returns a new Anvil composite component.
func Component(props *Props, opts ...PropOpt) crib.ComponentFunc {
	props = dry.When(props != nil, props, &Props{})
	return func(ctx context.Context) (crib.Component, error) {
		for _, opt := range opts {
			opt(props)
		}
		if err := props.Validate(ctx); err != nil {
			return nil, err
		}
		result, err := blockchain(ctx, props)
		return result, err
	}
}

// blockchain creates and returns a new blockchain composite component.
func blockchain(ctx context.Context, props crib.Props) (crib.Component, error) {
	parent := internal.ConstructFromContext(ctx)
	chart := cdk8s.NewChart(parent, crib.ResourceID(ComponentName, props), nil)
	ctx = internal.ContextWithConstruct(ctx, chart)

	anvilProps := dry.MustAs[*Props](props)

	commandArgs := `if [ ! -f ${ANVIL_STATE_PATH} ]; then
  echo "No state found, creating new state"
  anvil --host ${ANVIL_HOST} --port ${ANVIL_PORT} --chain-id ${ANVIL_CHAIN_ID} --block-time ${ANVIL_BLOCK_TIME} --dump-state ${ANVIL_STATE_PATH}
else
  echo "State found, loading state"
  anvil --host ${ANVIL_HOST} --port ${ANVIL_PORT} --chain-id ${ANVIL_CHAIN_ID} --block-time ${ANVIL_BLOCK_TIME} --dump-state ${ANVIL_STATE_PATH} --load-state ${ANVIL_STATE_PATH}
fi`

	envVars := []domain.EnvVar{
		{Name: "ANVIL_CHAIN_ID", Value: anvilProps.ChainID},
		{Name: "ANVIL_HOST", Value: "0.0.0.0"},
		{Name: "ANVIL_PORT", Value: "8545"},
		{Name: "ANVIL_BLOCK_TIME", Value: "1"},
		{Name: "ANVIL_STATE_PATH", Value: "/data/anvil/anvil_state.json"},
	}

	containerProps := &domain.Container{
		Name:            "blockchain",
		Image:           "ghcr.io/foundry-rs/foundry:latest",
		ImagePullPolicy: "Always",
		Command:         []string{"sh", "-c"},
		Args:            []string{commandArgs},
		Env:             envVars,
		Ports: []domain.ContainerPort{
			{
				Name:          "rpc",
				ContainerPort: servicePort,
				Protocol:      "TCP",
			},
		},
		Resources: &domain.Resources{
			Limits:   map[string]string{"cpu": "1500m", "memory": "2048Mi"},
			Requests: map[string]string{"cpu": "1000m", "memory": "512Mi"},
		},
		UserID:  1000,
		GroupID: 1000,
	}

	appInstanceName := fmt.Sprintf("anvil-%s", anvilProps.ChainID)
	k8sResourceName := appInstanceName
	appName := "anvil"

	var workloadComposite workloadv1.WorkloadResource
	if anvilProps.persistence.enabled {
		stsProps := &statefulsetv1.Props{
			Namespace:   anvilProps.Namespace,
			Name:        k8sResourceName,
			AppName:     appName,
			AppInstance: appInstanceName,
			Containers: []*domain.Container{
				containerProps,
			},
		}
		component, err := statefulsetv1.Component(stsProps)(ctx)
		if err != nil {
			return nil, dry.Wrapf(err, "error creating anvil statefulset component")
		}
		stsComposite := dry.MustAs[*statefulsetv1.StatefulSetComposite](component) // Fixing incorrect type cast
		workloadComposite = stsComposite
	} else {
		deploymentProps := &deploymentv1.Props{
			Namespace:   anvilProps.Namespace,
			Name:        k8sResourceName,
			AppName:     appName,
			AppInstance: appInstanceName,
			Containers: []*domain.Container{
				containerProps,
			},
		}
		component, err := deploymentv1.Component(deploymentProps)(ctx)
		if err != nil {
			return nil, dry.Wrapf(err, "error creating anvil deployment")
		}
		deploymentComposite := dry.MustAs[*deploymentv1.DeploymentComposite](component)
		workloadComposite = deploymentComposite
	}

	ports := []*k8s.ServicePort{
		{
			Port:       jsii.Number(8545),
			Protocol:   dry.ToPtr("TCP"),
			TargetPort: k8s.IntOrString_FromNumber(jsii.Number(8545)),
		},
	}

	// Expose the Anvil instance via Ingress or Service based on the ingressProps
	if err := anvilProps.exposeService(ctx, workloadComposite, ports); err != nil {
		return nil, err
	}

	return Result{
		Component:       chart,
		appInstanceName: appInstanceName,
		namespace:       anvilProps.Namespace,
	}, nil
}

func (p *Props) exposeService(ctx context.Context, w workloadv1.WorkloadResource, ports []*k8s.ServicePort) error {
	if p.ingress.enabled {
		ingressProps := &workloadv1.ExposeViaIngressProps{
			// todo: set ingress class name based on cluster context
			IngressClassName: "example-ingress",
			Ports:            ports,
		}
		_, err := workloadv1.ExposeViaIngress(ctx, w, "/", ingressProps)
		return dry.Wrapf(err, "error exposing anvil ingress")
	}

	serviceProps := &workloadv1.ExposeViaServiceProps{
		ServiceType: "ClusterIP",
		Ports:       ports,
	}
	_, err := workloadv1.ExposeViaService(ctx, w, serviceProps)
	return dry.Wrapf(err, "error exposing anvil service")
}
