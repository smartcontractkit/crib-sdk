package workloadv1

import (
	"context"
	"errors"
	"fmt"

	"github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2/k8s"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"

	clientsideapply "github.com/smartcontractkit/crib-sdk/crib/scalar/clientsideapply/v1"
	ingressv1 "github.com/smartcontractkit/crib-sdk/crib/scalar/k8s/ingress/v1"
	servicev1 "github.com/smartcontractkit/crib-sdk/crib/scalar/k8s/service/v1"
)

// WorkloadResource defines the common interface for workload resources.
type WorkloadResource interface {
	// Component provides the crib.Component interface
	crib.Component
	// GetNamespace returns the namespace of the workload resource
	GetNamespace() string
	// GetAppInstance returns the application instance identifier
	GetAppInstance() string
	// GetName returns the name of the workload resource
	GetName() string
	// GetComponent returns the underlying component of the workload
	GetComponent() crib.Component
	// GetContainers returns the containers
	GetContainers() []*domain.Container
	// GetLabelsSelector returns a Labels Selector
	GetLabelsSelector() map[string]*string
	// GetResourceType returns the type of kubernetes workload resource
	GetResourceType() string
	// GetAppName returns the AppName
	GetAppName() string
}

// ExposeViaServiceProps defines the configuration for exposing a DeploymentComposite via a Service resource. type.
type ExposeViaServiceProps struct {
	Name string
	// ServiceType specifies the type of service to create (ClusterIP, NodePort, LoadBalancer, or ExternalName).
	ServiceType string
	Ports       []*k8s.ServicePort
}

// ExposeViaIngressProps defines the configuration for exposing a workload via an Ingress resource.
type ExposeViaIngressProps struct {
	// Name is the name for the Ingress resource. If empty, a name will be generated.
	Name string
	// IngressClassName specifies the ingress controller to use.
	IngressClassName string `validate:"required"`
	// Ports defines the service ports to expose through the Ingress.
	Ports []*k8s.ServicePort `validate:"dive,required, min=1"`
}

// Validate ensures that the Props are valid.
func (p *ExposeViaServiceProps) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(p)
}

// WaitForRollout sets up a dependency that ensures the system waits for a workload
// to fully roll out before proceeding.
func WaitForRollout(ctx context.Context, resource WorkloadResource) error {
	resourceType := resource.GetResourceType()
	waitFor, err := clientsideapply.New(ctx, &clientsideapply.Props{
		Namespace: resource.GetNamespace(),
		OnFailure: "abort",
		Action:    "kubectl",
		Args: []string{
			"rollout",
			"status",
			resourceType,
			"-l",
			fmt.Sprintf("app.kubernetes.io/instance=%s", resource.GetAppInstance()),
			"--timeout=600s",
		},
	})
	if err != nil {
		return dry.Wrapf(err, "failed to wait for rollout")
	}

	waitFor.Node().AddDependency(resource.GetComponent())
	return nil
}

// ExposeViaService wraps the kplus.DeploymentComposite.ExposeViaService method.
// It allows exposing the DeploymentComposite via a Service resource using the specified options.
func ExposeViaService(ctx context.Context, w WorkloadResource, options *ExposeViaServiceProps) (*servicev1.Service, error) {
	if options == nil {
		return nil, errors.New("options are required")
	}
	err := options.Validate(ctx)
	if err != nil {
		return nil, dry.Wrapf(err, "failed to validate options")
	}

	// Get service name or use auto-generated one
	name := options.Name
	if name == "" {
		name = w.GetName()
	}

	// Prepare service ports
	ports := make([]*k8s.ServicePort, 0)
	if len(options.Ports) > 0 {
		ports = options.Ports
	} else {
		// Extract ports from kubeDeployment containers if not specified
		containerPorts := make([]*k8s.ServicePort, 0)
		for _, container := range w.GetContainers() {
			for _, port := range container.Ports {
				servicePort := &k8s.ServicePort{
					Port:     dry.ToFloat64Ptr(port.ContainerPort),
					Protocol: dry.ToPtr(port.Protocol),
				}
				containerPorts = append(containerPorts, servicePort)
			}
		}
		if len(containerPorts) > 0 {
			ports = containerPorts
		}
	}

	serviceProps := &servicev1.Props{
		Namespace:   w.GetNamespace(),
		AppName:     w.GetAppName(),
		AppInstance: w.GetAppInstance(),
		Name:        name,
		Selector:    w.GetLabelsSelector(),
		ServiceType: options.ServiceType,
		Ports:       ports,
	}

	svcScalar, err := servicev1.New(ctx, serviceProps)
	if err != nil {
		return nil, dry.Wrapf(err, "failed to expose service")
	}

	return svcScalar, nil
}

