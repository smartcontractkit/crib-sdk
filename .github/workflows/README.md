# GitHub Actions Workflows

This directory contains GitHub Actions workflows for the crib-sdk repository.

## Workflows

### commit-message-enforcement.yaml

**Purpose**: Enforces conventional commit message format for human authors after merge to main.

**Trigger**: Push to `main` branch

**What it does**:

1. Checks if the commit author is a human (not a bot)
2. Validates the commit message against conventional commit format
3. Creates an issue and assigns it to the author if the format is invalid

**Why this exists**:

- Release Please requires conventional commit messages to generate proper changelogs and version bumps
- While pre-commit hooks can enforce this locally, GitHub's "Squash & Merge" feature allows overriding commit messages
- This workflow provides post-merge enforcement specifically for human authors

**Conventional Commit Format**:

```markdown
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

**Valid Types**:

- `feat`: A new feature
- `fix`: A bug fix  
- `docs`: Documentation only changes
- `style`: Changes that do not affect the meaning of the code
- `refactor`: A code change that neither fixes a bug nor adds a feature
- `perf`: A code change that improves performance
- `test`: Adding missing tests or correcting existing tests
- `build`: Changes that affect the build system or external dependencies
- `ci`: Changes to CI configuration files and scripts
- `chore`: Other changes that don't modify src or test files
- `revert`: Reverts a previous commit

**Bot Detection**: The workflow automatically skips validation for commits from:

- GitHub bots (`noreply@github.com`, `bot@github.com`)
- Dependabot
- Renovate
- Release Please
- Any author with "[bot]" in their name

### Other Workflows

- `release-please.yaml`: Handles automated releases using Release Please
- `pr-main.yaml`: Main PR workflow for testing and validation
- `pr-pre-commit.yaml`: Pre-commit checks for PRs
- `receive-pr.yaml`: Handles incoming PRs
- `auto-ready.yaml`: Automatically marks PRs as ready for review
