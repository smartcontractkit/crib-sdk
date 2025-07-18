# Individual Node Deployment Example

This example demonstrates how to deploy multiple Chainlink nodes as individual components, where each node gets its own dedicated PostgreSQL database. This approach provides maximum isolation and flexibility compared to shared database architectures.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                Individual Node Deployment                   │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │ Chainlink       │  │ Chainlink       │  │   ...        │ │
│  │ Node 0          │  │ Node 1          │  │              │ │
│  │                 │  │                 │  │              │ │
│  │ ┌─────────────┐ │  │ ┌─────────────┐ │  │ ┌───────────┐│ │
│  │ │ PostgreSQL  │ │  │ │ PostgreSQL  │ │  │ │PostgreSQL ││ │
│  │ │ Database    │ │  │ │ Database    │ │  │ │ Database  ││ │
│  │ │ (Dedicated) │ │  │ │ (Dedicated) │ │  │ │(Dedicated)││ │
│  │ └─────────────┘ │  │ └─────────────┘ │  │ └───────────┘│ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
│                                                             │
│  Each node is deployed as a separate component with:        │
│  • Individual PostgreSQL instance                           │
│  • Isolated configuration                                   │
│  • Independent lifecycle management                         │
│  • Dedicated API and P2P ports                              │
└─────────────────────────────────────────────────────────────┘
```

## Key Features

- **Database Isolation**: Each Chainlink node has its own PostgreSQL database instance
- **Independent Scaling**: Nodes can be scaled, updated, or removed independently
- **Flexible Configuration**: Each node can have different configurations, resources, and settings
- **Automatic Database Creation**: PostgreSQL databases are automatically created for each node
- **No Shared Dependencies**: No single point of failure from shared infrastructure

## How It Works

The example creates multiple Chainlink nodes using individual `nodev1.Component()` calls:

1. **Component Creation**: Each node is created as a separate component function
2. **Database Provisioning**: Each node automatically gets its own PostgreSQL instance when `DatabaseURL` is not specified
3. **Plan Execution**: All components are deployed together in a single plan
4. **Result Collection**: API URLs and credentials are collected from each deployed node

## Usage Example

```go
func nodesetFromNodes() {
    componentFuncs := make([]crib.ComponentFunc, 0)

    ns := "crib-local"
    for i := 0; i < 5; i++ {
        cFunc := nodev1.Component(&nodev1.Props{
            Namespace:       ns,
            AppInstanceName: fmt.Sprintf("chainlink-%d", i),
            Image:           "localhost:5001/chainlink:nightly-20250624-plugins",
            Config: `
                [WebServer]
                AllowOrigins = '\*'
                SecureCookies = false

                [WebServer.TLS]
                HTTPSPort = 0
            `,
            ConfigOverrides: map[string]string{
                "overrides.toml": `
                    [Log]
                    Level = 'debug'
                    JSONConsole = true
                `,
            },
        })
        componentFuncs = append(componentFuncs, cFunc)
    }

    plan := crib.NewPlan(
        "nodesets",
        crib.Namespace(ns),
        crib.ComponentSet(componentFuncs...),
    )

    planState, err := plan.Apply(context.Background())
    // Handle results...
}
```

## Configuration Options

Each node can be configured independently with:

- **Custom Images**: Different Chainlink versions per node
- **Resource Limits**: Individual CPU and memory allocations
- **Port Configuration**: Custom API and P2P ports
- **Environment Variables**: Node-specific environment settings
- **Config Overrides**: Additional TOML configuration files
- **Secrets**: Individual secret configurations

## Database Configuration

When `DatabaseURL` is not specified in the node props:

- A dedicated PostgreSQL instance is automatically created
- Database name: `chainlink` (default)
- Username/Password: `staticlongpassword` (default)
- Connection URL format: `postgresql://chainlink:staticlongpassword@{nodeName}-postgres:5432/chainlink?sslmode=disable`

## Benefits vs Shared Database Approach

### Individual Database Benefits:
- **Complete Isolation**: Database failures don't affect other nodes
- **Independent Scaling**: Each database can be sized appropriately
- **Flexible Upgrades**: Database versions can be upgraded per node
- **Simplified Debugging**: Issues are isolated to individual nodes
- **Security**: No cross-node data access concerns

### Trade-offs:
- **Resource Usage**: Higher resource consumption due to multiple database instances
- **Complexity**: More components to manage and monitor
- **Cost**: Higher infrastructure costs in cloud environments

## Running the Example

```bash
cd examples/nodeset-example
go run main.go
```

This will:
1. Create 5 individual Chainlink nodes
2. Deploy each with its own PostgreSQL database
3. Configure each node with debug logging
4. Print API URLs and credentials for each node

## Output

The example will output information for each deployed node:

```
Node API URL: http://chainlink-0.crib-local.svc.cluster.local:6688
API Credentials: username: admin@chain.link , password: staticlongpassword
Node API URL: http://chainlink-1.crib-local.svc.cluster.local:6688
API Credentials: username: admin@chain.link , password: staticlongpassword
...
```

## When to Use This Approach

Choose individual node deployment when:

- **High Availability**: Need maximum isolation between nodes
- **Different Configurations**: Nodes require significantly different setups
- **Independent Lifecycle**: Nodes need to be managed separately
- **Compliance Requirements**: Regulatory requirements for data isolation
- **Development/Testing**: Need to test different node configurations

For scenarios requiring shared infrastructure or resource optimization, consider using the `nodeset` composite component instead.

## Related Examples

- [NodeSet with Shared PostgreSQL](../nodeset-shared-postgres-example/README.md) - For resource-efficient deployments with shared database
- [Single Node Example](../single-node-example/README.md) - For deploying a single Chainlink node
