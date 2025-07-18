package jdv1

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/jsii-runtime-go"
	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2/k8s"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"

	deploymentv1 "github.com/smartcontractkit/crib-sdk/crib/composite/deployment/v1"
	workloadv1 "github.com/smartcontractkit/crib-sdk/crib/composite/workload/v1"
	postgresv1 "github.com/smartcontractkit/crib-sdk/crib/scalar/charts/postgres/v1"
	clientsideapply "github.com/smartcontractkit/crib-sdk/crib/scalar/clientsideapply/v1"
	helmchartv1 "github.com/smartcontractkit/crib-sdk/crib/scalar/helmchart/v1"
)

const (
	ComponentName        = "sdk.composite.chainlink.jd.v1"
	grpcServicePort      = 42242
	wsRPCServicePort     = 8080
	wsrpcHealthCheckPort = 8081
)

type (
	// DBProps contains properties for the database component.
	DBProps struct {
		Values        map[string]any
		ValuesPatches [][]string
	}
	// JDProps contains properties specific to the JD component.
	JDProps struct {
		Image            string `validate:"required,image_uri"`
		CSAEncryptionKey string `validate:"required,hexadecimal,len=64"`
	}

	// Props contains Composite component props.
	Props struct {
		JD              JDProps
		AppInstanceName string `default:"jd"`
		Namespace       string `validate:"required"`
		DB              DBProps
		WaitForRollout  bool
	}

	Result struct {
		crib.Component
		namespace       string
		appInstanceName string
	}
)

// Returns GRPCHostURL without protocol prefix.
func (r *Result) GRPCHostURL() string {
	return domain.ClusterLocalServiceURL("", r.appInstanceName, r.namespace, grpcServicePort)
}

// Returns WSRPCHostURL without protocol prefix.
func (r *Result) WSRPCHostURL() string {
	return domain.ClusterLocalServiceURL("", r.appInstanceName, r.namespace, wsRPCServicePort)
}

// Validate validates the props.
func (p *Props) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(p)
}

func (p *Props) dbInstanceName() string {
	return fmt.Sprintf("%s-db", p.AppInstanceName)
}

// Component returns a new JD composite component.
func Component(props *Props) crib.ComponentFunc {
	return func(ctx context.Context) (crib.Component, error) {
		return jdComposite(ctx, props)
	}
}

// jdComposite creates and returns a new JD composite component.
func jdComposite(ctx context.Context, props crib.Props) (crib.Component, error) {
	if err := props.Validate(ctx); err != nil {
		return nil, err
	}
	parent := internal.ConstructFromContext(ctx)
	chart := cdk8s.NewChart(parent, crib.ResourceID(ComponentName, props), nil)
	ctx = internal.ContextWithConstruct(ctx, chart)

	jdProps := dry.MustAs[*Props](props)

	_, _, err := AddPostgresWithWait(ctx, jdProps)
	if err != nil {
		return nil, dry.Wrapf(err, "failed to create postgres subcomponent")
	}

	_, err = jdApp(ctx, jdProps)
	if err != nil {
		return nil, dry.Wrapf(err, "failed to create jd app component")
	}

	return Result{
		Component:       chart,
		namespace:       jdProps.Namespace,
		appInstanceName: jdProps.AppInstanceName,
	}, nil
}

