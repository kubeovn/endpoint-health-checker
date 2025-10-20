# Contributing to Endpoint Health Checker

Thank you for your interest in contributing to Endpoint Health Checker! This document provides guidelines and information for contributors.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Pull Request Process](#pull-request-process)
- [Code Style Guidelines](#code-style-guidelines)
- [Testing Requirements](#testing-requirements)
- [Documentation](#documentation)
- [Issue Reporting](#issue-reporting)
- [Community](#community)

## Getting Started

### Prerequisites

- Go 1.19 or later
- Docker
- Helm 3.x
- kubectl
- Git

### Setup Development Environment

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/yourusername/endpoint-health-checker.git
   cd endpoint-health-checker
   ```
3. Add the upstream repository:
   ```bash
   git remote add upstream https://github.com/originalowner/endpoint-health-checker.git
   ```
4. Install dependencies:
   ```bash
   go mod download
   ```
5. Run tests to ensure everything is working:
   ```bash
   make test
   ```

## Development Workflow

1. Create a new branch from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   ```
2. Make your changes following the code style guidelines
3. Add tests for your changes
4. Run all tests and ensure they pass:
   ```bash
   make test
   make lint
   ```
5. Commit your changes with a clear commit message:
   ```bash
   git commit -m "feat: add new health check feature"
   ```
6. Push to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

## Pull Request Process

1. Ensure your PR description clearly describes the problem and solution
2. Link to relevant issues in your PR description
3. Ensure all CI checks pass
4. Request review from at least one maintainer
5. Address any feedback from reviewers

### PR Title Format

Use conventional commits format:
- `feat:` for new features
- `fix:` for bug fixes
- `docs:` for documentation changes
- `style:` for code style changes
- `refactor:` for refactoring
- `test:` for adding tests
- `chore:` for maintenance tasks

Examples:
- `feat: add support for UDP health checks`
- `fix: resolve memory leak in health checker`
- `docs: update README with deployment instructions`

## Code Style Guidelines

### Go Code Style

- Follow standard Go formatting (`gofmt`)
- Use `golangci-lint` for linting
- Keep functions small and focused
- Add comments for exported functions and complex logic
- Use meaningful variable and function names

### Example Code Style

```go
// CheckHealth performs a health check on the given endpoint
func (h *HealthChecker) CheckHealth(ctx context.Context, endpoint string) error {
    if endpoint == "" {
        return errors.New("endpoint cannot be empty")
    }

    // Implementation here
    return nil
}
```

## Testing Requirements

### Unit Tests

- All new features must include unit tests
- Aim for at least 80% code coverage
- Use table-driven tests for multiple scenarios

```go
func TestCheckHealth(t *testing.T) {
    tests := []struct {
        name     string
        endpoint string
        wantErr  bool
    }{
        {
            name:     "valid endpoint",
            endpoint: "http://example.com:8080",
            wantErr:  false,
        },
        {
            name:     "empty endpoint",
            endpoint: "",
            wantErr:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Integration Tests

- Integration tests should be added for complex scenarios
- Use the `test/` directory for integration tests

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run integration tests
make test-integration
```

## Documentation

- Update relevant documentation when adding features
- Add code comments for exported functions
- Update the README if you change user-facing behavior
- Document any new configuration options

## Issue Reporting

### Bug Reports

When reporting bugs, please include:

1. **Environment Information**:
   - Kubernetes version
   - Endpoint Health Checker version
   - Operating system
   - Go version

2. **Description**:
   - Clear description of the problem
   - Steps to reproduce
   - Expected behavior
   - Actual behavior

3. **Logs**:
   - Relevant logs from the health checker
   - Kubernetes events if applicable

### Feature Requests

For feature requests, please include:

1. **Use Case**: Why is this feature needed?
2. **Proposed Solution**: How do you envision this working?
3. **Alternatives Considered**: What other approaches did you consider?

## Community

- Be respectful and constructive in all interactions
- Follow our [Code of Conduct](CODE_OF_CONDUCT.md)
- Join our discussions for questions and ideas

## Getting Help

- Check existing issues and documentation
- Join our community discussions
- Feel free to ask questions in issues or discussions

## Release Process

Releases are handled by maintainers following semantic versioning. For more information, see our [release documentation](RELEASE.md).

## Maintainer Guidelines

Maintainers should:

- Review PRs in a timely manner
- Provide constructive feedback
- Help guide contributors
- Follow the code review checklist
- Ensure CI/CD processes are working properly

Thank you for contributing to Endpoint Health Checker! 🎉