# Release Process

This document explains how to create releases for the Crib SDK using our automated GitHub Actions workflow powered by Release Please.

## Overview

The release process uses [Release Please](https://github.com/googleapis/release-please) to automate:

- Analyzing conventional commit messages
- Creating release pull requests with changelogs
- Automatically releasing when release PRs are merged
- Maintaining semantic versioning based on commit types
- Building and publishing release artifacts

## How Release Please Works

Release Please follows a **completely automated** approach:

1. **Every push to main** triggers the Release Please workflow
2. **Analyzes conventional commits** since the last release
3. **Creates a release PR** when releasable changes are found
4. **Merging the release PR** automatically creates the GitHub release
5. **Builds and publishes** binaries and artifacts

## Creating a Release

### Prerequisites

1. You must have **write access** to the repository
2. All changes must be **merged to the `main` branch**
3. Commits must follow **conventional commit format**
4. All tests must be **passing** on the main branch

### Step-by-Step Process

#### 1. Make Conventional Commits

Every commit that should trigger a release must follow the conventional commit format:

```markdown
<type>(<scope>): <description>

[optional body]

[optional footer]
```

#### 2. Push to Main Branch

```bash
git push origin main
```

#### 3. Wait for Release PR

Release Please automatically:

- Analyzes your commits
- Creates a release PR if changes warrant a release
- Generates changelog entries
- Calculates the appropriate version bump

#### 4. Review and Merge Release PR

1. **Review the auto-generated release PR**
2. **Check the changelog** and version bump
3. **Merge the PR** to trigger the actual release

#### 5. Automatic Release

When you merge the release PR:

- GitHub release is automatically created
- Binaries are built for all platforms
- Artifacts are uploaded to the release
- Tags are created with proper versioning

## Conventional Commit Format

### Required Format

```markdown
<type>(<scope>): <description>

[optional body]

[optional footer]
```

### Commit Types

| Type | Description | Version Bump | Example |
|------|-------------|--------------|---------|
| `feat` | New features | Minor | `feat: add new anvil component` |
| `fix` | Bug fixes | Patch | `fix: resolve memory leak in helmchart` |
| `perf` | Performance improvements | Patch | `perf: optimize plan execution` |
| `refactor` | Code refactoring | Patch | `refactor: simplify command structure` |
| `docs` | Documentation | Patch | `docs: update installation guide` |
| `test` | Tests | No release | `test: add unit tests for component` |
| `ci` | CI/CD changes | No release | `ci: update workflow permissions` |
| `chore` | Maintenance | No release | `chore: update dependencies` |
| `style` | Code style | No release | `style: fix formatting` |
| `build` | Build system | No release | `build: update go version` |

### Breaking Changes

To trigger a **major version** bump, use:

```bash
# Option 1: Add ! after type
git commit -m "feat!: remove deprecated API"

# Option 2: Add BREAKING CHANGE footer
git commit -m "feat: redesign configuration

BREAKING CHANGE: configuration format has changed"
```

### Examples

```bash
# Patch release (1.0.0 → 1.0.1)
git commit -m "fix: resolve nil pointer in validator"
git commit -m "docs: update README with new examples"

# Minor release (1.0.0 → 1.1.0)
git commit -m "feat: add new postgres component"
git commit -m "feat(cli): add --debug flag to commands"

# Major release (1.0.0 → 2.0.0)
git commit -m "feat!: remove legacy API endpoints"
git commit -m "feat: redesign component interface

BREAKING CHANGE: Component interface now requires Run() method"
```

## Version Numbering

Release Please automatically follows [Semantic Versioning](https://semver.org/):

- **PATCH** (1.0.0 → 1.0.1): `fix`, `perf`, `refactor`, `docs`
- **MINOR** (1.0.0 → 1.1.0): `feat`
- **MAJOR** (1.0.0 → 2.0.0): Any type with `!` or `BREAKING CHANGE`

### Pre-1.0.0 Versioning

Before reaching v1.0.0:
- **Breaking changes** bump minor version (0.1.0 → 0.2.0)
- **Features** bump minor version (0.1.0 → 0.2.0)
- **Fixes** bump patch version (0.1.0 → 0.1.1)

## What Happens During a Release

### When You Push to Main
1. **Release Please workflow** runs automatically
2. **Analyzes commits** since last release
3. **Creates/updates release PR** if needed

### When You Merge Release PR
1. **GitHub release** is created automatically
2. **Binaries are built** for multiple platforms
3. **Tests run** to ensure quality
4. **Artifacts uploaded** to release
5. **Tags created** with semantic version

## Supported Platforms

The release process creates binaries for:

| OS      | Architectures        |
|---------|---------------------|
| Linux   | 386, amd64, arm, arm64 |
| macOS   | amd64, arm64        |
| Windows | 386, amd64, arm64   |

## Installation Methods

After a release is created, users can install cribctl using:

### 1. Go Install (Recommended)

```bash
go install github.com/smartcontractkit/crib-sdk/cmd/cribctl@latest
# Or specific version:
go install github.com/smartcontractkit/crib-sdk/cmd/cribctl@v1.0.0
```

### 2. Direct Binary Download

Download the appropriate binary from the [releases page](https://github.com/smartcontractkit/crib-sdk/releases).

### 3. Using as a Go Module

```go
import "github.com/smartcontractkit/crib-sdk"
```

## Configuration

Release Please is configured through two files:

### `.release-please-config.json`

Contains the main configuration including:

- Release type (Go)
- Package name
- Changelog sections
- Extra files to update

### `.release-please-manifest.json`

Tracks the current version of each package.

## Troubleshooting

### Common Issues

#### "No Release PR Created"

- **Check commit format**: Ensure commits follow conventional format
- **Check commit types**: Only `feat`, `fix`, `perf`, `refactor`, `docs` trigger releases
- **Check if already released**: Release Please won't create PR if no new releasable commits

#### "Wrong Version Bump"

- **Check commit prefixes**: `feat` = minor, `fix` = patch, `feat!` = major
- **Check for breaking changes**: Look for `!` or `BREAKING CHANGE` in commits
- **Review commit history**: Version is calculated from all commits since last release

#### "Release PR Won't Merge"

- **Check for conflicts**: Resolve any merge conflicts in the PR
- **Check branch protection**: Ensure release PRs can bypass required reviews if needed
- **Check permissions**: Ensure workflow has write permissions

#### "Tests Failing in Release"

- **Run tests locally**: `go test ./...`
- **Check GitHub Actions logs**: Review detailed error messages
- **Fix issues and push**: New commits will update the release PR

### Manual Intervention

#### Force a Release

If you need to force a release without conventional commits:

```bash
# Create an empty commit with conventional format
git commit --allow-empty -m "feat: trigger release"
git push origin main
```

#### Skip a Release

Add `Release-As: x.y.z` to commit body to specify exact version:

```bash
git commit -m "feat: add feature

Release-As: 2.0.0"
```

#### Update Release PR

Release PRs automatically update when you push new commits to main. The PR will:

- Recalculate version based on all commits
- Update changelog with new entries
- Adjust version bump if needed

## Best Practices

### Commit Messages

1. **Be descriptive**: Clear, concise descriptions of changes
2. **Use scopes**: Group related changes with scopes like `(cli)`, `(core)`, `(docs)`
3. **Follow conventions**: Stick to conventional commit format consistently
4. **One concern per commit**: Keep commits focused on single changes

### Release Management

1. **Review release PRs**: Always review generated changelogs and version bumps
2. **Test before merging**: Ensure all changes work as expected
3. **Coordinate breaking changes**: Plan major version bumps with team
4. **Keep dependencies updated**: Regular maintenance keeps releases smooth

### Team Workflow

1. **Train team on conventional commits**: Ensure everyone follows the format
2. **Use PR templates**: Include commit format reminders in PR templates
3. **Set up commit linting**: Use tools like commitlint to enforce format
4. **Review commit history**: Clean up commit messages before merging

## Post-Release Checklist

After a release is created:

1. ✅ **Verify the release**: Check the [releases page](https://github.com/smartcontractkit/crib-sdk/releases)
2. ✅ **Test installation**: Try installing the new version
3. ✅ **Update documentation**: Update any version references in docs
4. ✅ **Announce**: Share the release with the team (Slack, etc.)
5. ✅ **Check binaries**: Verify all platform binaries work correctly

## Emergency Procedures

### Rollback a Release

If you need to rollback a release:

1. **Mark as draft**: Edit the release on GitHub and mark as draft
2. **Create hotfix release**: Make a fix commit and let Release Please create new release
3. **Update manifest**: Manually edit `.release-please-manifest.json` if needed

### Delete a Release

If you need to completely remove a release:

1. **Delete the release**: Go to releases, click on the release, and delete it
2. **Delete the tag**:

   ```bash
   git tag -d v1.0.0
   git push origin :refs/tags/v1.0.0
   ```

3. **Update manifest**: Reset version in `.release-please-manifest.json`

### Fix Version Issues

If Release Please calculates wrong version:

1. **Check commit history**: Review commits since last release
2. **Use Release-As**: Add `Release-As: x.y.z` to commit body
3. **Manually edit manifest**: Update `.release-please-manifest.json` (advanced)

## Configuration Files

### Key Files

- `.github/workflows/release-please.yaml`: GitHub Actions workflow
- `.release-please-config.json`: Release Please configuration
- `.release-please-manifest.json`: Version tracking
- `CHANGELOG.md`: Auto-maintained changelog

For advanced configuration changes, refer to the [Release Please documentation](https://github.com/googleapis/release-please).
