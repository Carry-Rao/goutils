# Contributing to Goutils

Thank you for considering contributing to Goutils. This document outlines the conventions and guidelines to help you get started.

## Table of Contents

- [Commit Message Convention](#commit-message-convention)
- [Branch Strategy](#branch-strategy)
- [Code Style](#code-style)
- [Pull Request Process](#pull-request-process)
- [Getting Help](#getting-help)

## Commit Message Convention

This project enforces a **structured commit message format**. Every commit message **must** follow the pattern below:

```
Operation: <action>[, Module: <module>][: <description>]
```

### Format Breakdown

| Part | Required | Description |
|------|----------|-------------|
| `Operation:` | Yes | Fixed prefix indicating this is an operation commit |
| `<action>` | Yes | The type of action being performed (see below) |
| `, Module:` | No | The module being affected (e.g. `database`, `log`, `docs`) |
| `: <description>` | No | A brief description of the change |

### Actions

Use one of the following action keywords:

| Action | When to Use |
|--------|-------------|
| `Docs` | Adding or updating documentation |
| `Fix` | Bug fixes |
| `Format` | Code formatting, whitespace, or style changes |
| `Automatic` | Automated tooling changes (CI, workflows, etc.) |
| `Merge` | Merge new module |
| `Update` | Updating existing features or dependencies |

### Examples

```
Operation: Format.
Operation: Automatic: Update github workflow for code check, use ast.
Operation: Automatic: Add github workflow for code check.
Operation: Fix, Module: Log: Compare and correct log levels
Operation: Docs: Add documents of database.
Operation: Merge, Module: database: Provide a unity init function.
Operation: Update, Module: init: Update go.mod and go.sum.
Operation: Merge, Module: database/memory: Merge New Type of database: memory.
```

### Commit Body (Optional)

If additional context is needed, add a blank line after the subject line and write the body. There is no strict formatting requirement for the body, but keep it concise and informative.

## Branch Strategy

- **`master`** (or **`main`**) — the primary development branch. All pull requests should target this branch.
- Feature branches should be created from `master` and use descriptive names, e.g.:
  - `feature/add-mongodb-support`
  - `fix/log-buffer-flush`
  - `docs/update-readme`

## Code Style

This project uses **Go** and follows the [official Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) guidelines. Additionally, the CI pipeline performs static analysis with the following rules:

### Function Length

- Functions must be between **4 and 50 lines** (inclusive).
- A function shorter than 4 lines will trigger a CI warning.
- A function longer than 50 lines will trigger a CI warning.

### Error Handling

- **Do not** ignore returned errors using blank identifiers (`_`) unless the return type is from `fmt` package.
- Example of what to avoid:
  ```go
  // BAD: ignoring a non-fmt error
  _ = db.Close()
  ```
- Example of what is acceptable:
  ```go
  // OK: fmt.Fprint returns (int, error) and is safe to ignore
  _, _ = fmt.Fprint(w, "hello")
  ```

### General Go Conventions

- Run `go fmt` before committing to ensure consistent formatting.
- Use meaningful variable and function names.
- Keep packages focused and single-purpose.
- Write unit tests for new functionality.
- Document exported functions, types, and constants with Go-style comments.

## Pull Request Process

1. Fork the repository and create a feature branch from `master`.
2. Make your changes following the code style guidelines above.
3. Write clear commit messages following the commit message convention.
4. Ensure all existing tests pass and add tests for new functionality.
5. Run `go fmt ./...` to format your code.
6. Submit a pull request against the `master` branch.
7. In the PR description, include:
   - A summary of the changes
   - The motivation for the change
   - Any relevant issue numbers

## Getting Help

If you have questions or need help, feel free to open an issue on GitHub.