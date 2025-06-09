package cache

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrContextIsNil = errors.New("context is nil")

// KeyExistsError: a custom error type that is returned when an attempt is
// made to add a key to the cache that already exists.
type DuplicateKeyError struct {
	Key any
}

func (e *DuplicateKeyError) Error() string {
	return fmt.Sprintf("key %v already exists", e.Key)
}

// VCache: a struct that implements a simple in-memory key-value cache with eviction.
type MemCache struct {
	timeToLiveInSeconds   int64
	evictionCheckInterval time.Duration
	store                 map[any]*cacheValue

	// one mutex for each cache
	mutex *sync.RWMutex
}

// cacheValue: a struct that contains a value and its expiration time.
type cacheValue struct {
	value          any
	expirationTime time.Time
}

// New: a function that creates and returns a new Cache instance.
// Assumes duration is the same of each member of the cache
func New(checkInterval time.Duration, timeRecordEvict time.Duration) *MemCache {
	return &MemCache{
		mutex:                 &sync.RWMutex{},
		store:                 make(map[any]*cacheValue),
		evictionCheckInterval: checkInterval,
		timeToLiveInSeconds:   int64(timeRecordEvict.Seconds()),
	}
}

// add a key to the cache using default expire tike
func (c *MemCache) Add(key any, value any) error {
	err := c.AddWithExpireTime(key, value, c.timeToLiveInSeconds)
	if err != nil {
		return err
	}
	return nil
}

// add a key to the cache with explicit expire time.
func (c *MemCache) AddWithExpireTime(key any, value any, timeToLiveInSeconds int64) error {
	// lock the cache for an add
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// check for duplicate keys and add an error
	if _, ok := c.store[key]; ok {
		return &DuplicateKeyError{Key: key}
	}

	currentTime := time.Now()
	expireTime := currentTime.Add(time.Duration(timeToLiveInSeconds) * time.Second)
	c.store[key] = &cacheValue{
		value:          value,
		expirationTime: expireTime,
	}

	return nil
}

// Get: a method that retrieves a value from the cache by its key.
// Returns the value and a boolean indicating if the key was found.
func (c *MemCache) Get(key any) (any, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	val, foundKey := c.store[key]

	if foundKey {
		// make sure we have a non-expired cached item
		if time.Now().After(val.expirationTime) {
			return nil, false
		}
		return val.value, foundKey
	}
	return nil, false
}

// Delete: a method that deletes a key-value pair from the cache.
func (c *MemCache) Delete(key any) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.store, key)
}

// Evict: a method that evicts expired key-value pairs from the cache.
func (c *MemCache) Evict() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var evictedItems []interface{}

	for key, val := range c.store {
		if time.Now().After(val.expirationTime) {
			evictedItems = append(evictedItems, key)
		}
	}

	for _, key := range evictedItems {
		delete(c.store, key)
	}
}

// StartEvict: a method that starts the eviction process in a separate goroutine.
// It stops when the context passed as an argument is done.
func (c *MemCache) StartEvictionChecks(context context.Context) error {
	if context == nil {
		return ErrContextIsNil
	}

	trigger := time.NewTicker(c.evictionCheckInterval)
	defer trigger.Stop()

	go func() {
		for {
			select {
			case <-trigger.C:
				c.Evict()
			case <-context.Done():
				return
			}
		}
	}()

	return nil
}
