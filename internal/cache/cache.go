package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Cache represents an in-memory cache with file persistence
type Cache struct {
	data     map[string]cacheEntry
	filePath string
	mu       sync.RWMutex
}

type cacheEntry struct {
	Value      interface{} `json:"value"`
	Expiration int64       `json:"expiration"`
}

// New creates a new Cache instance
func New(cacheDir string) (*Cache, error) {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	filePath := filepath.Join(cacheDir, "featherhead.cache")
	cache := &Cache{
		data:     make(map[string]cacheEntry),
		filePath: filePath,
	}

	// Load existing cache if it exists
	if err := cache.load(); err != nil {
		return nil, err
	}

	return cache, nil
}

// Get retrieves a value from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, found := c.data[key]
	if !found {
		return nil, false
	}

	if entry.Expiration > 0 && entry.Expiration < time.Now().UnixNano() {
		return nil, false
	}

	return entry.Value, true
}

// Set adds a value to the cache
func (c *Cache) Set(key string, value interface{}, duration time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var exp int64
	if duration > 0 {
		exp = time.Now().Add(duration).UnixNano()
	}

	c.data[key] = cacheEntry{
		Value:      value,
		Expiration: exp,
	}

	return c.save()
}

// load reads the cache from disk
func (c *Cache) load() error {
	data, err := os.ReadFile(c.filePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	return json.Unmarshal(data, &c.data)
}

// save writes the cache to disk
func (c *Cache) save() error {
	data, err := json.Marshal(c.data)
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %w", err)
	}

	return os.WriteFile(c.filePath, data, 0644)
}

// Clear removes all entries from the cache
func (c *Cache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]cacheEntry)
	return c.save()
}
