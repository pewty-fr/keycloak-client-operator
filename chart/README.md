# Keycloak Client Operator Helm Chart

This Helm chart deploys the Keycloak Client Operator on a Kubernetes cluster.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- A running Keycloak instance

## Installing the Chart

To install the chart with the release name `my-release`:

```bash
helm install my-release ./chart \
  --set keycloak.url=https://keycloak.example.com \
  --set keycloak.user=admin \
  --set keycloak.password=admin
```

Or using an existing secret:

```bash
kubectl create secret generic keycloak-credentials \
  --from-literal=KEYCLOAK_URL=https://keycloak.example.com \
  --from-literal=KEYCLOAK_USER=admin \
  --from-literal=KEYCLOAK_PASSWORD=admin

helm install my-release ./chart \
  --set keycloak.existingSecret=keycloak-credentials
```

## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```bash
helm delete my-release
```

## Configuration

The following table lists the configurable parameters of the Keycloak Client Operator chart and their default values.

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Image repository | `controller` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `image.tag` | Image tag | `latest` |
| `keycloak.url` | Keycloak server URL | `""` |
| `keycloak.user` | Keycloak admin username | `""` |
| `keycloak.password` | Keycloak admin password | `""` |
| `keycloak.existingSecret` | Name of existing secret for credentials | `""` |
| `serviceAccount.create` | Create service account | `true` |
| `serviceAccount.name` | Service account name | `""` |
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `128Mi` |
| `resources.requests.cpu` | CPU request | `10m` |
| `resources.requests.memory` | Memory request | `64Mi` |
| `leaderElection.enabled` | Enable leader election | `true` |
| `metrics.enabled` | Enable metrics service | `true` |
| `rbac.create` | Create RBAC resources | `true` |
| `crds.install` | Install CRDs | `true` |

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example:

```bash
helm install my-release ./chart \
  --set replicaCount=2 \
  --set keycloak.url=https://keycloak.example.com
```

Alternatively, a YAML file that specifies the values for the parameters can be provided while installing the chart:

```bash
helm install my-release ./chart -f values.yaml
```

## Example Custom Values

```yaml
replicaCount: 2

image:
  repository: ghcr.io/pewty/keycloak-client-operator
  tag: v0.1.0

keycloak:
  existingSecret: keycloak-credentials

resources:
  limits:
    cpu: 1000m
    memory: 256Mi
  requests:
    cpu: 100m
    memory: 128Mi

nodeSelector:
  kubernetes.io/os: linux

tolerations:
  - key: "workload"
    operator: "Equal"
    value: "operators"
    effect: "NoSchedule"
```

## Using the Operator

After installing the operator, you can create Keycloak clients by applying custom resources:

```yaml
apiVersion: keycloak.pewty.fr/v1
kind: Client
metadata:
  name: example-oidc-client
  namespace: default
spec:
  realm: master
  client:
    clientId: example-client
    enabled: true
    protocol: openid-connect
    publicClient: false
    redirectUris:
      - https://app.example.com/callback
    webOrigins:
      - https://app.example.com
```

## Metrics

The operator exposes metrics on port 8443 (HTTPS). The metrics service is created automatically when `metrics.enabled` is set to `true`.

## Health Checks

The operator exposes health check endpoints:
- Liveness: `/healthz` on port 8081
- Readiness: `/readyz` on port 8081