// ExposeViaIngress allows exposing a WorkloadResource via an Ingress resource using the specified path and options.
func ExposeViaIngress(ctx context.Context, w WorkloadResource, path string, options *ExposeViaIngressProps) (*ingressv1.Ingress, error) {
	if options == nil {
		return nil, errors.New("options are required")
	}

	// Expose service first
	exposeSvcOptions := &ExposeViaServiceProps{
		Name:        "",
		Ports:       options.Ports,
		ServiceType: "ClusterIP",
	}
	serviceScalar, err := ExposeViaService(ctx, w, exposeSvcOptions)
	if err != nil {
		return nil, dry.Wrapf(err, "failed to expose service")
	}

	// Get ingress name or use auto-generated one
	name := options.Name
	if name == "" {
		name = w.GetName() + "-ingress"
	}

	// Create ingress rules based on service ports
	var rules []ingressv1.IngressRule
	servicePorts := serviceScalar.Ports()
	if len(servicePorts) > 0 {
		for _, port := range servicePorts {
			rule := ingressv1.IngressRule{
				Path:     path,
				PathType: "Prefix", // Add default path type
				Host:     "*",      // Add default host
				Service: ingressv1.IngressBackendService{
					Name: serviceScalar.Name(),
					Port: int(*port.Port),
				},
			}
			rules = append(rules, rule)
		}
	}

	ingressProps := &ingressv1.Props{
		Namespace:        w.GetNamespace(),
		AppName:          w.GetAppName(),
		AppInstance:      w.GetAppInstance(),
		Name:             name,
		IngressClassName: options.IngressClassName,
		Rules:            rules,
	}
	ingressScalar, err := ingressv1.New(ctx, ingressProps)
	if err != nil {
		return nil, dry.Wrapf(err, "failed to create ingress")
	}

	return dry.MustAs[*ingressv1.Ingress](ingressScalar), nil
}

// ConvertVolumeClaimTemplates converts PVC templates to k8s format.
func ConvertVolumeClaimTemplates(w WorkloadResource, pvcs []*domain.PersistentVolumeClaim) (*[]*k8s.KubePersistentVolumeClaimProps, error) {
	if len(pvcs) == 0 {
		return nil, nil
	}

	k8sPvcs := make([]*k8s.KubePersistentVolumeClaimProps, len(pvcs))
	for i, pvc := range pvcs {
		defaultResourceMetadataProps := &domain.DefaultResourceMetadataProps{
			Namespace:    w.GetNamespace(),
			AppName:      w.GetAppName(),
			AppInstance:  w.GetAppInstance(),
			ResourceName: fmt.Sprintf("%s-%s", w.GetAppInstance(), pvc.NameSuffix),
		}
		factory, err := domain.NewMetadataFactory(defaultResourceMetadataProps)
		if err != nil {
			return nil, dry.Wrapf(err, "failed to create default metadata")
		}

		// Use configurable storage class or default to "gp3"
		storageClass := pvc.StorageClass
		if storageClass == "" {
			storageClass = "gp3"
		}

		k8sPvcs[i] = &k8s.KubePersistentVolumeClaimProps{
			Metadata: factory.K8sResourceMetadata(),
			Spec: &k8s.PersistentVolumeClaimSpec{
				AccessModes: &[]*string{dry.ToPtr("ReadWriteOnce")},
				Resources: &k8s.VolumeResourceRequirements{
					Requests: &map[string]k8s.Quantity{
						"storage": k8s.Quantity_FromString(&pvc.Capacity),
					},
				},
				StorageClassName: dry.ToPtr(storageClass),
			},
		}
	}

	return dry.ToPtr(k8sPvcs), nil
}
