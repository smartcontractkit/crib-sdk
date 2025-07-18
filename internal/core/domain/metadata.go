package domain

import (
	"github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2/k8s"
	"github.com/go-playground/validator/v10"

	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

type (
	Labels      map[string]string
	Annotations map[string]string
)

// DefaultResourceMetadataProps is used for generating standard K8s resource metadata
// you can pass ExtraLabels or ExtraAnnotations fields to append additional labels
// They will be merged with the defaults.
type DefaultResourceMetadataProps struct {
	ExtraLabels      Labels
	ExtraAnnotations Annotations
	Namespace        string `validate:"required"`
	ResourceName     string `validate:"required"`
	AppName          string `validate:"required"`
	AppInstance      string `validate:"required"`
}

// MetadataFactory provides a unified way for setting metadata fields in K8s resources.
type MetadataFactory struct {
	props *DefaultResourceMetadataProps
}

// NewMetadataFactory creates a new MetadataFactory with the provided properties.
// It validates that all required fields in props are set before creating the factory.
// Returns an error if validation fails.
func NewMetadataFactory(props *DefaultResourceMetadataProps) (*MetadataFactory, error) {
	validate := validator.New(validator.WithRequiredStructEnabled())
	err := validate.Struct(props)
	if err != nil {
		return nil, dry.Wrapf(err, "validation failed")
	}

	return &MetadataFactory{
		props: props,
	}, nil
}

// K8sResourceMetadata returns a default metadata for K8S Resource based on the supplied props.
func (f *MetadataFactory) K8sResourceMetadata() *k8s.ObjectMeta {
	defaultLabels := map[string]string{
		"app.kubernetes.io/instance": f.props.AppInstance,
		"app.kubernetes.io/name":     f.props.AppName,
	}

	// Merge defaultLabels with custom labels
	mergedLabels := make(map[string]string)
	for k, v := range defaultLabels {
		mergedLabels[k] = v
	}
	for k, v := range f.props.ExtraLabels {
		mergedLabels[k] = v
	}

	// In most of the cases we can default resourceName to be the same as App Instance name
	resourceName := f.props.ResourceName
	if resourceName == "" {
		resourceName = f.props.AppInstance
	}

	return &k8s.ObjectMeta{
		Name:        dry.ToPtr(resourceName),
		Namespace:   dry.ToPtr(f.props.Namespace),
		Labels:      dry.PtrMapping(mergedLabels),
		Annotations: dry.PtrMapping(f.props.ExtraAnnotations),
	}
}

// SelectorLabels returns default selector labels.
//
//nolint:gocritic // required for cdk8s compatibility
func (f *MetadataFactory) SelectorLabels() *map[string]*string {
	selectorLabels := map[string]string{
		"app.kubernetes.io/instance": f.props.AppInstance,
		"app.kubernetes.io/name":     f.props.AppName,
	}

	return dry.PtrMapping(selectorLabels)
}
