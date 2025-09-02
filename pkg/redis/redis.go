// Owner: JeelRupapara (zeelrupapara@gmail.com)
package redis

import (
	"context"
	"fmt"
	"sync"
	"time"

	"blueprint/config"

	"github.com/redis/go-redis/v9"
)

const (
	defaultPoolSize     = 100
	defaultMinIdleConns = 10
	defaultPoolTimeout  = 30 * time.Second
	defaultDialTimeout  = 5 * time.Second
	defaultReadTimeout  = 3 * time.Second
	defaultWriteTimeout = 3 * time.Second
	defaultMaxRetries   = 3
	// v9 optimized buffer sizes
	defaultReadBufferSize  = 32 * 1024 // 32KB
	defaultWriteBufferSize = 32 * 1024 // 32KB
)

type RedisClient struct {
	client *redis.Client
	config *config.Config
	mu     sync.RWMutex
	stats  RedisStats
}

type RedisStats struct {
	TotalCommands   uint64
	FailedCommands  uint64
	ConnectedClients uint32
	BlockedClients  uint32
}

type RedisOptions struct {
	Addr            string
	Password        string
	DB              int
	PoolSize        int
	MinIdleConns    int
	PoolTimeout     time.Duration
	DialTimeout     time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	MaxRetries      int
	MaxRetryBackoff time.Duration
	MinRetryBackoff time.Duration
	ReadBufferSize  int
	WriteBufferSize int
	Protocol        int // RESP protocol version (2 or 3)
}

func NewRedisClient(cfg *config.Config) (*RedisClient, error) {
	opts := buildOptions(cfg)
	return NewRedisClientWithOptions(cfg, opts)
}

func NewRedisClientWithOptions(cfg *config.Config, opts RedisOptions) (*RedisClient, error) {
	if opts.Addr == "" {
		opts.Addr = "localhost:6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr:            opts.Addr,
		Password:        opts.Password,
		DB:              opts.DB,
		PoolSize:        opts.PoolSize,
		MinIdleConns:    opts.MinIdleConns,
		PoolTimeout:     opts.PoolTimeout,
		DialTimeout:     opts.DialTimeout,
		ReadTimeout:     opts.ReadTimeout,
		WriteTimeout:    opts.WriteTimeout,
		MaxRetries:      opts.MaxRetries,
		MaxRetryBackoff: opts.MaxRetryBackoff,
		MinRetryBackoff: opts.MinRetryBackoff,
		Protocol:        opts.Protocol,
		OnConnect: func(ctx context.Context, cn *redis.Conn) error {
			return cn.Ping(ctx).Err()
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Check server info for version compatibility
	info, err := client.Info(ctx, "server").Result()
	if err == nil {
		fmt.Printf("Connected to Redis server: %s\n", info[:50])
	}

	rc := &RedisClient{
		client: client,
		config: cfg,
	}

	return rc, nil
}

func buildOptions(cfg *config.Config) RedisOptions {
	opts := RedisOptions{
		Addr:            cfg.Redis.RedisAddr,
		PoolSize:        cfg.Redis.PoolSize,
		MinIdleConns:    cfg.Redis.MinIdleConn,
		PoolTimeout:     time.Duration(cfg.Redis.PoolTimeout) * time.Second,
		DialTimeout:     defaultDialTimeout,
		ReadTimeout:     defaultReadTimeout,
		WriteTimeout:    defaultWriteTimeout,
		MaxRetries:      defaultMaxRetries,
		ReadBufferSize:  defaultReadBufferSize,
		WriteBufferSize: defaultWriteBufferSize,
		Protocol:        3, // Use RESP3 by default for better performance
	}

	if opts.Addr == "" {
		opts.Addr = "localhost:6379"
	}

	if opts.PoolSize == 0 {
		opts.PoolSize = defaultPoolSize
	}

	if opts.MinIdleConns == 0 {
		opts.MinIdleConns = defaultMinIdleConns
	}

	if opts.PoolTimeout == 0 {
		opts.PoolTimeout = defaultPoolTimeout
	}

	// Note: Buffer size configuration removed in v9
	// Redis v9 handles buffer optimization internally

	return opts
}

func (r *RedisClient) GetClient() *redis.Client {
	return r.client
}

func (r *RedisClient) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

func (r *RedisClient) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *RedisClient) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := r.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("Redis health check failed: %w", err)
	}

	// Additional health checks for v9
	stats := r.client.PoolStats()
	if stats.Misses > 0 && float64(stats.Misses)/float64(stats.Hits+stats.Misses) > 0.5 {
		return fmt.Errorf("Redis pool miss rate too high: %.2f%%", 
			float64(stats.Misses)/float64(stats.Hits+stats.Misses)*100)
	}

	return nil
}

func (r *RedisClient) GetPoolStats() *redis.PoolStats {
	if r.client != nil {
		return r.client.PoolStats()
	}
	return nil
}

// Pipeline returns a pipeline for batch operations
func (r *RedisClient) Pipeline() redis.Pipeliner {
	return r.client.Pipeline()
}

// TxPipeline returns a transactional pipeline
func (r *RedisClient) TxPipeline() redis.Pipeliner {
	return r.client.TxPipeline()
}

// Subscribe to channels using pub/sub
func (r *RedisClient) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return r.client.Subscribe(ctx, channels...)
}

// Publish message to a channel
func (r *RedisClient) Publish(ctx context.Context, channel string, message interface{}) error {
	return r.client.Publish(ctx, channel, message).Err()
}

// Watch implements optimistic locking
func (r *RedisClient) Watch(ctx context.Context, fn func(*redis.Tx) error, keys ...string) error {
	return r.client.Watch(ctx, fn, keys...)
}

func (r *RedisClient) FlushDB(ctx context.Context) error {
	return r.client.FlushDB(ctx).Err()
}

func (r *RedisClient) FlushAll(ctx context.Context) error {
	return r.client.FlushAll(ctx).Err()
}

// UpdateStats fetches current Redis statistics
func (r *RedisClient) UpdateStats() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	info, err := r.client.Info(ctx, "stats").Result()
	if err != nil {
		return fmt.Errorf("failed to get Redis stats: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	
	fmt.Sscanf(info, "total_commands_processed:%d", &r.stats.TotalCommands)
	
	return nil
}

func (r *RedisClient) GetStats() RedisStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.stats
}

// Common operations with improved error handling

func (r *RedisClient) SetWithExpiry(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("key %s not found", key)
	}
	return val, err
}

func (r *RedisClient) GetBytes(ctx context.Context, key string) ([]byte, error) {
	return r.client.Get(ctx, key).Bytes()
}

func (r *RedisClient) Delete(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

func (r *RedisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Exists(ctx, keys...).Result()
}

// Scan iterates over keys matching a pattern
func (r *RedisClient) Scan(ctx context.Context, cursor uint64, match string, count int64) ([]string, uint64, error) {
	keys, cursor, err := r.client.Scan(ctx, cursor, match, count).Result()
	return keys, cursor, err
}

// SetNX sets a key only if it doesn't exist (useful for distributed locks)
func (r *RedisClient) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return r.client.SetNX(ctx, key, value, expiration).Result()
}

// TTL returns the remaining time to live of a key
func (r *RedisClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.client.TTL(ctx, key).Result()
}