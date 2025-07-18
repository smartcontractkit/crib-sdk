# Git Commit Instructions for CRIB SDK

## Overview

This document provides standardized git commit guidelines for the CRIB SDK project, designed to ensure consistency, clarity, and maintainability across all contributions, especially for automated/agentic systems.

## Commit Message Format

Follow the Conventional Commits specification with the following structure:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Types

Use one of the following commit types:

- **feat**: A new feature for the user or a new component
- **fix**: A bug fix or correction
- **docs**: Documentation only changes
- **style**: Changes that do not affect the meaning of the code (white-space, formatting, missing semi-colons, etc)
- **refactor**: A code change that neither fixes a bug nor adds a feature
- **perf**: A code change that improves performance
- **test**: Adding missing tests or correcting existing tests
- **build**: Changes that affect the build system or external dependencies
- **ci**: Changes to CI configuration files and scripts
- **chore**: Other changes that don't modify src or test files
- **revert**: Reverts a previous commit

### Scope (Optional)

The scope should indicate the area of the codebase affected:

- **cribctl**: CLI-related changes
- **scalar**: Scalar component changes
- **composite**: Composite component changes
- **core**: Core SDK functionality
- **api**: API schema changes
- **helm**: Helm chart related changes
- **k8s**: Kubernetes resource changes
- **testing**: Test infrastructure changes
- **contrib**: Plan registry changes
- **internal**: Internal package changes

### Description

- Use imperative mood ("add" not "added" or "adds")
- Keep it concise (50 characters or less)
- Don't capitalize the first letter
- Don't end with a period
- Be specific about what changed

### Body (Optional)

- Explain the motivation for the change
- Contrast this with previous behavior
- Include any breaking changes
- Wrap at 72 characters

### Footer (Optional)

- Reference issues and pull requests
- Note breaking changes with "BREAKING CHANGE:"
- Include co-authors if applicable

## Examples

### Good Commit Messages

```
feat(scalar): add new anvil component with custom configuration

- Implements anvil blockchain component with configurable ports
- Adds validation for network parameters
- Includes comprehensive test coverage

Closes #123
```

```
fix(cribctl): resolve helm chart validation error

The helm chart validator was incorrectly rejecting valid schemas
due to case sensitivity in property names.

Fixes #456
```

```
docs: update component development guidelines

- Add section on error handling best practices
- Include examples for composite components
- Update testing requirements
```

```
refactor(core): simplify plan execution logic

- Extract common validation into shared utility
- Remove duplicate error handling code
- Improve readability without changing behavior
```

```
test(composite): add integration tests for node component

- Test component lifecycle management
- Verify configuration propagation
- Add edge case scenarios
```

### Poor Commit Messages to Avoid

```
// Too vague
fix: bug fix

// Not descriptive
update files

// Past tense
added new feature

// Too long description
feat: add a new very comprehensive and detailed anvil component that supports multiple configurations and various network parameters with extensive validation

// Missing type
resolve issue with helm charts
```

## Commit Frequency and Size

### Best Practices

1. **Make atomic commits**: Each commit should represent a single logical change
2. **Commit frequently**: Don't let changes accumulate for too long
3. **Keep commits focused**: Don't mix unrelated changes in a single commit
4. **Complete work**: Each commit should leave the codebase in a working state

### Guidelines for Agentic Systems

1. **Validate before committing**: Always run tests and linting before creating commits
2. **Group related changes**: When making multiple related changes, organize them logically
3. **Document complex changes**: Include detailed commit messages for non-trivial changes
4. **Reference context**: Link to relevant issues, documentation, or discussions

## Branch Naming Conventions

Use the following patterns for branch names:

- `feat/component-name-description` - New features
- `fix/issue-description` - Bug fixes
- `docs/section-being-updated` - Documentation updates
- `refactor/area-being-refactored` - Code refactoring
- `test/component-or-area` - Test additions/improvements

Examples:
- `feat/anvil-component-validation`
- `fix/helm-chart-deployment-error`
- `docs/component-development-guide`
- `refactor/plan-execution-logic`

## Pre-commit Checklist

Before creating any commit, ensure:

- [ ] Code follows Go formatting standards (`task go:fmt`)
- [ ] All tests pass (`task go:test`)
- [ ] Linting passes (`task go:lint`)
- [ ] Documentation is updated if needed
- [ ] Commit message follows the specified format
- [ ] Changes are atomic and focused
- [ ] No sensitive information is included

## Special Considerations for Agentic Systems

### Automated Commits

When making automated commits:

1. **Be explicit about automation**: Include context about the automated nature
2. **Provide detailed descriptions**: Since humans may not have full context
3. **Reference source requests**: Link back to the original request or issue
4. **Include validation results**: Mention that automated checks passed

Example:
```
feat(scalar): implement bootstrap component per user request

Automated implementation based on user requirements:
- Configurable timeout and retry parameters
- Integration with existing plan registry
- Comprehensive error handling and validation

All automated tests pass. Manual review recommended for complex logic.

Ref: User request #789
```

### Error Recovery

If automated changes cause issues:

1. Create a revert commit immediately
2. Use descriptive commit message explaining the revert
3. Include the original commit hash being reverted

Example:
```
revert: revert "feat(scalar): implement bootstrap component"

This reverts commit abc123def456 due to test failures in CI.
Automated validation missed integration test dependencies.

Requires manual review before re-implementation.
```

## Integration with Development Workflow

### Pull Request Process

1. Create feature branch with descriptive name
2. Make focused, well-documented commits
3. Ensure all commits pass pre-commit checks
4. Create pull request with clear description
5. Link to relevant issues and documentation

### Continuous Integration

All commits must:
- Pass automated tests
- Meet code coverage requirements
- Pass security scans
- Follow formatting standards

## Tools and Automation

### Recommended Tools

- **commitizen**: For interactive commit message creation
- **conventional-changelog**: For automated changelog generation
- **husky**: For git hooks and pre-commit validation
- **lint-staged**: For running checks on staged files

### Git Hooks

Consider implementing pre-commit hooks that:
- Run code formatting
- Execute relevant tests
- Validate commit message format
- Check for sensitive information

## Summary

Following these git commit instructions ensures:

- Clear project history
- Easy debugging and bisecting
- Automated changelog generation
- Better collaboration between humans and AI
- Consistent code quality
- Traceable changes and decisions

For questions or clarifications about these guidelines, refer to the main development documentation or create an issue for discussion.
