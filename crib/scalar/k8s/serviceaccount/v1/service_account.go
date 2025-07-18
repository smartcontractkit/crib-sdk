package v1

import (
	"context"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2/k8s"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
)

type Props struct {
	AutomountServiceAccountToken *bool    `validate:"omitempty"`
	Namespace                    string   `validate:"required"`
	AppName                      string   `validate:"required"`
	AppInstance                  string   `validate:"required"`
	Name                         string   `validate:"required"`
	Secrets                      []string `validate:"omitempty"`
	ImagePullSecrets             []string `validate:"omitempty"`
}

type ServiceAccount struct {
	crib.Component
	props Props
}

func (sa *ServiceAccount) Name() string {
	return sa.props.Name
}

func (p *Props) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(p)
}

// New creates a new Kubernetes ServiceAccount scalar component.
func New(ctx context.Context, props crib.Props) (*ServiceAccount, error) {
	if err := props.Validate(ctx); err != nil {
		return nil, err
	}

	scalarProps := dry.MustAs[*Props](props)
	parent := internal.ConstructFromContext(ctx)
	chart := cdk8s.NewChart(parent, crib.ResourceID("sdk.scalar.ServiceAccountV1", props), nil)

	resourceMetadataProps := &domain.DefaultResourceMetadataProps{
		Namespace:    scalarProps.Namespace,
		AppName:      scalarProps.AppName,
		AppInstance:  scalarProps.AppInstance,
		ResourceName: scalarProps.Name,
	}
	metadataFactory, err := domain.NewMetadataFactory(resourceMetadataProps)
	if err != nil {
		return nil, dry.Wrapf(err, "failed to create default metadata for role")
	}
	metadata := metadataFactory.K8sResourceMetadata()

	// Create Kubernetes ServiceAccount
	saSpec := &k8s.KubeServiceAccountProps{
		Metadata: metadata,
	}

	// Add secrets if provided
	if len(scalarProps.Secrets) > 0 {
		secrets := make([]*k8s.ObjectReference, len(scalarProps.Secrets))
		for i, secret := range scalarProps.Secrets {
			secrets[i] = &k8s.ObjectReference{
				Name: &secret,
			}
		}
		saSpec.Secrets = &secrets
	}

	// Add image pull secrets if provided
	if len(scalarProps.ImagePullSecrets) > 0 {
		imagePullSecrets := make([]*k8s.LocalObjectReference, len(scalarProps.ImagePullSecrets))
		for i, secret := range scalarProps.ImagePullSecrets {
			imagePullSecrets[i] = &k8s.LocalObjectReference{
				Name: &secret,
			}
		}
		saSpec.ImagePullSecrets = &imagePullSecrets
	}

	// Set automount service account token if provided
	if scalarProps.AutomountServiceAccountToken != nil {
		saSpec.AutomountServiceAccountToken = scalarProps.AutomountServiceAccountToken
	}

	serviceAccount := k8s.NewKubeServiceAccount(chart, crib.ResourceID("ServiceAccount", props), saSpec)

	return &ServiceAccount{
		Component: serviceAccount,
		props:     *scalarProps,
	}, nil
}

// GetServiceAccountName returns the name of the service account.
func (sa *ServiceAccount) GetServiceAccountName() string {
	return sa.props.Name
}

// GetNamespace returns the namespace of the service account.
func (sa *ServiceAccount) GetNamespace() string {
	return sa.props.Namespace
}
