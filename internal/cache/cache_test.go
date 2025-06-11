package cache

import (
	"os"
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "featherhead-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a new cache instance
	cache, err := New(tempDir)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Test setting and getting a value
	testKey := "test-key"
	testValue := "test-value"

	err = cache.Set(testKey, testValue, time.Hour)
	if err != nil {
		t.Errorf("Failed to set cache value: %v", err)
	}

	value, found := cache.Get(testKey)
	if !found {
		t.Error("Failed to get cached value")
	}

	if strValue, ok := value.(string); !ok || strValue != testValue {
		t.Errorf("Got wrong value: %v, want: %v", value, testValue)
	}

	// Test expiration
	expiredKey := "expired-key"
	err = cache.Set(expiredKey, "expired-value", time.Millisecond)
	if err != nil {
		t.Errorf("Failed to set expired value: %v", err)
	}

	time.Sleep(time.Millisecond * 2)
	_, found = cache.Get(expiredKey)
	if found {
		t.Error("Expired value should not be found")
	}

	// Test clearing cache
	err = cache.Clear()
	if err != nil {
		t.Errorf("Failed to clear cache: %v", err)
	}

	_, found = cache.Get(testKey)
	if found {
		t.Error("Value should not be found after clearing cache")
	}
}
