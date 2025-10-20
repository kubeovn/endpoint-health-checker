# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release of Endpoint Health Checker
- TCP port probing support
- ICMP probing support
- Kubernetes Leader Election integration
- Configurable health check intervals and timeouts
- Parallel health check execution with worker pool
- Pod readiness annotation support
- Pod readinessGates support
- Helm chart for easy deployment
- Comprehensive CI/CD pipeline with GitHub Actions
- E2E testing framework

### Changed
- N/A

### Deprecated
- N/A

### Removed
- N/A

### Fixed
- N/A

### Security
- N/A

## [v0.1.0] - 2024-10-17

### Added
- **Core Functionality**:
  - Fast detection of Kubernetes service endpoint health status
  - Support for both TCP port and ICMP probing methods
  - Automatic fallback from TCP to ICMP based on port availability
  - Configurable retry mechanism for failed health checks
  - Parallel execution with configurable worker pool (default: 10 workers)

- **Kubernetes Integration**:
  - Leader Election using Kubernetes coordination API
  - Pod Ready state management via annotations
  - Pod readinessGates integration for custom readiness conditions
  - ServiceAccount and RBAC configuration
  - Deployment in kube-system namespace

- **Configuration Options**:
  - `HEALTH_CHECK_INTERVAL`: Health check polling interval (default: 1s)
  - `HEALTH_CHECK_TIMEOUT`: Single probe timeout (default: 1s)
  - `HEALTH_CHECK_CONCURRENCY`: Concurrent worker threads (default: 10)
  - `HEALTH_CHECK_RETRY_COUNT`: Health check retry count (default: 10)
  - Leader election configuration with customizable lease duration and retry periods

- **Deployment & Operations**:
  - Helm chart with configurable values
  - Docker container image with multi-architecture support
  - Comprehensive logging with configurable log levels
  - Host network access for accurate endpoint testing
  - Host path volume mounting for log persistence

- **Quality & Testing**:
  - Complete CI/CD pipeline with GitHub Actions
  - Unit tests with high code coverage
  - End-to-end testing with Kind clusters
  - Security scanning with Gosec
  - Code quality checks with golangci-lint
  - Automated Docker image building and pushing

- **Documentation**:
  - Comprehensive README with usage examples
  - API documentation and configuration guide
  - Deployment examples for different use cases
  - Troubleshooting guide and best practices

### Technical Details
- **Language**: Go 1.19+
- **Kubernetes Support**: 1.24+
- **Container Runtime**: Docker
- **Deployment Method**: Helm chart
- **License**: Apache 2.0

### Contributors
- Initial development team

---

## How to Update This Changelog

When adding new entries to this changelog:

1. Use the format `[Unreleased]` for changes not yet released
2. Add entries under appropriate sections: Added, Changed, Deprecated, Removed, Fixed, Security
3. Include the version number and release date when cutting a release
4. Link version numbers to corresponding tags/releases
5. Be specific about what changed and why

Example entry:
```markdown
### Added
- Support for UDP health checking ([#123](https://github.com/user/repo/pull/123))
- Custom timeout configuration per endpoint ([#124](https://github.com/user/repo/pull/124))

### Fixed
- Memory leak in health check worker pool ([#125](https://github.com/user/repo/pull/125))
```

For more information on changelog best practices, see [Keep a Changelog](https://keepachangelog.com/).