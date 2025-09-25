# AKS-MCP Helm Chart Configuration

This document describes the configuration parameters for the AKS-MCP Helm chart. AKS-MCP is a Model Context Protocol (MCP) server that enables AI assistants to interact with Azure Kubernetes Service (AKS) clusters.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.8+

## Configuration Parameters

### Basic Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.repository` | Container image repository | `ghcr.io/azure/aks-mcp` |
| `image.tag` | Container image tag | `""` (uses chart appVersion) |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `imagePullSecrets` | Image pull secrets | `[]` |
| `nameOverride` | Override chart name | `""` |
| `fullnameOverride` | Override full chart name | `""` |

### Application Settings

| Parameter | Description | Default |
|-----------|-------------|---------|
| `app.transport` | Transport mechanism (stdio, sse, streamable-http) | `streamable-http` |
| `app.port` | Port to listen on | `8000` |
| `app.accessLevel` | Access level (readonly, readwrite, admin) | `readonly` |
| `app.timeout` | Command execution timeout in seconds | `600` |
| `app.verbose` | Enable verbose logging | `false` |
| `app.cache` | Enable cache for better performance | `true` |

### Azure Authentication

| Parameter | Description | Default |
|-----------|-------------|---------|
| `azure.existingSecret` | Use existing secret for Azure credentials | `""` |
| `azure.tenantId` | Azure tenant ID | `""` |
| `azure.clientId` | Azure client ID | `""` |
| `azure.clientSecret` | Azure client secret | `""` |
| `azure.subscriptionId` | Azure subscription ID | `""` |

### OAuth Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `oauth.enabled` | Enable OAuth authentication | `false` |
| `oauth.tenantId` | Azure AD tenant ID for OAuth | `""` |
| `oauth.clientId` | Azure AD client ID for OAuth | `""` |
| `oauth.redirectURIs` | Custom redirect URIs | `[]` |
| `oauth.corsOrigins` | Custom CORS origins | `[]` |

### Ingress Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `ingress.enabled` | Enable ingress for the application | `false` |
| `ingress.ingressClassName` | Ingress class name | `""` |
| `ingress.annotations` | Annotations for the ingress resource | `{}` |
| `ingress.hosts` | List of hosts for the ingress | `[]` |
| `ingress.tls` | TLS configuration for the ingress | `[]` |

### Additional Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.additionalTools` | Additional Kubernetes tools (helm, cilium, hubble) | `[]` |
| `config.allowNamespaces` | Allowed Kubernetes namespaces (empty means all) | `[]` |
| `config.cacheTimeout` | Cache timeout | `"60s"` |
| `kubeconfig.enabled` | Enable kubeconfig secret mount | `false` |
| `kubeconfig.secretName` | Name of kubeconfig secret | `"kubeconfig"` |

### Security Settings

| Parameter | Description | Default |
|-----------|-------------|---------|
| `rbac.create` | Create RBAC resources | `true` |
| `serviceAccount.create` | Create service account | `true` |
| `serviceAccount.automount` | Auto-mount service account token | `true` |
| `serviceAccount.annotations` | Service account annotations | `{}` |
| `serviceAccount.name` | Service account name (auto-generated if empty) | `""` |
| `podSecurityContext.runAsNonRoot` | Run as non-root user | `true` |
| `podSecurityContext.runAsUser` | User ID to run as | `1000` |
| `podSecurityContext.runAsGroup` | Group ID to run as | `1000` |
| `podSecurityContext.fsGroup` | File system group ID | `1000` |
| `securityContext.allowPrivilegeEscalation` | Allow privilege escalation | `false` |
| `securityContext.readOnlyRootFilesystem` | Enable read-only root filesystem | `true` |
| `securityContext.runAsNonRoot` | Run as non-root user | `true` |
| `securityContext.runAsUser` | User ID to run as | `1000` |
| `securityContext.runAsGroup` | Group ID to run as | `1000` |
| `securityContext.capabilities.drop` | Dropped capabilities | `["ALL"]` |

### Service Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `service.type` | Kubernetes service type | `ClusterIP` |
| `service.port` | Service port | `8000` |

### Resource Management

| Parameter | Description | Default |
|-----------|-------------|---------|
| `resources.limits.cpu` | CPU resource limits | `500m` |
| `resources.limits.memory` | Memory resource limits | `512Mi` |
| `resources.requests.cpu` | CPU resource requests | `100m` |
| `resources.requests.memory` | Memory resource requests | `128Mi` |

### Health Checks

| Parameter | Description | Default |
|-----------|-------------|---------|
| `livenessProbe.httpGet.path` | Liveness probe HTTP path | `/health` |
| `livenessProbe.httpGet.port` | Liveness probe HTTP port | `http` |
| `livenessProbe.initialDelaySeconds` | Initial delay for liveness probe | `30` |
| `livenessProbe.periodSeconds` | Period for liveness probe | `10` |
| `livenessProbe.timeoutSeconds` | Timeout for liveness probe | `5` |
| `livenessProbe.failureThreshold` | Failure threshold for liveness probe | `3` |
| `readinessProbe.httpGet.path` | Readiness probe HTTP path | `/health` |
| `readinessProbe.httpGet.port` | Readiness probe HTTP port | `http` |
| `readinessProbe.initialDelaySeconds` | Initial delay for readiness probe | `5` |
| `readinessProbe.periodSeconds` | Period for readiness probe | `5` |
| `readinessProbe.timeoutSeconds` | Timeout for readiness probe | `3` |
| `readinessProbe.failureThreshold` | Failure threshold for readiness probe | `3` |

### Advanced Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `extraVolumes` | Additional volumes to mount | `[]` |
| `extraVolumeMounts` | Additional volume mounts | `[]` |
| `extraEnv` | Additional environment variables | `[]` |
| `podAnnotations` | Annotations for pods | `{}` |
| `nodeSelector` | Node selector for scheduling | `{}` |
| `tolerations` | Tolerations for scheduling | `[]` |
| `affinity` | Affinity rules for scheduling | `{}` |

### Telemetry and Monitoring

| Parameter | Description | Default |
|-----------|-------------|---------|
| `telemetry.otlpEndpoint` | OpenTelemetry OTLP endpoint for traces | `""` |

## Access Levels

The `app.accessLevel` parameter controls what operations AKS-MCP can perform:

- **readonly**: Read-only access to Azure resources and Kubernetes clusters
- **readwrite**: Read and write access to Azure resources and Kubernetes clusters
- **admin**: Full administrative access including destructive operations

## Authentication Methods

AKS-MCP supports multiple Azure authentication methods:

1. **Service Principal**: Using `azure.tenantId`, `azure.clientId`, and `azure.clientSecret`
2. **Managed Identity**: Automatic when running on Azure resources
3. **Azure CLI**: Uses existing Azure CLI authentication
4. **Existing Secret**: Reference pre-created Kubernetes secret with `azure.existingSecret`

## Architecture Notes

- **Single Replica**: The deployment is hardcoded to 1 replica due to OAuth state management requirements
- **Security**: Read-only root filesystem with writable `/tmp` directory for temporary files
- **RBAC**: Dynamic permissions based on access level configuration
- **Configuration**: All application settings are passed via command-line arguments