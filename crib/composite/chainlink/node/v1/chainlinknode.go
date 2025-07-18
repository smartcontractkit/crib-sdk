package chainlinknodev1

import (
	"context"
	"fmt"
	"slices"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2/k8s"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"

	postgresv1 "github.com/smartcontractkit/crib-sdk/crib/scalar/charts/postgres/v1"
	helmchartv1 "github.com/smartcontractkit/crib-sdk/crib/scalar/helmchart/v1"
	configmapv1 "github.com/smartcontractkit/crib-sdk/crib/scalar/k8s/configmap/v1"
	secretv1 "github.com/smartcontractkit/crib-sdk/crib/scalar/k8s/secret/v1"
	servicev1 "github.com/smartcontractkit/crib-sdk/crib/scalar/k8s/service/v1"
)

const (
	ComponentName   = "sdk.composite.chainlink.node.v1"
	defaultLogin    = "admin@chain.link"
	defaultPassword = "staticlongpassword"
	defaultAPIPort  = 6688
	defaultP2PPort  = 5001
)

// ContainerPort represents a port configuration for the Chainlink container.
type ContainerPort struct {
	Name          string `validate:"required,lte=63,dns_rfc1035_label"`
	Protocol      string `default:"TCP"                                validate:"oneof=TCP UDP"`
	ContainerPort int    `validate:"required,min=1,max=65535"`
}

// PostgresInfo contains information about the automatically created postgres component.
type PostgresInfo struct {
	ReleaseName       string
	DatabaseURL       string
	Username          string
	Password          string
	Database          string
	SuperUser         string
	SuperUserPassword string
}

// Props contains properties specific to the Chainlink node.
type Props struct {
	Namespace       string `omitempty,validate:"required"`
	AppInstanceName string `validate:"required"`
	Image           string `validate:"required"`
	ImagePullPolicy string `default:"IfNotPresent"        validate:"oneof=Always IfNotPresent Never"`
	Command         []string
	Args            []string
	EnvVars         map[string]string
	Config          string `validate:"required"`
	// ConfigOverrides contains additional configuration files to be passed as -c arguments.
	// The files are processed in lexicographic order by filename to ensure consistent
	// behavior when Chainlink performs configuration merging.
	ConfigOverrides map[string]string
	// SecretsOverrides contains additional secrets files to be passed as -s arguments.
	// The files are processed in lexicographic order by filename to ensure consistent
	// behavior when Chainlink performs configuration merging.
	SecretsOverrides map[string]string
	Resources        ResourceRequirements
	// DatabaseURL for connecting to existing database. If not provided, a postgres component will be created automatically.
	DatabaseURL string
	Ports       []ContainerPort // Container ports to expose, defaults to API and P2P ports
	Replicas    int32           `default:"1"`
}

// ResourceRequirements represents CPU and memory resource requirements.
type ResourceRequirements struct {
	Limits   map[string]string `default:"{\"cpu\": \"1\", \"memory\": \"2048Mi\"}"`
	Requests map[string]string `default:"{\"cpu\": \"0.5\", \"memory\": \"128Mi\"}"`
}
type APICredentials struct {
	UserName string
	Password string
}

type Result struct {
	crib.Component
	Postgres       *PostgresInfo
	APICredentials *APICredentials
	nodeName       string
	namespace      string
	apiPort        int
	P2PPort        int
}

func (r *Result) APIUrl() string {
	return domain.ClusterLocalServiceURL("http", r.nodeName, r.namespace, r.apiPort)
}

func (r *Result) HostName() string {
	return domain.ClusterLocalServiceURL("", r.nodeName, r.namespace, 0)
}

// Validate validates the props.
func (p *Props) Validate(ctx context.Context) error {
	v := internal.ValidatorFromContext(ctx)

	// Process defaults on nested structs if they exist
	if err := v.Struct(&p.Resources); err != nil {
		return err
	}

	return v.Struct(p)
}

// convertToK8sPorts converts our ContainerPort slice to k8s ContainerPort slice.
func convertToK8sPorts(ports []ContainerPort) *[]*k8s.ContainerPort {
	k8sPorts := make([]*k8s.ContainerPort, len(ports))
	for i, port := range ports {
		k8sPorts[i] = &k8s.ContainerPort{
			Name:          dry.ToPtr(port.Name),
			ContainerPort: dry.ToPtr(float64(port.ContainerPort)),
			Protocol:      dry.ToPtr(port.Protocol),
		}
	}
	return &k8sPorts
}

