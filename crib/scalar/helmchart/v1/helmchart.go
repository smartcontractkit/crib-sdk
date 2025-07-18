// Package helmchart provides a crib-sdk scalar component for managing Helm charts. This scalar is a unique component
// in that other scalar components may include it as a dependency to ease the management of Helm charts.
package helmchart

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"
	"sync"

	"github.com/aws/jsii-runtime-go"
	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/samber/lo"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/adapter/helm"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/port"

	clientsideapply "github.com/smartcontractkit/crib-sdk/crib/scalar/clientsideapply/v1"
	namespace "github.com/smartcontractkit/crib-sdk/crib/scalar/k8s/namespace/v1"
)

const helmBinaryName = "helm"

var helmBinary = sync.OnceValues(func() (string, error) {
	prog, err := exec.LookPath(helmBinaryName)
	return dry.Wrapf2(prog, err, "failed to locate helm binary")
})

type ChartProps struct {
	Name          string `validate:"required,lte=63,dns_rfc1035_label"`
	Chart         string `validate:"required,lte=63,dns_rfc1035_label"`
	Namespace     string `validate:"omitempty,lte=63,dns_rfc1035_label"`
	ReleaseName   string `validate:"omitempty,lte=63"`
	Repo          string `validate:"omitempty,startswith=http|startswith=oci"`
	ValuesLoader  port.ValuesLoader
	Values        map[string]any `validate:"omitempty"`
	Version       string         `validate:"omitempty,lte=63,semver|eq=main"`
	ValuesPatches [][]string     `validate:"omitempty"`
	Flags         []string       `default:"[\"--skip-tests\"]"               validate:"omitempty"`
	WaitForReady  bool           // If true, the chart will wait for resources to be ready before returning.
}

func (c *ChartProps) Validate(ctx context.Context) error {
	v := internal.ValidatorFromContext(ctx)
	if v == nil {
		return errors.New("validator not found in context")
	}
	return v.Struct(c)
}

// New creates a new Helm chart scalar component. A Helm Chart scalar can represent any Helm Chart entity.
// Typically, a custom Helm Chart scalar should be created that depends on this component for ease of use.
// This method will attempt to resolve the chart using a locally installed version of Helm.
func New(parentCtx context.Context, props crib.Props) (component crib.Component, retErr error) {
	var errs error
	// cdk8s.NewHelm can panic if the chart is not found, so we need to handle that.
	defer func() {
		if r := recover(); r != nil {
			retErr = errors.Join(errs, fmt.Errorf("failed to create helm chart: %v", r))
		}
	}()
	chartProps := dry.MustAs[*ChartProps](props)
	commonLabels := dry.PtrMapping(map[string]string{
		"helm.crib.sdk/chart":     chartProps.Chart,
		"helm.crib.sdk/namespace": chartProps.Namespace,
		"helm.crib.sdk/release":   chartProps.ReleaseName,
		"helm.crib.sdk/name":      chartProps.Name,
	})

	// Determine the location of the Helm binary on the system.
	prog, err := helmBinary()
	if err := errors.Join(errs, err, props.Validate(parentCtx)); err != nil {
		return nil, err
	}

	parent := internal.ConstructFromContext(parentCtx)
	// The HelmChart component needs to exist in a cdk8s chart so that it can own
	// the namespace of the deployed chart.
	// TODO: Need to append the parent node id to the resource id so that it is not lost.
	chart := cdk8s.NewChart(parent, crib.ResourceID("sdk.HelmChart", props), &cdk8s.ChartProps{
		Namespace: dry.ToPtr(chartProps.Namespace),
		Labels:    commonLabels,
	})
	ctx := internal.ContextWithConstruct(parentCtx, chart)

	cmdProps, err := helmProps(prog, chartProps)
	if err != nil {
		return nil, err
	}
	// Important: This method resolves the full helm chart, making a network call
	// when initialized. This method panics if the chart cannot be resolved.
	// It's important to catch and handle the panic.
	deployment := cdk8s.NewHelm(chart, crib.ResourceID(chartProps.Chart, props), cmdProps)

	// Ensure that the namespace is created before the chart.
	ns, err := namespace.New(ctx, &namespace.Props{
		Namespace: chartProps.Namespace,
	})
	if err != nil {
		return nil, errors.Join(errs, err)
	}
	chart.Node().AddDependency(ns)

	// If the chart should wait for resources to be ready before returning, we add a client-side apply to wait.
	if chartProps.WaitForReady {
		labels := lo.MapToSlice(dry.FromPtr(commonLabels), func(label string, value *string) string {
			return fmt.Sprintf("%s=%s", label, dry.FromPtr(value))
		})
		slices.Sort(labels)

		// If the chart is configured to wait for resources to be ready, we need to add a dependency
		// that will wait for the resources to be ready before returning.
		waitFor, err := clientsideapply.New(parentCtx, &clientsideapply.Props{
			Namespace: chartProps.Namespace,
			OnFailure: "abort",
			Action:    "kubectl",
			Args: []string{
				"wait",
				"-n", chartProps.Namespace,
				"--for=condition=ready",
				"pod",
				fmt.Sprintf("-l=%s", strings.Join(labels, ",")),
				"--timeout=600s",
			},
		})
		if err != nil {
			return nil, errors.Join(errs, err)
		}
		waitFor.Node().AddDependency(deployment)
	}

	return chart, nil
}

// helmProps converts ChartProps to cdk8s.HelmProps. Depending on whether the chart is an OCI chart or a
// regular Helm chart, it will set the appropriate fields.
func helmProps(executable string, props *ChartProps) (*cdk8s.HelmProps, error) {
	r := &helm.Release{
		Name:        props.Name,
		ReleaseName: props.ReleaseName,
		Repository:  props.Repo,
		Version:     props.Version,
	}
	if r.IsOCI() {
		// OCI charts use the full repository URL as the chart name and don't use ReleaseName or Repo.
		return &cdk8s.HelmProps{
			Chart:          jsii.String(r.PullRef()),
			Namespace:      jsii.String(props.Namespace),
			ReleaseName:    jsii.String(props.ReleaseName),
			Values:         dry.ToPtr(props.Values),
			Version:        jsii.String(r.ChartVersion().Version),
			HelmExecutable: jsii.String(executable),
			HelmFlags:      dry.PtrSlice(props.Flags),
		}, nil
	}

	return &cdk8s.HelmProps{
		Chart:          jsii.String(props.Chart),
		Namespace:      jsii.String(props.Namespace),
		ReleaseName:    jsii.String(props.ReleaseName),
		Repo:           jsii.String(props.Repo),
		Values:         dry.ToPtr(props.Values),
		Version:        jsii.String(props.Version),
		HelmExecutable: jsii.String(executable),
		HelmFlags:      dry.PtrSlice(props.Flags),
	}, nil
}
