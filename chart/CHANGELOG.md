# Changelog

## [Unreleased]

### Features

- Initial Helm chart for Keycloak Client Operator deployment
- Support for production and development configurations
- Configurable resource limits and requests
- RBAC configuration with cluster-scoped permissions
- Leader election for high availability
- Metrics service on port 8443
- Health probes (liveness, readiness, startup)
- Security context (non-root, read-only root filesystem)
