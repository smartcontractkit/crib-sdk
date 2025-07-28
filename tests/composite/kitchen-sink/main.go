// kitchen-sink is a holistic example of how to leverage the Composite API to share state among Scalar Components.
// It includes a variety of components that demonstrate the capabilities of the Composite API.
package main

import (
	"context"

	"github.com/samber/lo"
	"github.com/smartcontractkit/crib-sdk/crib"
	configmap "github.com/smartcontractkit/crib-sdk/crib/scalar/k8s/configmap/v1"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
)

type (
	// ConfigMapper is an intermediate Scalar Component. It's "local" to the Composite API,
	// but is used to stitch together results from various Scalar Components.
	ConfigMapper struct{}

	// HostnamePrinter is a simple example interface that is used to print the Hostname.
	// If a Scalar component implements this interface, it will automatically be available
	// to the Composite context as a slice of the interface.
	HostnamePrinter interface {
		Host() string
		Port() string
	}
)

// Composite shows how to utilize the Composite API to share state among Scalar Components.
// It includes a variety of components that demonstrate the capabilities of the Composite API.
var composite = crib.NewComposite(
	// Producers
	NewDockerRegistry("5000"), // Deploy a docker registry
	NewDockerRegistry("5001"), // Deploy a docker registry
	NewDockerRegistry("5002"), // Deploy a docker registry

	NewKindCluster, // Deploy a local kind cluster

	// Consumers
	NewConfigMapper, // Deploy a config map
)

func main() {
	ctx := context.Background()

	// Create a new plan with the composite.
	plan := crib.NewPlan("kitchen-sink",
		crib.ComponentSet(composite),
	)

	// Apply the plan.
	plan.Apply(ctx)
}

// NewConfigMapper is a helper Component. It consumes the available HostnamePrinters within the
// Composite container and creates a config map with the data.
func NewConfigMapper() *ConfigMapper {
	return &ConfigMapper{}
}

func (h *ConfigMapper) Apply(ctx context.Context, entries []HostnamePrinter) (crib.Component, error) {
	component := configmap.Component(&configmap.Props{
		// Map the entries to a map of strings.
		Data: dry.PtrMapping(lo.Associate(entries, func(e HostnamePrinter) (string, string) {
			return e.Host(), e.Port()
		})),
		// Miscellaneous required fields.
		Namespace:   domain.DefaultNamespace,
		AppName:     "kitchen-sink",
		AppInstance: "config-mapper",
		Name:        "config-mapper",
	})
	return component(ctx)
}

var (
	_ HostnamePrinter = (*DockerResults)(nil)
	_ HostnamePrinter = (*KindResults)(nil)
)
