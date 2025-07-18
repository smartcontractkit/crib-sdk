# CRIB SDK Development Guidelines

## Code Style and Structure

1. Follow Go best practices and standard formatting:
    - Use `gofmt` for basic code formatting
    - Use `goimports` for import organization
    - Use `golangci-lint` for comprehensive linting
    - Follow Go naming conventions
    - Keep functions focused and small
    - Use meaningful variable and function names
    - Prefer using Google's and Uber's style guide when authoring Go code
    - Prefer using internally available packages over external packages or standard library packages
      - Most "helper" libraries are found under `internal/core/common` or `internal/adapter`

2. Component Development:
    - Place scalar components in `crib/scalar/[component]/v1/`
    - Place composite components in `crib/composite/`
    - Each component should have clear props and validation
    - Include proper documentation and examples

3. High-level Directory Structure:
    - GitHub Actions workflows in `.github/workflows`
    - API schemas like OpenAPI in `api`
    - Build files like Dockerfile in `build`
    - CLI-related files in `cmd`
    - SDK "Plans" in `contrib`
    - Components in `crib/scalar` and `crib/composite`
    - Core SDK exported methods in `crib`
    - Deployment automation like Docker Compose in `deployments`
    - Examples in `examples`
    - Hack scripts in `hack`
    - Adapters (Hexagonal Architecture) in `internal/adapter`
    - Common utilities in `internal/core/common`
    - Domain objects in `internal/core/domain`
    - Interfaces in `internal/core/interfaces`
    - Services in `internal/core/services`
    - Taskfiles in `taskfiles`
    - E2E tests in `tests`

4. Testing Requirements:
    - Write unit tests for all new components using `testify`
    - Use `testify/assert` for assertions and `testify/require` for critical checks
    - Use `testify/suite` for test suites where appropriate
    - Include integration tests for composite components
    - Maintain test coverage above 80%
    - Use table-driven tests where appropriate
    - Mock external dependencies using `testify/mock`
    - Use `testify/assert.Equal` for value comparisons
    - Use `testify/assert.NoError` for error checking
    - Use `testify/assert.NotNil` for pointer validation

## Documentation

1. Code Documentation:
    - Document all exported functions and types
    - Include usage examples in comments
    - Keep README.md up to date
    - Document any breaking changes

2. API Documentation:
    - Document all public APIs
    - Include parameter descriptions
    - Provide usage examples
    - Document error cases

## Development Workflow

1. Pre-commit Checks:
    - Run `task go:lint` before committing
    - Run `task go:test` to ensure tests pass
    - Run `task go:fmt` to format code
    - Check for any security vulnerabilities

2. Pull Request Process:
    - Update documentation if needed
    - Add tests for new features
    - Ensure CI passes
    - Get at least one review

## Component Guidelines

1. Scalar Components:
    - Keep components simple and focused
    - Implement proper validation
    - Handle errors gracefully
    - Include proper logging

2. Composite Components:
    - Document dependencies clearly
    - Handle component ordering
    - Include proper error handling
    - Provide clear configuration options

## Security Guidelines

1. General Security:
    - Never hardcode sensitive information
    - Use secure defaults
    - Validate all inputs
    - Follow principle of least privilege

2. Kubernetes Security:
    - Use appropriate RBAC
    - Follow security best practices
    - Implement proper resource limits
    - Use network policies where needed

## Performance Guidelines

1. Resource Usage:
    - Optimize memory usage
    - Minimize API calls
    - Use appropriate caching
    - Monitor resource consumption

2. Scalability:
    - Design for horizontal scaling
    - Use efficient data structures
    - Implement proper error handling
    - Consider edge cases

## Error Handling

1. Error Management:
    - Use proper error wrapping
    - Include context in errors
    - Handle all error cases
    - Provide clear error messages

2. Logging:
    - Use appropriate log levels
    - Include relevant context
    - Follow logging best practices
    - Consider log rotation

## Versioning

1. Version Management:
    - Follow semantic versioning
    - Document breaking changes
    - Maintain backward compatibility
    - Update version numbers appropriately

2. Release Process:
    - Create release notes
    - Tag releases properly
    - Update documentation
    - Announce breaking changes

## Promptfiles and AI Agent Instructions

1. Instructions for Copilot and other Agent AI tooling can be found in this document (`.github/copilot-instructions.md`)
2. Instructions for authoring good git commit messages can be found in this document located at `.github/git-commit-instructions.md`
3. Instructions for when and how to create Weekly Stakeholder Status Updates can be found in `.github/weekly-status-agent-instructions.md`
