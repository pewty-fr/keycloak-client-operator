# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.0](https://github.com/pewty-fr/keycloak-client-operator/compare/operator-v0.3.1...operator-v0.4.0) (2025-12-18)


### Features

* dedicated secret ([#11](https://github.com/pewty-fr/keycloak-client-operator/issues/11)) ([fd77401](https://github.com/pewty-fr/keycloak-client-operator/commit/fd7740181e5dd6409ca2efd3afde2a55cad72f4d))

## [0.3.1](https://github.com/pewty-fr/keycloak-client-operator/compare/operator-v0.3.0...operator-v0.3.1) (2025-12-11)


### Bug Fixes

* login realm & loger ([#8](https://github.com/pewty-fr/keycloak-client-operator/issues/8)) ([cb85d33](https://github.com/pewty-fr/keycloak-client-operator/commit/cb85d336374aa6ec730f2185a96ef8cb8febe8b7))

## [0.3.0](https://github.com/pewty-fr/keycloak-client-operator/compare/operator-v0.2.0...operator-v0.3.0) (2025-12-11)


### Features

* use zerolog for json format ([#4](https://github.com/pewty-fr/keycloak-client-operator/issues/4)) ([b8700f7](https://github.com/pewty-fr/keycloak-client-operator/commit/b8700f7618e69173c02f71635daad9c7ecfd643c))


### Bug Fixes

* deployment ([#5](https://github.com/pewty-fr/keycloak-client-operator/issues/5)) ([f0c5067](https://github.com/pewty-fr/keycloak-client-operator/commit/f0c50676d4b6764e279d43233bf0912bdc6aa7c4))

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
