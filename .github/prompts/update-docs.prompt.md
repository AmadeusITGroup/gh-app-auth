---
mode: agent
description: "Update documentation for gh-app-auth"
---

# Update Documentation

Update or create documentation for gh-app-auth features.

## Documentation Structure

```
gh-app-auth/
├── README.md                    # Overview, quick start, command reference
├── CONTRIBUTING.md              # Development setup, guidelines
├── CHANGELOG.md                 # Version history
├── docs/
│   ├── installation.md          # Complete setup guide
│   ├── configuration.md         # Config file reference
│   ├── security.md              # Security best practices
│   ├── troubleshooting.md       # Problem solving
│   ├── ci-cd-guide.md           # CI/CD integration
│   ├── architecture.md          # Technical design (contributors)
│   └── testing.md               # Testing guide (contributors)
└── .github/
    ├── SECURITY.md              # Security policy
    └── ISSUE_TEMPLATE/          # Issue templates
```

## Writing Guidelines

### Style

- Use clear, concise language
- Include working code examples
- Use generic placeholders (not internal references)
- Link to related documentation

### Code Examples

- All examples must be copy-pasteable and work
- Use realistic but generic values
- Show both simple and advanced usage
- Include expected output where helpful

### Format

- Use Markdown with clear headings
- Include table of contents for long documents
- Use tables for structured data
- Use admonitions for warnings/tips

## Common Updates

### Adding a New Feature

1. Update README.md with brief mention
2. Add detailed guide to relevant doc
3. Update command reference if new CLI command
4. Add troubleshooting section if needed

### Fixing Inaccuracy

1. Identify all locations with incorrect info
2. Update with correct information
3. Verify code examples still work
4. Check for broken links

### Improving Clarity

1. Add more examples
2. Break long sections into subsections
3. Add visual aids (diagrams, tables)
4. Cross-reference related docs

## Example Templates

### Command Documentation

```markdown
## `command-name`

Brief description.

### Usage

\`\`\`bash
gh app-auth command-name [flags]
\`\`\`

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--flag` | Description | `value` |

### Examples

\`\`\`bash
# Basic usage
gh app-auth command-name

# With options
gh app-auth command-name --flag value
\`\`\`
```

### Troubleshooting Section

```markdown
### Issue: Brief description

**Symptoms:**
- What the user sees

**Cause:**
Why this happens

**Solution:**
\`\`\`bash
# Commands to fix
\`\`\`
```

## Verification

- [ ] All code examples work
- [ ] Links are not broken
- [ ] Spelling/grammar checked
- [ ] Consistent with other docs
- [ ] No sensitive/internal information
