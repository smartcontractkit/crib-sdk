package ingressv1

import (
	"context"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2/k8s"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
)

type IngressRule struct {
	Host     string                `validate:"required"`
	PathType string                `validate:"required"`
	Path     string                `validate:"required"`
	Service  IngressBackendService `validate:"required"`
}

type IngressBackendService struct {
	Name string `validate:"required"`
	Port int    `validate:"required"`
}

type Props struct {
	Namespace        string             `validate:"required"`
	AppName          string             `validate:"required"`
	AppInstance      string             `validate:"required"`
	Name             string             `validate:"required"`
	Annotations      map[string]*string `validate:"omitempty"`
	IngressClassName string             `validate:"omitempty"`
	Rules            []IngressRule      `validate:"required,gt=0,dive,required"`
}

type Ingress struct {
	crib.Component
}

// Validate ensures that the Props struct satisfies the crib.Props interface.
func (p *Props) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(p)
}

// New creates a new kubernetes ingress component. The resulting [crib.Component] represents a full intent to
// install a single ingress resource.
func New(ctx context.Context, props crib.Props) (*Ingress, error) {
	if err := props.Validate(ctx); err != nil {
		return nil, err
	}

	scalarProps := dry.MustAs[*Props](props)
	parent := internal.ConstructFromContext(ctx)
	kplusChart := cdk8s.NewChart(parent, crib.ResourceID("sdk.IngressV1", props), nil)

	resourceID := crib.ResourceID(domain.CDK8sResource, props)

	// Create ingress rules from props
	ingressRules := make([]*k8s.IngressRule, 0, len(scalarProps.Rules))
	for _, rule := range scalarProps.Rules {
		httpPaths := []*k8s.HttpIngressPath{
			{
				Path:     dry.ToPtr(rule.Path),
				PathType: dry.ToPtr(rule.PathType),
				Backend: &k8s.IngressBackend{
					Service: &k8s.IngressServiceBackend{
						Name: dry.ToPtr(rule.Service.Name),
						Port: &k8s.ServiceBackendPort{
							Number: dry.ToPtr(float64(rule.Service.Port)),
						},
					},
				},
			},
		}

		ingressRules = append(ingressRules, &k8s.IngressRule{
			Host: dry.ToPtr(rule.Host),
			Http: &k8s.HttpIngressRuleValue{
				Paths: dry.ToPtr(httpPaths),
			},
		})
	}

	resourceMetadataProps := &domain.DefaultResourceMetadataProps{
		Namespace:    scalarProps.Namespace,
		AppName:      scalarProps.AppName,
		AppInstance:  scalarProps.AppInstance,
		ResourceName: scalarProps.Name,
	}
	metadataFactory, err := domain.NewMetadataFactory(resourceMetadataProps)
	if err != nil {
		return nil, dry.Wrapf(err, "failed to create default metadata for ingress")
	}
	metadata := metadataFactory.K8sResourceMetadata()

	ingressProps := &k8s.KubeIngressProps{
		Metadata: metadata,
		Spec: &k8s.IngressSpec{
			Rules: dry.ToPtr(ingressRules),
		},
	}

	// Set ingress class name if provided
	if scalarProps.IngressClassName != "" {
		ingressProps.Spec.IngressClassName = dry.ToPtr(scalarProps.IngressClassName)
	}

	ingress := k8s.NewKubeIngress(kplusChart, resourceID, ingressProps)

	return &Ingress{
		Component: ingress,
	}, nil
}
