# NodeSet Composite Component

The NodeSet composite component provides a convenient way to deploy multiple Chainlink nodes with a shared PostgreSQL database. This component automatically:

1. Creates a PostgreSQL database with a "root" superuser
2. Generates initialization SQL scripts to create individual databases and users for each Chainlink node
3. Deploys the specified number of Chainlink nodes, each connected to its own database

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     NodeSet Component                       │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────┐    ┌─────────────────┐                │
│  │   PostgreSQL    │    │  Init SQL       │                │
│  │   Database      │◄───┤  Scripts        │                │
│  │                 │    │                 │                │
│  │  ┌─────────────┐│    │ • User 0        │                │
│  │  │ chainlink_  ││    │ • Database 0    │                │
│  │  │ node_0      ││    │ • User 1        │                │
│  │  └─────────────┘│    │ • Database 1    │                │
│  │  ┌─────────────┐│    │ • User N-1      │                │
│  │  │ chainlink_  ││    │ • Database N-1  │                │
│  │  │ node_1      ││    └─────────────────┘                │
│  │  └─────────────┘│                                       │
│  │  ┌─────────────┐│                                       │
│  │  │    ...      ││                                       │
│  │  └─────────────┘│                                       │
│  └─────────────────┘                                       │
│                                                             │
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐│
│  │ Chainlink       │  │ Chainlink       │  │   ...        ││
│  │ Node 0          │  │ Node 1          │  │              ││
│  │                 │  │                 │  │              ││
│  │ DB: node_0      │  │ DB: node_1      │  │ DB: node_N-1 ││
│  │ User: user_0    │  │ User: user_1    │  │ User: user_N ││
│  └─────────────────┘  └─────────────────┘  └──────────────┘│
└─────────────────────────────────────────────────────────────┘
```

## Properties

The NodeSet component accepts the following properties:

- **`Namespace`** (required): Kubernetes namespace for all resources
- **`Size`** (required): Number of Chainlink nodes to create (minimum 1)
- **`NodeProps`** (required): Array of Chainlink node configurations (must match Size)
- **`PostgresReleaseName`** (optional): Release name for PostgreSQL (default: "shared-postgres")
- **`PostgresPassword`** (optional): PostgreSQL superuser password (default: "postgres")

## Database Configuration

Each Chainlink node gets its own:
- Database: `chainlink_node_N` where N is the node index (0-based)
- User: `chainlink_user_N` with password `chainlink_pass_N`
- Full privileges on its respective database

The PostgreSQL connection URL is automatically generated for each node as:
```
postgresql://chainlink_user_N:chainlink_pass_N@{PostgresReleaseName}:5432/chainlink_node_N?sslmode=disable
```

## Usage Example

```go
package main

import (
    "github.com/smartcontractkit/crib-sdk/crib"
    chainlinknodev1 "github.com/smartcontractkit/crib-sdk/crib/composite/chainlink/node/v1"
    nodesetv1 "github.com/smartcontractkit/crib-sdk/crib/composite/chainlink/nodeset/v1"
)

func main() {
    plan := crib.NewPlan(
        "my-nodeset",
        crib.Namespace("chainlink-cluster"),
        crib.ComponentSet(
            nodesetv1.Component(&nodesetv1.Props{
                Namespace:           "chainlink-cluster",
                Size:                3,
                PostgresReleaseName: "shared-postgres",
                PostgresPassword:    "supersecret",
                NodeProps: []*chainlinknodev1.Props{
                    {
                        Namespace:   "chainlink-cluster",
                        ReleaseName: "chainlink-node-0",
                        Image:       "chainlink/chainlink:2.18.0",
                        Config:      "...", // Chainlink TOML config
                    },
                    {
                        Namespace:   "chainlink-cluster",
                        ReleaseName: "chainlink-node-1",
                        Image:       "chainlink/chainlink:2.18.0",
                        Config:      "...", // Chainlink TOML config
                    },
                    {
                        Namespace:   "chainlink-cluster",
                        ReleaseName: "chainlink-node-2",
                        Image:       "chainlink/chainlink:2.18.0",
                        Config:      "...", // Chainlink TOML config
                    },
                },
            }),
        ),
    )
}
```

## Features

- **Automatic Database Setup**: Creates isolated databases for each Chainlink node
- **Flexible Node Configuration**: Each node can have different configurations, resources, and settings
- **Shared Infrastructure**: Uses a single PostgreSQL instance to minimize resource usage
- **Security**: Each node has its own database user with restricted privileges
- **Validation**: Ensures Size matches the number of NodeProps provided

## Benefits

1. **Resource Efficiency**: Single PostgreSQL instance serves multiple nodes
2. **Isolation**: Each node has its own database and user for security
3. **Scalability**: Easy to add or remove nodes by changing the Size and NodeProps
4. **Consistency**: Standardized approach to multi-node deployments
5. **Simplicity**: Single component manages the entire stack

## Running the Example

```bash
cd examples/nodeset-example
go run main.go
```

This will create a plan with 3 Chainlink nodes, each with different configurations demonstrating various features like custom ports, resource limits, and configuration overrides.
