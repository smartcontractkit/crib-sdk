# Scratch

## Generate scalar component for helm chart

### Current

To generate a scalar for a Helm chart, use the `cribctl helm create-component` command. This command creates a new CRIB-SDK Scalar Component from a Helm Chart.

Basic usage:
```bash
cribctl helm create-component <name> <release>@<url> [--version=<version>] [--saveto=<path>] [--scalar-version=<version>]
```

Example for creating an Anvil component:
```bash
# Create a scalar component named anvil in crib/scalar/charts/anvil/v1
cribctl helm create-component anvil component-chart@https://charts.devspace.sh --version=0.9.1
```

The command will:
1. Create a new directory structure at `crib/scalar/charts/<name>/<scalar-version>` (default: `v1`)
2. Generate the necessary Go files for the component
3. Create a `chart.defaults.yaml` file with default values
4. Set up the component to use the specified Helm chart

The generated component will include:
- A main component file (e.g., `anvil.go`)
- Default values in `chart.defaults.yaml`
- Test files and test data
- Proper integration with the CRIB-SDK framework

The component can then be used in your CRIB-SDK application to manage the Helm chart deployment.

### Hack

- update `chart.defaults.yaml` + `testdata/values.yaml` w/ proper values
- if valuespatches are needed add to `component.go`:
  ```golang
  // TEMP: Apply the values patches.
  for _, patch := range chartProps.ValuesPatches {
      values = internal.SetValueAtPath(values, patch[0], patch[1])
  }
  ```
- run `task go:test` to generate snapshot


## chainlink node secret store

1. generate node secrets and store them in the secret store
2. check if the secret store is available
3a. if the secret store is not available, create a secret store resource
3b. if the secret store is available, read the node secrets from the secret store
4. create the node secret resource
