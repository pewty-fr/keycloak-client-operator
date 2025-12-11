# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0](https://github.com/pewty-fr/keycloak-client-operator/compare/operator-v0.1.0...operator-v0.2.0) (2025-12-11)


### Features

* reconcile first version ([b66e1a4](https://github.com/pewty-fr/keycloak-client-operator/commit/b66e1a46e8d1eecb0c93b9ad6b941b5690ae0b88))
* v1 ([7755840](https://github.com/pewty-fr/keycloak-client-operator/commit/77558408d90e30fab3b45b10cfc8d545375b4958))


### Bug Fixes

* login handler ([#3](https://github.com/pewty-fr/keycloak-client-operator/issues/3)) ([2a1d11d](https://github.com/pewty-fr/keycloak-client-operator/commit/2a1d11dcb0365657b58bd3f3403bde76a72c2fed))
* workflow ([5ea2d6e](https://github.com/pewty-fr/keycloak-client-operator/commit/5ea2d6ea988dc4b86c7bab4516cadf3cfc74e9fc))
* workflow ([b794034](https://github.com/pewty-fr/keycloak-client-operator/commit/b7940342a051c65f7401541228ff7004367eca2f))
* workflow ([7fda4d1](https://github.com/pewty-fr/keycloak-client-operator/commit/7fda4d11ba33bea19a9edf53e8ccf5368201d017))

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
