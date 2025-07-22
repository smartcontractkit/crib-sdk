module.exports = async ({github, context}) => {
  const issueBody = `## Commit Message Format Violation

A commit was merged to the \`main\` branch that doesn't follow the conventional commit message format required by Release Please.

### Details:
- **Author**: ${context.payload.head_commit.author.name} (${context.payload.head_commit.author.email})
- **Commit SHA**: \`${context.payload.head_commit.id}\`
- **Commit Message**:
\`\`\`
${context.payload.head_commit.message}
\`\`\`

### Expected Format:
Conventional commit messages should follow this pattern:
\`\`\`
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
\`\`\`

### Valid Types:
- \`feat\`: A new feature
- \`fix\`: A bug fix
- \`docs\`: Documentation only changes
- \`style\`: Changes that do not affect the meaning of the code
- \`refactor\`: A code change that neither fixes a bug nor adds a feature
- \`perf\`: A code change that improves performance
- \`test\`: Adding missing tests or correcting existing tests
- \`build\`: Changes that affect the build system or external dependencies
- \`ci\`: Changes to CI configuration files and scripts
- \`chore\`: Other changes that don't modify src or test files
- \`revert\`: Reverts a previous commit

### Action Required:
Please update the commit message to follow the conventional format. You can do this by:
1. Creating a new commit with the correct format, or
2. Using \`git commit --amend\` if this is the most recent commit

### Resources:
- [Conventional Commits Specification](https://www.conventionalcommits.org/)
- [Release Please Documentation](https://github.com/googleapis/release-please)

---
*This issue was automatically created by the commit message enforcement workflow.*`;

  const { data: issue } = await github.rest.issues.create({
    owner: context.repo.owner,
    repo: context.repo.repo,
    title: `🚨 Invalid commit message format on main branch`,
    body: issueBody,
    labels: ['bug', 'documentation', 'enhancement'],
    assignees: [context.payload.head_commit.author.name]
  });

  console.log(`Created issue #${issue.number} for commit message violation`);
};