func jdApp(ctx context.Context, jdProps *Props) (*deploymentv1.DeploymentComposite, error) {
	envVars := []domain.EnvVar{
		{Name: "ENVIRONMENT", Value: "development"},
		{Name: "DATABASE_URL", Value: fmt.Sprintf("postgresql://chainlink:JGVgp7M2Emcg7Av8KKVUgMZb@%s:5432/chainlink?sslmode=disable", jdProps.dbInstanceName())},
		{Name: "CSA_KEY_ENCRYPTION_SECRET", Value: jdProps.JD.CSAEncryptionKey},
		{Name: "SERVER_ENABLE_REFLECTION", Value: "true"},
	}

	containerProps := &domain.Container{
		Name:            "app",
		Image:           jdProps.JD.Image,
		ImagePullPolicy: "IfNotPresent",
		Env:             envVars,
		Ports: []domain.ContainerPort{
			{
				Name:          "grpc",
				ContainerPort: grpcServicePort,
				Protocol:      "TCP",
			},
			{
				Name:          "wsrpc",
				ContainerPort: wsRPCServicePort,
				Protocol:      "TCP",
			},
			{
				Name:          "wsrpc-health",
				ContainerPort: wsrpcHealthCheckPort,
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

	appName := "jd"

	deploymentProps := &deploymentv1.Props{
		Namespace:   jdProps.Namespace,
		Name:        jdProps.AppInstanceName,
		AppName:     appName,
		AppInstance: jdProps.AppInstanceName,
		Containers: []*domain.Container{
			containerProps,
		},
	}
	component, err := deploymentv1.Component(deploymentProps)(ctx)
	if err != nil {
		return nil, dry.Wrapf(err, "error creating jd deployment")
	}
	deploymentComposite := dry.MustAs[*deploymentv1.DeploymentComposite](component)

	ports := []*k8s.ServicePort{
		{
			Name:       dry.ToPtr("grpc"),
			Port:       jsii.Number(grpcServicePort),
			Protocol:   dry.ToPtr("TCP"),
			TargetPort: k8s.IntOrString_FromNumber(jsii.Number(grpcServicePort)),
		},
		{
			Name:       dry.ToPtr("wsrpc"),
			Port:       jsii.Number(wsRPCServicePort),
			Protocol:   dry.ToPtr("TCP"),
			TargetPort: k8s.IntOrString_FromNumber(jsii.Number(wsRPCServicePort)),
		},
		{
			Name:       dry.ToPtr("wsrpc-health"),
			Port:       jsii.Number(wsrpcHealthCheckPort),
			Protocol:   dry.ToPtr("TCP"),
			TargetPort: k8s.IntOrString_FromNumber(jsii.Number(wsrpcHealthCheckPort)),
		},
	}

	exposeViaServiceProps := &workloadv1.ExposeViaServiceProps{
		ServiceType: "ClusterIP",
		Ports:       ports,
	}

	_, err = deploymentComposite.ExposeViaService(exposeViaServiceProps)
	if err != nil {
		return nil, dry.Wrapf(err, "error exposing jd service")
	}
	if jdProps.WaitForRollout {
		if err := deploymentComposite.WaitForRollout(ctx); err != nil {
			return nil, err
		}
	}

	return deploymentComposite, nil
}

// AddPostgresWithWait adds postgres with wait as dependency to the current ctx.
func AddPostgresWithWait(ctx context.Context, jdProps *Props) (pg, waitForDB crib.Component, err error) {
	pg, err = postgresv1.Component(&helmchartv1.ChartProps{
		Namespace:   jdProps.Namespace,
		ReleaseName: jdProps.dbInstanceName(),
		Values: map[string]any{
			"fullnameOverride": jdProps.dbInstanceName(),
		},
	})(ctx)
	if err != nil {
		return nil, nil, dry.Wrapf(err, "error creating jd postgres")
	}

	// Wait for PostgreSQL to be ready
	labels := []string{
		fmt.Sprintf("statefulset.kubernetes.io/pod-name=%s-0", jdProps.dbInstanceName()),
		"app.kubernetes.io/name=postgresql",
	}
	waitForDB, err = clientsideapply.New(ctx, &clientsideapply.Props{
		Namespace: jdProps.Namespace,
		OnFailure: "abort",
		Action:    "kubectl",
		Args: []string{
			"wait",
			"-n", jdProps.Namespace,
			"--for=condition=ready",
			"pod",
			fmt.Sprintf("-l=%s", strings.Join(labels, ",")),
			"--timeout=600s",
		},
	})
	if err != nil {
		return nil, nil, err
	}
	waitForDB.Node().AddDependency(pg)
	return pg, waitForDB, nil
}
