package nodesetv1

import (
	"context"
	"fmt"
	"strings"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"

	chainlinknodev1 "github.com/smartcontractkit/crib-sdk/crib/composite/chainlink/node/v1"
	postgresv1 "github.com/smartcontractkit/crib-sdk/crib/scalar/charts/postgres/v1"
	clientsideapplyv1 "github.com/smartcontractkit/crib-sdk/crib/scalar/clientsideapply/v1"
	helmchartv1 "github.com/smartcontractkit/crib-sdk/crib/scalar/helmchart/v1"
)

const (
	ComponentName = "sdk.composite.chainlink.nodeset.v1"
)

// Props contains properties for the NodeSet composite component.
type Props struct {
	Namespace           string                   `validate:"required"`
	PostgresReleaseName string                   `default:"shared-postgres"`
	PostgresPassword    string                   `default:"postgres"`
	NodeProps           []*chainlinknodev1.Props `validate:"required"`
	Size                int                      `validate:"required,min=1"`
}

type Result struct {
	crib.Component
	Nodes []*chainlinknodev1.Result
}

// Validate validates the props.
func (p *Props) Validate(ctx context.Context) error {
	v := internal.ValidatorFromContext(ctx)

	if err := v.Struct(p); err != nil {
		return err
	}

	// Validate that the number of NodeProps matches Size
	if len(p.NodeProps) != p.Size {
		return fmt.Errorf("NodeProps length (%d) must match Size (%d)", len(p.NodeProps), p.Size)
	}
	// Set the namespace for each node props and validate it
	// as we had to remove "dive" from struct tags
	for i, nodeProps := range p.NodeProps {
		if nodeProps.Namespace != "" {
			return fmt.Errorf("NodeProps[%d].Namespace must be empty to allow automatic propagation", i)
		}
		nodeProps.Namespace = p.Namespace
		p.NodeProps[i] = nodeProps
		if err := nodeProps.Validate(ctx); err != nil {
			return err
		}
	}

	return nil
}

// Component returns a new NodeSet composite component.
func Component(props *Props) crib.ComponentFunc {
	return func(ctx context.Context) (crib.Component, error) {
		if err := props.Validate(ctx); err != nil {
			return nil, err
		}
		return nodeSet(ctx, props)
	}
}

// nodeSet creates and returns a new NodeSet composite component.
func nodeSet(ctx context.Context, props crib.Props) (crib.Component, error) {
	parent := internal.ConstructFromContext(ctx)
	chart := cdk8s.NewChart(parent, crib.ResourceID(ComponentName, props), nil)
	ctx = internal.ContextWithConstruct(ctx, chart)

	nodeSetProps := dry.MustAs[*Props](props)

	// Generate initialization SQL script for creating databases and users
	initSQL := generateInitSQL(nodeSetProps)

	// Create PostgreSQL database with custom initialization scripts
	_, err := postgresv1.Component(&helmchartv1.ChartProps{
		Namespace:   nodeSetProps.Namespace,
		ReleaseName: nodeSetProps.PostgresReleaseName,
		Values: map[string]any{
			"fullnameOverride": nodeSetProps.PostgresReleaseName,
			"auth": map[string]any{
				"enablePostgresUser": true,
				"postgresPassword":   nodeSetProps.PostgresPassword,
			},
			"primary": map[string]any{
				"persistence": map[string]any{
					"enabled": false,
				},
				"initdb": map[string]any{
					"scripts": map[string]string{
						"init.sql": initSQL,
					},
				},
			},
		},
	})(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create PostgreSQL component: %w", err)
	}

	results := make([]*chainlinknodev1.Result, 0, len(nodeSetProps.NodeProps))

	// Create Chainlink nodes
	for i, nodeProps := range nodeSetProps.NodeProps {
		// Generate database URL for this specific node
		dbName := fmt.Sprintf("chainlink_node_%d", i)
		username := fmt.Sprintf("chainlink_user_%d", i)
		password := fmt.Sprintf("chainlink_pass_%d", i)

		dbURL := fmt.Sprintf("postgresql://%s:%s@%s:5432/%s?sslmode=disable",
			username, password, nodeSetProps.PostgresReleaseName, dbName)

		// Copy node props and set the database URL
		nodePropsWithDB := *nodeProps
		nodePropsWithDB.DatabaseURL = dbURL
		// Create the Chainlink node component
		result, err := chainlinknodev1.Component(&nodePropsWithDB)(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create Chainlink node %d: %w", i, err)
		}
		// Convert the result to the expected type
		nodeResult := dry.MustAs[chainlinknodev1.Result](result)
		results = append(results, &nodeResult)
	}

	// Wait for all Chainlink nodes to be ready
	waitForNodes, err := clientsideapplyv1.New(ctx, &clientsideapplyv1.Props{
		Namespace: nodeSetProps.Namespace,
		OnFailure: "abort",
		Action:    "kubectl",
		Args: []string{
			"wait",
			"-n", nodeSetProps.Namespace,
			"--for=condition=ready",
			"pod",
			"-l=app.kubernetes.io/name=chainlink",
			"--timeout=600s",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create wait for nodes: %w", err)
	}

	// Set up dependencies: wait for all nodes to be created before checking if they're ready
	for _, nodeResult := range results {
		waitForNodes.Node().AddDependency(nodeResult.Component)
	}

	return Result{
		Component: chart,
		Nodes:     results,
	}, nil
}

// generateInitSQL creates the SQL script to initialize databases and users for each Chainlink node.
func generateInitSQL(props *Props) string {
	var sqlStatements []string

	for i := 0; i < props.Size; i++ {
		dbName := fmt.Sprintf("chainlink_node_%d", i)
		username := fmt.Sprintf("chainlink_user_%d", i)
		password := fmt.Sprintf("chainlink_pass_%d", i)

		// Create user
		//nolint:gocritic // for readability
		sqlStatements = append(sqlStatements, fmt.Sprintf(
			"CREATE USER %s WITH PASSWORD '%s';", username, password))

		// Create database
		sqlStatements = append(sqlStatements, fmt.Sprintf(
			"CREATE DATABASE %s OWNER %s;", dbName, username))

		// Grant privileges
		sqlStatements = append(sqlStatements, fmt.Sprintf(
			"GRANT ALL PRIVILEGES ON DATABASE %s TO %s;", dbName, username))
	}

	return strings.Join(sqlStatements, "\n")
}
