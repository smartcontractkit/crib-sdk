package jdv1

import (
	"context"
	"fmt"
	"strings"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"

	jdv1 "github.com/smartcontractkit/crib-sdk/crib/scalar/charts/jd/v1"
	postgresv1 "github.com/smartcontractkit/crib-sdk/crib/scalar/charts/postgres/v1"
	clientsideapply "github.com/smartcontractkit/crib-sdk/crib/scalar/clientsideapply/v1"
	helmchartv1 "github.com/smartcontractkit/crib-sdk/crib/scalar/helmchart/v1"
)

const ComponentName = "sdk.composite.chainlink.jd.v1"

// DBProps contains properties for the database component.
type DBProps struct {
	Namespace     string
	ReleaseName   string
	Values        map[string]any
	ValuesPatches [][]string
}

// JDProps contains properties specific to the JD component.
type JDProps struct {
	Namespace     string
	ReleaseName   string
	Values        map[string]any
	ValuesPatches [][]string
}

type Props struct {
	JD JDProps
	DB DBProps
}

// Validate validates the props.
func (p *Props) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(p)
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

	// Create PostgreSQL database for JD
	pg, err := postgresv1.Component(&helmchartv1.ChartProps{
		Namespace:     jdProps.DB.Namespace,
		ReleaseName:   jdProps.DB.ReleaseName,
		Values:        jdProps.DB.Values,
		ValuesPatches: jdProps.DB.ValuesPatches,
	})(ctx)
	if err != nil {
		return nil, err
	}

	// Wait for PostgreSQL to be ready
	labels := []string{
		fmt.Sprintf("statefulset.kubernetes.io/pod-name=%s-0", jdProps.DB.ReleaseName),
		"app.kubernetes.io/name=postgresql",
	}
	waitForDB, err := clientsideapply.New(ctx, &clientsideapply.Props{
		Namespace: jdProps.DB.Namespace,
		OnFailure: "abort",
		Action:    "kubectl",
		Args: []string{
			"wait",
			"-n", jdProps.DB.Namespace,
			"--for=condition=ready",
			"pod",
			fmt.Sprintf("-l=%s", strings.Join(labels, ",")),
			"--timeout=600s",
		},
	})
	if err != nil {
		return nil, err
	}

	// Create JD component
	jd, err := jdv1.Component(&helmchartv1.ChartProps{
		Namespace:     jdProps.DB.Namespace,
		ReleaseName:   jdProps.JD.ReleaseName,
		Values:        jdProps.JD.Values,
		ValuesPatches: jdProps.JD.ValuesPatches,
	})(ctx)
	if err != nil {
		return nil, err
	}

	// Wait for JD to be ready
	labels = []string{
		fmt.Sprintf("app.kubernetes.io/component=%s", jdProps.JD.ReleaseName),
		"app.kubernetes.io/name=devspace-app",
	}
	waitForJD, err := clientsideapply.New(ctx, &clientsideapply.Props{
		Namespace: jdProps.JD.Namespace,
		OnFailure: "abort",
		Action:    "kubectl",
		Args: []string{
			"wait",
			"-n", jdProps.JD.Namespace,
			"--for=condition=ready",
			"pod",
			fmt.Sprintf("-l=%s", strings.Join(labels, ",")),
			"--timeout=600s",
		},
	})
	if err != nil {
		return nil, err
	}

	// Set up dependencies
	// Use ComponentState to access the underlying chart components from Result objects
	waitForDB.Node().AddDependency(pg)
	jd.Node().AddDependency(crib.ComponentState[*clientsideapply.Result](waitForDB).Component)
	crib.ComponentState[*clientsideapply.Result](waitForJD).Component.Node().AddDependency(jd)

	return waitForJD, nil
}
