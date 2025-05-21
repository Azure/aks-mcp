// Package azure provides Azure SDK integration for AKS MCP server.
package azure

import (
	"sync"
	"time"
)

// Cache is a simple in-memory cache for Azure resources.
type AzureCache struct {
	data           map[string]cacheItem
	mu             sync.RWMutex
	defaultTimeout time.Duration
}

// cacheItem represents a cached resource with expiration time.
type cacheItem struct {
	value      interface{}
	expiration time.Time
}

// NewAzureCache creates a new cache with the default timeout.
func NewAzureCache() *AzureCache {
	return &AzureCache{
		data:           make(map[string]cacheItem),
		defaultTimeout: 5 * time.Minute, // Default 5 minute cache timeout
	}
}

// Get retrieves a value from the cache.
// Returns the value and true if the item exists and hasn't expired.
// Returns nil and false otherwise.
func (c *AzureCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.data[key]
	if !found {
		return nil, false
	}

	// Check if the item has expired
	if time.Now().After(item.expiration) {
		return nil, false
	}

	return item.value, true
}

// Set adds or updates a value in the cache with the default expiration time.
func (c *AzureCache) Set(key string, value interface{}) {
	c.SetWithExpiration(key, value, c.defaultTimeout)
}

// SetWithExpiration adds or updates a value in the cache with a custom expiration time.
func (c *AzureCache) SetWithExpiration(key string, value interface{}, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = cacheItem{
		value:      value,
		expiration: time.Now().Add(duration),
	}
}

// Delete removes a value from the cache.
func (c *AzureCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.data, key)
}

// Clear removes all values from the cache.
func (c *AzureCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]cacheItem)
}
