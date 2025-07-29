# Anvil Persistence Example

This example demonstrates how to use the Anvil blockchain component with persistence enabled. The Anvil component can be configured to use either a Deployment (without persistence) or a StatefulSet (with persistence) based on your requirements.

## Features Demonstrated

1. **Basic Persistence**: Enable persistence with default settings (2Gi storage, gp3 storage class)
2. **Custom Persistence**: Configure custom storage size and storage class
3. **Persistence with Ingress**: Combine persistence with ingress exposure

## Usage

### Basic Persistence

```go
anvilWithPersistence := anvilv1.Component(&anvilv1.Props{
    Namespace: "crib-local",
    ChainID:   "1337",
}, anvilv1.UsePersistence)
```

### Custom Persistence Configuration

```go
anvilWithCustomPersistence := anvilv1.Component(&anvilv1.Props{
    Namespace: "crib-local",
    ChainID:   "31337",
}, anvilv1.UsePersistenceWithConfig("5Gi", "fast-ssd"))
```

### Persistence with Ingress

```go
anvilWithPersistenceAndIngress := anvilv1.Component(&anvilv1.Props{
    Namespace: "crib-local",
    ChainID:   "9999",
}, anvilv1.UsePersistence, anvilv1.UseIngress)
```

## Persistence Details

When persistence is enabled:

- **Workload Type**: Uses Kubernetes StatefulSet instead of Deployment
- **Storage**: Creates a PersistentVolumeClaim (PVC) for blockchain state
- **State Path**: Anvil state is stored at `/data/anvil/anvil_state.json`
- **Volume Mount**: The `/data` directory is mounted as a persistent volume
- **State Persistence**: Blockchain state survives pod restarts and reschedules

## Default Configuration

- **Storage Size**: 2Gi (configurable)
- **Storage Class**: gp3 (configurable)
- **Access Mode**: ReadWriteOnce
- **State Path**: `/data/anvil/anvil_state.json`

## Running the Example

```bash
cd examples/anvil-persistence-example
go run main.go
```

## Requirements

- Kubernetes cluster with persistent volume support
- Storage class available (default: gp3)
- CRIB SDK properly configured

## Notes

- Persistence requires a cloud environment or cluster with persistent volume support
- Local development clusters (like kind) may not support persistence
- The blockchain state will be preserved across pod restarts when persistence is enabled 