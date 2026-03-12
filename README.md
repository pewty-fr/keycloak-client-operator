<div align="center">

# 🔐 Keycloak Client Operator

**Kubernetes operator for managing Keycloak clients declaratively**

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/pewty/keycloak-client-operator)](https://goreportcard.com/report/github.com/pewty/keycloak-client-operator)
[![Docker Pulls](https://img.shields.io/docker/pulls/ghcr.io/pewty/keycloak-client-operator)](https://github.com/pewty/keycloak-client-operator/pkgs/container/keycloak-client-operator)
[![Release](https://img.shields.io/github/v/release/pewty/keycloak-client-operator)](https://github.com/pewty/keycloak-client-operator/releases/latest)
[![CI](https://github.com/pewty/keycloak-client-operator/actions/workflows/ci.yaml/badge.svg)](https://github.com/pewty/keycloak-client-operator/actions/workflows/ci.yaml)

[Features](#features) •
[Installation](#installation) •
[Usage](#usage) •
[Configuration](#configuration) •
[Contributing](#contributing)

</div>

---

## 📋 Overview

Keycloak Client Operator enables you to manage Keycloak clients (OAuth2/OIDC applications) as Kubernetes custom resources. Define your clients in YAML, apply them to your cluster, and let the operator handle the synchronization with Keycloak.

**Why use this operator?**
- 🎯 **GitOps-ready**: Manage Keycloak clients alongside your application deployments
- 🔄 **Declarative**: Define desired state; the operator ensures it's maintained
- 🚀 **Production-ready**: Multi-architecture support, comprehensive RBAC, leader election
- 📦 **Easy deployment**: Available as Helm chart or kubectl manifests
- 🔐 **Secure**: Non-root containers, read-only filesystem, distroless images

## ✨ Features

- ✅ Full Keycloak client lifecycle management (create, update, delete)
- ✅ Support for client authentication (confidential, public, bearer-only)
- ✅ Protocol mappers configuration
- ✅ Authorization settings and policies
- ✅ Multi-realm support
- ✅ Leader election for high availability
- ✅ Metrics endpoint for monitoring
- ✅ Multi-architecture images (amd64, arm64)

## 🧪 Keycloak Compatibility Matrix

The e2e test suite runs against multiple Keycloak versions on every pull request. Results below reflect the latest CI run.

| Keycloak Version | Status | Notes |
|-----------------|--------|-------|
| 25.0.6 | [![E2E 25.0.6](https://github.com/pewty-fr/keycloak-client-operator/actions/workflows/test-e2e.yml/badge.svg)](https://github.com/pewty-fr/keycloak-client-operator/actions/workflows/test-e2e.yml) | Older minor, client credentials flow |
| 26.0.7 | [![E2E 26.0.7](https://github.com/pewty-fr/keycloak-client-operator/actions/workflows/test-e2e.yml/badge.svg)](https://github.com/pewty-fr/keycloak-client-operator/actions/workflows/test-e2e.yml) | Current LTS series start |
| 26.2.4 | [![E2E 26.2.4](https://github.com/pewty-fr/keycloak-client-operator/actions/workflows/test-e2e.yml/badge.svg)](https://github.com/pewty-fr/keycloak-client-operator/actions/workflows/test-e2e.yml) | Stable patch |
| 26.5.5 | [![E2E 26.5.5](https://github.com/pewty-fr/keycloak-client-operator/actions/workflows/test-e2e.yml/badge.svg)](https://github.com/pewty-fr/keycloak-client-operator/actions/workflows/test-e2e.yml) | Latest stable |

> The operator uses the Keycloak [client credentials grant](https://www.keycloak.org/docs/latest/server_admin/#_client_credentials) flow for authentication and the Keycloak Admin REST API for client lifecycle management.

## 🚀 Installation

### Prerequisites

- Kubernetes 1.19+
- Keycloak 25+ (see compatibility matrix above)
- Helm 3.0+ (for Helm installation)

### Option 1: Helm (Recommended)

```bash
# Add credentials secret
kubectl create secret generic keycloak-credentials \
  --from-literal=KEYCLOAK_URL=https://keycloak.example.com \
  --from-literal=KEYCLOAK_USER=admin \
  --from-literal=KEYCLOAK_PASSWORD=your-password \
  --from-literal=KEYCLOAK_REALM=master

# Install from OCI registry
helm install keycloak-client-operator \
  oci://ghcr.io/pewty/keycloak-client-operator-chart \
  --version 0.1.0 \
  --set keycloak.existingSecret=keycloak-credentials
```

**With custom values:**

```bash
# Create custom values file
cat <<EOF > values.yaml
replicaCount: 2

resources:
  limits:
    cpu: 500m
    memory: 256Mi
  requests:
    cpu: 100m
    memory: 64Mi

keycloak:
  existingSecret: keycloak-credentials
EOF

# Install with custom values
helm install keycloak-client-operator \
  oci://ghcr.io/pewty/keycloak-client-operator-chart \
  --version 0.1.0 \
  -f values.yaml
```

### Option 2: kubectl

```bash
# Install the operator and CRDs
kubectl apply -f https://github.com/pewty/keycloak-client-operator/releases/latest/download/install.yaml

# Create Keycloak credentials
kubectl create secret generic keycloak-credentials \
  --namespace keycloak-client-operator-system \
  --from-literal=KEYCLOAK_URL=https://keycloak.example.com \
  --from-literal=KEYCLOAK_USER=admin \
  --from-literal=KEYCLOAK_PASSWORD=your-password \
  --from-literal=KEYCLOAK_REALM=master

# Update the deployment to use the secret
kubectl set env deployment/keycloak-client-operator-controller-manager \
  --namespace keycloak-client-operator-system \
  --from=secret/keycloak-credentials
```

### Option 3: From Source

```bash
git clone https://github.com/pewty/keycloak-client-operator.git
cd keycloak-client-operator

# Set environment variables
export KEYCLOAK_URL=https://keycloak.example.com
export KEYCLOAK_USER=admin
export KEYCLOAK_PASSWORD=your-password
export KEYCLOAK_REALM=master

# Install CRDs
make install

# Run locally (for development)
make run

# Or build and deploy to cluster
export KO_DOCKER_REPO=your-registry/keycloak-client-operator
make ko-build
make deploy IMG=your-registry/keycloak-client-operator:latest
```

## 📖 Usage

### Basic Example

Create a simple Keycloak client:

```yaml
apiVersion: keycloak.pewty.fr/v1
kind: Client
metadata:
  name: my-app
  namespace: default
spec:
  realm: master
  secretRef:
    name: "my-secret"
    clientIdKey: "client.id"
    clientSecretKey: "client.secret"
  client:
    enabled: true
    publicClient: false
    standardFlowEnabled: true
    directAccessGrantsEnabled: true
    serviceAccountsEnabled: true
    redirectUris:
      - "https://my-app.example.com/*"
    webOrigins:
      - "https://my-app.example.com"
```

Apply it:

```bash
kubectl apply -f client.yaml
```

### Public Client Example

For a frontend application:

```yaml
apiVersion: keycloak.pewty.fr/v1
kind: Client
metadata:
  name: frontend-app
spec:
  realm: production
  secretRef:
    name: "my-secret"
    clientIdKey: "client.id"
    clientSecretKey: "client.secret"
  client:
    enabled: true
    publicClient: true
    standardFlowEnabled: true
    implicitFlowEnabled: false
    redirectUris:
      - "https://app.example.com/*"
      - "http://localhost:3000/*"
    webOrigins:
      - "+"
```

### Service Account Client

For machine-to-machine communication:

```yaml
apiVersion: keycloak.pewty.fr/v1
kind: Client
metadata:
  name: api-service
spec:
  realm: production
  secretRef:
    name: "my-secret"
    clientIdKey: "client.id"
    clientSecretKey: "client.secret"
  client:
    enabled: true
    publicClient: false
    serviceAccountsEnabled: true
    standardFlowEnabled: false
    directAccessGrantsEnabled: false
```

### Check Status

```bash
# View all clients
kubectl get clients

# Describe a specific client
kubectl describe client my-app

# Check operator logs
kubectl logs -n keycloak-client-operator-system \
  deployment/keycloak-client-operator-controller-manager
```

## ⚙️ Configuration

### Helm Values

Key configuration options:

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of operator replicas | `1` |
| `image.repository` | Container image repository | `ghcr.io/pewty/keycloak-client-operator` |
| `image.tag` | Container image tag | Chart appVersion |
| `keycloak.url` | Keycloak server URL | `""` |
| `keycloak.user` | Keycloak admin username | `""` |
| `keycloak.password` | Keycloak admin password | `""` |
| `keycloak.existingSecret` | Use existing secret for credentials | `""` |
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `128Mi` |
| `resources.requests.cpu` | CPU request | `10m` |
| `resources.requests.memory` | Memory request | `64Mi` |

See [chart/values.yaml](chart/values.yaml) for all available options.

### Environment Variables

The operator can be configured via environment variables:

- `KEYCLOAK_URL`: Keycloak server URL (required)
- `KEYCLOAK_USER`: Admin username (required)
- `KEYCLOAK_PASSWORD`: Admin password (required)
- `KEYCLOAK_REALM`: Keycloak realm for operator authentication (default: `master`, client realms are specified in CRD spec)
- `METRICS_BIND_ADDRESS`: Metrics server address (default: `:8443`)
- `HEALTH_PROBE_BIND_ADDRESS`: Health probe address (default: `:8081`)
- `LEADER_ELECT`: Enable leader election (default: `true`)

## 🔍 Monitoring

The operator exposes Prometheus metrics on port 8443:

```yaml
# ServiceMonitor example
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: keycloak-client-operator
spec:
  selector:
    matchLabels:
      control-plane: controller-manager
  endpoints:
    - port: metrics
      scheme: https
      tlsConfig:
        insecureSkipVerify: true
```

## 🧪 Development

### Running Tests

```bash
# Run unit tests
make test

# Run tests with coverage
make test-coverage

# Run linter
make lint
```

### Building

```bash
# Build binary
make build

# Build and push container image with ko
export KO_DOCKER_REPO=your-registry/keycloak-client-operator
make ko-build

# Build locally for testing (single platform)
make ko-build-local
```

### Local Development

```bash
# Install CRDs
make install

# Run operator locally
export KEYCLOAK_URL=https://keycloak.example.com
export KEYCLOAK_USER=admin
export KEYCLOAK_PASSWORD=password
export KEYCLOAK_REALM=master
make run
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

### Workflow

1. Fork the repository
2. Create your feature branch (`git checkout -b feat/amazing-feature`)
3. Commit your changes using [Conventional Commits](https://www.conventionalcommits.org/)
   - `feat:` for new features
   - `fix:` for bug fixes
   - `docs:` for documentation changes
   - `test:` for test additions/changes
4. Push to the branch (`git push origin feat/amazing-feature`)
5. Open a Pull Request

### Testing PR Changes

Comment `/snapshot docker` or `/snapshot helm` on your PR to build test artifacts.

## 📜 License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

## 🙏 Acknowledgments

Built with:
- [Kubebuilder](https://kubebuilder.io/) - Kubernetes API extension framework
- [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) - Kubernetes controller libraries
- [Ginkgo](https://onsi.github.io/ginkgo/) & [Gomega](https://onsi.github.io/gomega/) - Testing framework

## 📞 Support

- 🐛 [Report a bug](https://github.com/pewty/keycloak-client-operator/issues/new?labels=bug)
- 💡 [Request a feature](https://github.com/pewty/keycloak-client-operator/issues/new?labels=enhancement)
- 💬 [Ask a question](https://github.com/pewty/keycloak-client-operator/discussions)

---

<div align="center">

Made with ❤️ by the Pewty community

⭐ Star us on GitHub — it motivates us a lot!

</div>

