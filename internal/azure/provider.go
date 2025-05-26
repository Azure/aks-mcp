// Package azure provides functionality for interacting with Azure.
package azure

// Provider defines an interface for accessing Azure resources.
type AzureProvider interface {
	GetClient() *AzureClient
	GetCache() *AzureCache
}

// AzureResourceProvider provides access to Azure resources.
type AzureResourceProvider struct {
	resourceID *AzureResourceID
	client     *AzureClient
	cache      *AzureCache
}

// Compile-time check that AzureResourceProvider implements AzureProvider
var _ AzureProvider = (*AzureResourceProvider)(nil)

// NewAzureResourceProvider creates a new Azure resource provider.
func NewAzureResourceProvider(resourceID *AzureResourceID, client *AzureClient, cache *AzureCache) *AzureResourceProvider {
	return &AzureResourceProvider{
		resourceID: resourceID,
		client:     client,
		cache:      cache,
	}
}

// GetClient returns the Azure client.
func (p *AzureResourceProvider) GetClient() *AzureClient {
	return p.client
}

// GetCache returns the cache.
func (p *AzureResourceProvider) GetCache() *AzureCache {
	return p.cache
}
