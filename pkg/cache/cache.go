// Owner: JeelRupapara (zeelrupapara@gmail.com)
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/pkg/errors"
)

const (
	defaultPrefix     = "blueprint"
	defaultExpiration = time.Hour
	maxRetries        = 3
	retryDelay        = time.Millisecond * 100
)

type Options struct {
	Prefix     string
	Expiration time.Duration
	MaxRetries int
}

type Cache struct {
	redis      *redis.Client
	prefix     string
	expiration time.Duration
	maxRetries int
	mu         sync.RWMutex
	stats      CacheStats
}

type CacheStats struct {
	Hits   uint64
	Misses uint64
	Sets   uint64
	Deletes uint64
}

func NewCache(redis *redis.Client) *Cache {
	return NewCacheWithOptions(redis, Options{
		Prefix:     defaultPrefix,
		Expiration: defaultExpiration,
		MaxRetries: maxRetries,
	})
}

func NewCacheWithOptions(redis *redis.Client, opts Options) *Cache {
	if opts.Prefix == "" {
		opts.Prefix = defaultPrefix
	}
	if opts.Expiration == 0 {
		opts.Expiration = defaultExpiration
	}
	if opts.MaxRetries == 0 {
		opts.MaxRetries = maxRetries
	}

	return &Cache{
		redis:      redis,
		prefix:     opts.Prefix,
		expiration: opts.Expiration,
		maxRetries: opts.MaxRetries,
	}
}

func (c *Cache) Set(ctx context.Context, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return errors.Wrap(err, "failed to marshal value")
	}

	fullKey := c.createKey(key)
	
	var lastErr error
	for i := 0; i < c.maxRetries; i++ {
		if err := c.redis.SetEx(ctx, fullKey, data, c.expiration).Err(); err == nil {
			c.incrementStats("sets")
			return nil
		} else {
			lastErr = err
			if i < c.maxRetries-1 {
				time.Sleep(retryDelay * time.Duration(i+1))
			}
		}
	}
	
	return errors.Wrapf(lastErr, "failed to set cache key %s after %d retries", fullKey, c.maxRetries)
}

func (c *Cache) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return errors.Wrap(err, "failed to marshal value")
	}

	fullKey := c.createKey(key)
	if err := c.redis.SetEx(ctx, fullKey, data, ttl).Err(); err != nil {
		return errors.Wrapf(err, "failed to set cache key %s", fullKey)
	}
	
	c.incrementStats("sets")
	return nil
}

func (c *Cache) Get(ctx context.Context, key string, dest interface{}) error {
	fullKey := c.createKey(key)
	
	data, err := c.redis.Get(ctx, fullKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			c.incrementStats("misses")
			return errors.Wrapf(err, "key %s not found", fullKey)
		}
		return errors.Wrapf(err, "failed to get cache key %s", fullKey)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return errors.Wrap(err, "failed to unmarshal cached value")
	}

	c.incrementStats("hits")
	return nil
}

func (c *Cache) GetRaw(ctx context.Context, key string) ([]byte, error) {
	fullKey := c.createKey(key)
	
	data, err := c.redis.Get(ctx, fullKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			c.incrementStats("misses")
			return nil, errors.Wrapf(err, "key %s not found", fullKey)
		}
		return nil, errors.Wrapf(err, "failed to get cache key %s", fullKey)
	}

	c.incrementStats("hits")
	return data, nil
}

func (c *Cache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = c.createKey(key)
	}

	deleted, err := c.redis.Del(ctx, fullKeys...).Result()
	if err != nil {
		return errors.Wrapf(err, "failed to delete cache keys")
	}

	c.incrementStats("deletes")
	if deleted != int64(len(keys)) {
		return errors.Errorf("expected to delete %d keys, but deleted %d", len(keys), deleted)
	}

	return nil
}

func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := c.createKey(key)
	
	exists, err := c.redis.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, errors.Wrapf(err, "failed to check existence of key %s", fullKey)
	}

	return exists > 0, nil
}

func (c *Cache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	fullKey := c.createKey(key)
	
	if err := c.redis.Expire(ctx, fullKey, expiration).Err(); err != nil {
		return errors.Wrapf(err, "failed to set expiration for key %s", fullKey)
	}

	return nil
}

func (c *Cache) Flush(ctx context.Context) error {
	pattern := fmt.Sprintf("%s:*", c.prefix)
	
	iter := c.redis.Scan(ctx, 0, pattern, 0).Iterator()
	var keys []string
	
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	
	if err := iter.Err(); err != nil {
		return errors.Wrap(err, "failed to scan keys")
	}

	if len(keys) > 0 {
		if err := c.redis.Del(ctx, keys...).Err(); err != nil {
			return errors.Wrap(err, "failed to delete keys")
		}
	}

	return nil
}

// Pipeline operations for batch processing
func (c *Cache) SetBatch(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	pipe := c.redis.Pipeline()
	
	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal value for key %s", key)
		}
		fullKey := c.createKey(key)
		pipe.SetEx(ctx, fullKey, data, ttl)
	}
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to execute pipeline")
	}
	
	c.incrementStatsBy("sets", uint64(len(items)))
	return nil
}

func (c *Cache) GetBatch(ctx context.Context, keys []string, dest map[string]interface{}) error {
	pipe := c.redis.Pipeline()
	
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = c.createKey(key)
		pipe.Get(ctx, fullKeys[i])
	}
	
	cmds, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return errors.Wrap(err, "failed to execute pipeline")
	}
	
	hits := uint64(0)
	misses := uint64(0)
	
	for i, cmd := range cmds {
		if stringCmd, ok := cmd.(*redis.StringCmd); ok {
			data, err := stringCmd.Bytes()
			if err == nil {
				var value interface{}
				if err := json.Unmarshal(data, &value); err == nil {
					dest[keys[i]] = value
					hits++
				}
			} else if err == redis.Nil {
				misses++
			}
		}
	}
	
	c.incrementStatsBy("hits", hits)
	c.incrementStatsBy("misses", misses)
	
	return nil
}

func (c *Cache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

func (c *Cache) ResetStats() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stats = CacheStats{}
}

func (c *Cache) createKey(key string) string {
	return fmt.Sprintf("%s:%s", c.prefix, key)
}

func (c *Cache) incrementStats(statType string) {
	c.incrementStatsBy(statType, 1)
}

func (c *Cache) incrementStatsBy(statType string, count uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	switch statType {
	case "hits":
		c.stats.Hits += count
	case "misses":
		c.stats.Misses += count
	case "sets":
		c.stats.Sets += count
	case "deletes":
		c.stats.Deletes += count
	}
}

func (c *Cache) Ping(ctx context.Context) error {
	return c.redis.Ping(ctx).Err()
}

// TTL returns the remaining time to live of a key
func (c *Cache) TTL(ctx context.Context, key string) (time.Duration, error) {
	fullKey := c.createKey(key)
	return c.redis.TTL(ctx, fullKey).Result()
}