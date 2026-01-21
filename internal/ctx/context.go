package ctx

type ContextKey string

// AzureTokenKey is the context key for storing Azure tokens extracted from HTTP headers.
// This is the name of the HTTP header, not a hardcoded credential.
// #nosec G101
const AzureTokenKey ContextKey = "X-Azure-Token"
