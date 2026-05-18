package ctx

type ContextKey string

// AzureTokenKey is the context key for storing Azure tokens extracted from HTTP headers.
// This is the name of the HTTP header, not a hardcoded credential.
// #nosec G101
const AzureTokenKey ContextKey = "X-Azure-Token"

// AzureClusterTokenKey is the context key for the AKS-scoped token used as the Kubernetes cluster token
// in RunCommand requests against AAD-enabled clusters (audience: 6dae42f8-4368-4678-94ff-3960e28e3630).
// #nosec G101
const AzureClusterTokenKey ContextKey = "X-Azure-Cluster-Token"