// Component returns a new ChainlinkNode composite component.
func Component(props *Props) crib.ComponentFunc {
	return func(ctx context.Context) (crib.Component, error) {
		if err := props.Validate(ctx); err != nil {
			return nil, err
		}
		return chainlinkNode(ctx, props)
	}
}

// chainlinkNode creates and returns a new ChainlinkNode composite component.
func chainlinkNode(ctx context.Context, props crib.Props) (crib.Component, error) {
	parent := internal.ConstructFromContext(ctx)
	chart := cdk8s.NewChart(parent, crib.ResourceID(ComponentName, props), nil)
	ctx = internal.ContextWithConstruct(ctx, chart)

	chainlinkProps := dry.MustAs[*Props](props)

	if len(chainlinkProps.Ports) == 0 {
		chainlinkProps.Ports = []ContainerPort{
			{
				Name:          "api",
				ContainerPort: defaultAPIPort,
				Protocol:      "TCP",
			},
			{
				Name:          "p2pv2",
				ContainerPort: defaultP2PPort,
				Protocol:      "TCP",
			},
		}
	}

	var postgresInfo *PostgresInfo
	var actualDatabaseURL string

	// If DatabaseURL is not provided, create a postgres component automatically
	if chainlinkProps.DatabaseURL == "" {
		postgresReleaseName := fmt.Sprintf("%s-postgres", chainlinkProps.AppInstanceName)
		username := chainlinkProps.AppInstanceName
		database := chainlinkProps.AppInstanceName

		// Create PostgreSQL component
		_, err := postgresv1.Component(&helmchartv1.ChartProps{
			Namespace:   chainlinkProps.Namespace,
			ReleaseName: postgresReleaseName,
			Values: map[string]any{
				"fullnameOverride": postgresReleaseName,
				"auth": map[string]any{
					"enablePostgresUser": true,
					"postgresPassword":   defaultPassword,
					"username":           username,
					"password":           defaultPassword,
					"database":           database,
				},
				"primary": map[string]any{
					"persistence": map[string]any{
						"enabled": false,
					},
				},
			},
		})(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create PostgreSQL component: %w", err)
		}

		// Generate database URL
		actualDatabaseURL = fmt.Sprintf("postgresql://%s:%s@%s:5432/%s?sslmode=disable",
			username, defaultPassword, postgresReleaseName, database)

		postgresInfo = &PostgresInfo{
			ReleaseName:       postgresReleaseName,
			DatabaseURL:       actualDatabaseURL,
			Username:          username,
			Password:          defaultPassword,
			Database:          database,
			SuperUser:         "postgres",
			SuperUserPassword: defaultPassword,
		}
	} else {
		actualDatabaseURL = chainlinkProps.DatabaseURL
	}

	// Build command and args dynamically based on config files
	command := []string{"chainlink"}
	args := []string{}

	// Always include the main config file
	args = append(args, "-c", "/chainlink/config/config.toml")

	// Add config overrides in sorted order for consistency
	overrideFiles := make([]string, 0, len(chainlinkProps.ConfigOverrides))
	for filename := range chainlinkProps.ConfigOverrides {
		overrideFiles = append(overrideFiles, filename)
	}
	// Sort to ensure consistent ordering
	slices.Sort(overrideFiles)
	for _, filename := range overrideFiles {
		args = append(args, "-c", "/chainlink/config/"+filename)
	}

	// Add secrets file
	args = append(args, "-s", "/chainlink/secrets/secrets.toml")

	// Add secrets overrides in sorted order for consistency
	secretsOverrideFiles := make([]string, 0, len(chainlinkProps.SecretsOverrides))
	for filename := range chainlinkProps.SecretsOverrides {
		secretsOverrideFiles = append(secretsOverrideFiles, filename)
	}
	// Sort to ensure consistent ordering
	slices.Sort(secretsOverrideFiles)
	for _, filename := range secretsOverrideFiles {
		args = append(args, "-s", "/chainlink/secrets/"+filename)
	}
	// Add the api credentials file and the actual command
	args = append(args, "node", "start", "-a", "/chainlink/secrets/apicredentials")

	// Use provided command/args if specified, otherwise use the generated ones
	if len(chainlinkProps.Command) > 0 {
		command = chainlinkProps.Command
	}
	if len(chainlinkProps.Args) > 0 {
		args = chainlinkProps.Args
	}

	// Create secret with database URL
	secretsTOMLContent := dry.RemoveIndentation(fmt.Sprintf(`
		[Password]
		Keystore = "keystorepassword"
		VRF = "vrfpassword"

		[Database]
		URL = "%s"
	`, actualDatabaseURL))

	// Build secret data with main secrets and overrides

	secretData := map[string]*string{
		"apicredentials": dry.ToPtr(fmt.Sprintf("%s\n%s", defaultLogin, defaultPassword)),
		"secrets.toml":   dry.ToPtr(secretsTOMLContent),
	}

	// Add secrets overrides as additional files
	for filename, content := range chainlinkProps.SecretsOverrides {
		secretData[filename] = dry.ToPtr(content)
	}

	_, err := secretv1.New(ctx, &secretv1.Props{
		Name:       fmt.Sprintf("%s-file", chainlinkProps.AppInstanceName),
		Namespace:  chainlinkProps.Namespace,
		StringData: secretData,
	})
	if err != nil {
		return nil, err
	}

	// Build configmap data with main config and overrides
	configMapData := map[string]*string{
		"config.toml": dry.ToPtr(chainlinkProps.Config),
	}

	// Add config overrides as additional files
	for filename, content := range chainlinkProps.ConfigOverrides {
		configMapData[filename] = dry.ToPtr(content)
	}

	_, err = configmapv1.New(ctx, &configmapv1.Props{
		Namespace:   chainlinkProps.Namespace,
		Name:        fmt.Sprintf("%s-config", chainlinkProps.AppInstanceName),
		AppName:     "chainlink",
		AppInstance: chainlinkProps.AppInstanceName,
		Data:        &configMapData,
	})
	if err != nil {
		return nil, err
	}
	// Create deployment using vanilla cdk8s types
	labels := map[string]*string{
		"app.kubernetes.io/name":     dry.ToPtr("chainlink"),
		"app.kubernetes.io/instance": dry.ToPtr(chainlinkProps.AppInstanceName),
	}

	// Convert environment variables with consistent ordering
	envVars := make([]*k8s.EnvVar, 0, len(chainlinkProps.EnvVars))
	envKeys := make([]string, 0, len(chainlinkProps.EnvVars))
	for key := range chainlinkProps.EnvVars {
		envKeys = append(envKeys, key)
	}
	slices.Sort(envKeys) // Sort keys for consistent ordering

	for _, key := range envKeys { // Use sorted keys
		envVars = append(envVars, &k8s.EnvVar{
			Name:  dry.ToPtr(key),
			Value: dry.ToPtr(chainlinkProps.EnvVars[key]),
		})
	}

	// Convert resources
	var resources *k8s.ResourceRequirements
	limits := make(map[string]k8s.Quantity)
	requests := make(map[string]k8s.Quantity)

	for k, v := range chainlinkProps.Resources.Limits {
		limits[k] = k8s.Quantity_FromString(&v)
	}
	for k, v := range chainlinkProps.Resources.Requests {
		requests[k] = k8s.Quantity_FromString(&v)
	}

	resources = &k8s.ResourceRequirements{
		Limits:   &limits,
		Requests: &requests,
	}

	// Create the deployment
	_ = k8s.NewKubeDeployment(chart, dry.ToPtr("chainlink-deployment"), &k8s.KubeDeploymentProps{
		Metadata: &k8s.ObjectMeta{
			Name:      dry.ToPtr(chainlinkProps.AppInstanceName),
			Namespace: dry.ToPtr(chainlinkProps.Namespace),
			Labels:    &labels,
		},
		Spec: &k8s.DeploymentSpec{
			Replicas: dry.ToPtr(float64(chainlinkProps.Replicas)),
			Selector: &k8s.LabelSelector{
				MatchLabels: &labels,
			},
			Template: &k8s.PodTemplateSpec{
				Metadata: &k8s.ObjectMeta{
					Labels: &labels,
				},
				Spec: &k8s.PodSpec{
					InitContainers: &[]*k8s.Container{
						{
							Name: dry.ToPtr("wait-for-db"),
							// Uses the same image and approach as chainlink-cluster chart
							Image:           dry.ToPtr("docker.io/bitnami/postgresql@sha256:6bea1699d088605204841b889fb79d7572030a36ec5731e736d73cd33018cc03"),
							ImagePullPolicy: dry.ToPtr("IfNotPresent"),
							Command:         dry.PtrSlice([]string{"/bin/bash", "-c"}),
							Args: dry.PtrSlice([]string{dry.RemoveIndentation(`
								set -e
								echo "Waiting for database to be ready..."
								
								# Extract database URL from secrets.toml
								DB_URL=$(grep -A 10 '\[Database\]' /chainlink/secrets/secrets.toml | grep 'URL' | sed 's/.*URL = "\([^"]*\)".*/\1/')
								
								if [ -z "$DB_URL" ]; then
									echo "Error: Could not extract database URL from secrets.toml"
									exit 1
								fi
								
								echo "Database URL extracted: $DB_URL"
								
								# Wait for database to be ready
								until pg_isready -d "$DB_URL"; do
									echo "Database is not ready. Waiting 2 seconds..."
									sleep 2
								done
								
								echo "Database is ready!"
								`)}),
							SecurityContext: &k8s.SecurityContext{
								RunAsUser:    dry.ToPtr(float64(1000)),
								RunAsGroup:   dry.ToPtr(float64(1000)),
								RunAsNonRoot: dry.ToPtr(true),
							},
							VolumeMounts: &[]*k8s.VolumeMount{
								{
									Name:      dry.ToPtr("secrets"),
									MountPath: dry.ToPtr("/chainlink/secrets"),
									ReadOnly:  dry.ToPtr(true),
								},
							},
						},
					},
					Containers: &[]*k8s.Container{
						{
							Name:            dry.ToPtr("chainlink"),
							Image:           dry.ToPtr(chainlinkProps.Image),
							ImagePullPolicy: dry.ToPtr(chainlinkProps.ImagePullPolicy),
							Command:         dry.PtrSlice(command),
							Args:            dry.PtrSlice(args),
							Env:             &envVars,
							Ports:           convertToK8sPorts(chainlinkProps.Ports),
							Resources:       resources,
							SecurityContext: &k8s.SecurityContext{
								RunAsUser:    dry.ToPtr(float64(1000)),
								RunAsGroup:   dry.ToPtr(float64(1000)),
								RunAsNonRoot: dry.ToPtr(true),
							},
							VolumeMounts: &[]*k8s.VolumeMount{
								{
									Name:      dry.ToPtr("config"),
									MountPath: dry.ToPtr("/chainlink/config"),
									ReadOnly:  dry.ToPtr(true),
								},
								{
									Name:      dry.ToPtr("secrets"),
									MountPath: dry.ToPtr("/chainlink/secrets"),
									ReadOnly:  dry.ToPtr(true),
								},
							},
						},
					},
					Volumes: &[]*k8s.Volume{
						{
							Name: dry.ToPtr("config"),
							ConfigMap: &k8s.ConfigMapVolumeSource{
								Name: dry.ToPtr(fmt.Sprintf("%s-config", chainlinkProps.AppInstanceName)),
							},
						},
						{
							Name: dry.ToPtr("secrets"),
							Secret: &k8s.SecretVolumeSource{
								SecretName: dry.ToPtr(fmt.Sprintf("%s-file", chainlinkProps.AppInstanceName)),
							},
						},
					},
				},
			},
		},
	})

	// Create service to expose the container ports
	servicePorts := make([]*k8s.ServicePort, len(chainlinkProps.Ports))
	for i, port := range chainlinkProps.Ports {
		servicePorts[i] = &k8s.ServicePort{
			Name:       dry.ToPtr(port.Name),
			Port:       dry.ToPtr(float64(port.ContainerPort)),
			TargetPort: k8s.IntOrString_FromNumber(dry.ToPtr(float64(port.ContainerPort))),
			Protocol:   dry.ToPtr(port.Protocol),
		}
	}

	// Create selector map with proper pointer format
	selector := map[string]*string{
		"app.kubernetes.io/name":     dry.ToPtr("chainlink"),
		"app.kubernetes.io/instance": dry.ToPtr(chainlinkProps.AppInstanceName),
	}

	_, err = servicev1.New(ctx, &servicev1.Props{
		Name:        chainlinkProps.AppInstanceName,
		Namespace:   chainlinkProps.Namespace,
		AppName:     "chainlink",
		AppInstance: chainlinkProps.AppInstanceName,
		ServiceType: "ClusterIP",
		Ports:       servicePorts,
		Selector:    selector,
	})
	if err != nil {
		return nil, err
	}

	// Set ports in results
	apiPort := defaultAPIPort
	p2pPort := defaultP2PPort
	for _, port := range chainlinkProps.Ports {
		if port.Name == "api" {
			apiPort = port.ContainerPort
		}
		if port.Name == "p2pv2" {
			p2pPort = port.ContainerPort
		}
	}

	return Result{
		Component: chart,
		nodeName:  chainlinkProps.AppInstanceName,
		namespace: chainlinkProps.Namespace,
		apiPort:   apiPort,
		APICredentials: &APICredentials{
			UserName: defaultLogin,
			Password: defaultPassword,
		},
		P2PPort:  p2pPort,
		Postgres: postgresInfo,
	}, nil
}
