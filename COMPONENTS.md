# CRIB Components Best Practices

## Component Names
Each component should expose a constant with the component name, which allows referencing it in result queries.

```go
const ComponentName = "sdk.composite.chainlink.jd.v1"
```

The name format follows this structure:
```
sdk.<ComponentType>.<SubPackages>.<Name>.<Version>
```

**Format breakdown:**
- `ComponentType` is either `scalar` or `composite`
- `SubPackages` contains the path to a component package (this helps avoid naming conflicts)
- `Name` is the same as the component package name
- `Version` specifies the component version

## Usage
Once the `ComponentName` constant is defined, it should be used for setting the ResourceID:

**Example:**
```go
parent := internal.ConstructFromContext(ctx)
chart := cdk8s.NewChart(parent, crib.ResourceID(ComponentName, props), nil)
ctx = internal.ContextWithConstruct(ctx, chart)

```
