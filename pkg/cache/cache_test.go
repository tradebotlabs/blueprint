package cache

import (
	"context"
	"testing"
	"time"

	"blueprint/config"
	"blueprint/pkg/redis"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnect(t *testing.T) {
	cfg := config.NewConfig()

	redisClient, err := redis.NewRedisClient(cfg)
	if err != nil {
		t.Skipf("Skipping test - Redis not available: %v", err)
		return
	}
	defer redisClient.Close()

	c := NewCache(redisClient.GetClient())

	ctx := context.Background()
	
	// Test setting a string value
	testKey := "test:key"
	testValue := "test-value"
	
	err = c.Set(ctx, testKey, testValue)
	require.NoError(t, err, "Failed to set cache value")

	// Test getting the value back
	var retrievedValue string
	err = c.Get(ctx, testKey, &retrievedValue)
	require.NoError(t, err, "Failed to get cache value")
	
	assert.Equal(t, testValue, retrievedValue, "Retrieved value doesn't match")
	
	t.Logf("Successfully stored and retrieved: %v", retrievedValue)
	
	// Cleanup
	err = c.Delete(ctx, testKey)
	assert.NoError(t, err, "Failed to delete test key")
}

func TestCacheWithTTL(t *testing.T) {
	cfg := config.NewConfig()

	redisClient, err := redis.NewRedisClient(cfg)
	if err != nil {
		t.Skipf("Skipping test - Redis not available: %v", err)
		return
	}
	defer redisClient.Close()

	c := NewCache(redisClient.GetClient())
	ctx := context.Background()
	
	testKey := "test:ttl:key"
	testValue := map[string]string{
		"field1": "value1",
		"field2": "value2",
	}
	
	// Set with 2 second TTL
	err = c.SetWithTTL(ctx, testKey, testValue, 2*time.Second)
	require.NoError(t, err, "Failed to set cache value with TTL")
	
	// Verify it exists
	exists, err := c.Exists(ctx, testKey)
	require.NoError(t, err)
	assert.True(t, exists, "Key should exist immediately after setting")
	
	// Wait for expiration
	time.Sleep(3 * time.Second)
	
	// Verify it's gone
	var result map[string]string
	err = c.Get(ctx, testKey, &result)
	assert.Error(t, err, "Key should have expired")
}

func TestCacheStats(t *testing.T) {
	cfg := config.NewConfig()

	redisClient, err := redis.NewRedisClient(cfg)
	if err != nil {
		t.Skipf("Skipping test - Redis not available: %v", err)
		return
	}
	defer redisClient.Close()

	c := NewCache(redisClient.GetClient())
	ctx := context.Background()
	
	// Reset stats
	c.ResetStats()
	
	// Perform operations
	testKey := "test:stats:key"
	testValue := "stats-value"
	
	// This should increment sets
	err = c.Set(ctx, testKey, testValue)
	require.NoError(t, err)
	
	// This should increment hits
	var value string
	err = c.Get(ctx, testKey, &value)
	require.NoError(t, err)
	
	// This should increment misses
	err = c.Get(ctx, "non-existent-key", &value)
	assert.Error(t, err)
	
	// This should increment deletes
	err = c.Delete(ctx, testKey)
	require.NoError(t, err)
	
	// Check stats
	stats := c.GetStats()
	assert.Equal(t, uint64(1), stats.Sets, "Should have 1 set operation")
	assert.Equal(t, uint64(1), stats.Hits, "Should have 1 cache hit")
	assert.Equal(t, uint64(1), stats.Misses, "Should have 1 cache miss")
	assert.Equal(t, uint64(1), stats.Deletes, "Should have 1 delete operation")
	
	t.Logf("Cache stats: Sets=%d, Hits=%d, Misses=%d, Deletes=%d", 
		stats.Sets, stats.Hits, stats.Misses, stats.Deletes)
}

func TestCacheBatchDelete(t *testing.T) {
	cfg := config.NewConfig()

	redisClient, err := redis.NewRedisClient(cfg)
	if err != nil {
		t.Skipf("Skipping test - Redis not available: %v", err)
		return
	}
	defer redisClient.Close()

	c := NewCache(redisClient.GetClient())
	ctx := context.Background()
	
	// Set multiple keys
	keys := []string{"test:batch:1", "test:batch:2", "test:batch:3"}
	for i, key := range keys {
		err = c.Set(ctx, key, i)
		require.NoError(t, err)
	}
	
	// Delete all at once
	err = c.Delete(ctx, keys...)
	require.NoError(t, err)
	
	// Verify all are deleted
	for _, key := range keys {
		exists, err := c.Exists(ctx, key)
		require.NoError(t, err)
		assert.False(t, exists, "Key %s should be deleted", key)
	}
}