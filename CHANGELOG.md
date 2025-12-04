# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Initial release of Keycloak Client Operator
- Support for managing Keycloak OIDC and SAML clients via Kubernetes CRDs
- Automatic synchronization of client configurations with Keycloak
- Finalizer support for proper cleanup on deletion
- Status conditions for tracking reconciliation state
- Helm chart for easy deployment
- Comprehensive unit tests with 32.7% code coverage
- Support for protocol mappers and client attributes
- Leader election for high availability deployments
