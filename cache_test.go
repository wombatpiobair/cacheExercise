package cache

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var cacheMap map[string]any

func init() {
	cacheMap = map[string]any{
		"celery":  "seett",
		"banana":  "peel",
		"cherry":  "pit",
		"apricot": "pit",
		"peach":   "pit",
		"apple":   "seed",
		"grape":   "pit",
	}
}

func TestCacheHappyPathWithExpire(t *testing.T) {
	// Create a context for the cache eviction
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a cache with a check ticker of 1 second and a record eviction of 5 seconds
	c := New(time.Second, 5*time.Second)

	// Start eviction routine
	c.StartEvictionChecks(ctx)

	// Add key-value pairs to the cache
	for key, value := range cacheMap {
		err := c.Add(key, value)
		if err != nil {
			t.Error(err)
		}
	}

	for key, value := range cacheMap {
		fetchedValue, found := c.Get(key)
		assert.True(t, found, "key not found")
		assert.Equal(t, fetchedValue, value, "value doesn't match")
	}

	// Wait for 7 seconds for keys to be evicted
	time.Sleep(7 * time.Second)

	// Try to get key1 after eviction
	for key := range cacheMap {
		val, found := c.Get(key)
		assert.False(t, found, "key found after exipriattion")
		assert.Nil(t, val, "value is not nil")
		break
	}

}

func TestCacheMultiThreadWrite(t *testing.T) {
	// Create a context for the cache eviction
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a cache with a check ticker of 1 second and a record eviction of 5 seconds
	c := New(time.Second, 5*time.Second)

	// Start eviction routine
	c.StartEvictionChecks(ctx)

	for key, value := range cacheMap {
		println("storing key: %s, value: %s", key, value)
		go func() {
			err := writeWorker(c, key, value)
			if err != nil {
				t.Error(err)
			}
		}()
	}

	time.Sleep(2 * time.Second)

	for key, value := range cacheMap {
		val, found := c.Get(key)
		assert.True(t, found, "key not found %s", key)
		assert.Equal(t, val, value, "value does not match")
	}

}

func TestCacheMultiThreadRead(t *testing.T) {
	// Create a context for the cache eviction
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a cache with a check ticker of 1 second and a record eviction of 5 seconds
	c := New(time.Second, 5*time.Second)

	// Start eviction routine
	c.StartEvictionChecks(ctx)

	// Add key-value pairs to the cache
	for key, value := range cacheMap {
		err := c.Add(key, value)
		if err != nil {
			t.Error(err)
		}
	}

	// Add key-value pairs to the cache
	for key, value := range cacheMap {
		println("fetching key: %s", key)
		go func() {
			readWorker(t, c, key, value)
		}()
	}

}

func writeWorker(c *MemCache, key string, value any) error {
	err := c.Add(key, value)
	if err != nil {
		return err
	}
	return nil
}

func readWorker(t *testing.T, c *MemCache, key string, value any) {
	fetchedValue, found := c.Get(key)
	assert.True(t, found, "key not found in cache")
	assert.Equal(t, fetchedValue, value, "value does not match")
}
