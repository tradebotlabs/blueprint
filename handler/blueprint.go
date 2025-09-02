// Owner: JeelRupapara (zeelrupapara@gmail.com)
package handler

import (
	"context"
	"fmt"
	"sync"
	"time"

	pb "blueprint/proto/blueprint"
	"blueprint/pkg/cache"
	"blueprint/pkg/logger"
	"blueprint/pkg/i18n"
	
	"gorm.io/gorm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultTimeout = 30 * time.Second
	maxRetries     = 3
)

type Blueprint struct {
	pb.UnimplementedBlueprintServer
	
	Local       *i18n.Lang
	Log         *logger.Logger
	Cache       *cache.Cache
	DB          *gorm.DB
	
	mu          sync.RWMutex
	metrics     Metrics
	rateLimiter *RateLimiter
}

type Metrics struct {
	TotalRequests   uint64
	SuccessfulCalls uint64
	FailedCalls     uint64
	CacheHits       uint64
	CacheMisses     uint64
	AvgResponseTime time.Duration
}

type RateLimiter struct {
	requests map[string][]time.Time
	mu       sync.Mutex
	limit    int
	window   time.Duration
}

func NewBlueprint(local *i18n.Lang, l *logger.Logger, c *cache.Cache, db *gorm.DB) *Blueprint {
	return &Blueprint{
		Local: local,
		Log:   l,
		Cache: c,
		DB:    db,
		rateLimiter: &RateLimiter{
			requests: make(map[string][]time.Time),
			limit:    100,
			window:   time.Minute,
		},
	}
}

func (b *Blueprint) Call(ctx context.Context, req *pb.CallRequest) (*pb.CallResponse, error) {
	start := time.Now()
	defer func() {
		b.recordMetrics(time.Since(start), nil)
	}()

	if err := b.validateRequest(req); err != nil {
		b.Log.WithError(err).Error("Invalid request")
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if !b.checkRateLimit(ctx, req.Name) {
		return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
	}

	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	b.Log.WithFields(map[string]interface{}{
		"method": "Blueprint.Call",
		"name":   req.Name,
	}).Info("Processing request")

	cacheKey := fmt.Sprintf("call:%s", req.Name)
	
	var cachedResponse pb.CallResponse
	if err := b.Cache.Get(ctx, cacheKey, &cachedResponse); err == nil {
		b.incrementCacheHit()
		b.Log.Debug("Cache hit for key: " + cacheKey)
		return &cachedResponse, nil
	}
	b.incrementCacheMiss()

	response := &pb.CallResponse{
		Msg: fmt.Sprintf("Hello %s from Forex Platform", req.Name),
	}

	if err := b.processBusinessLogic(ctx, req, response); err != nil {
		b.Log.WithError(err).Error("Failed to process business logic")
		return nil, status.Error(codes.Internal, "internal server error")
	}

	if err := b.Cache.SetWithTTL(ctx, cacheKey, response, 5*time.Minute); err != nil {
		b.Log.WithError(err).Warn("Failed to cache response")
	}

	return response, nil
}

func (b *Blueprint) validateRequest(req *pb.CallRequest) error {
	if req == nil {
		return fmt.Errorf("request is nil")
	}
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(req.Name) > 100 {
		return fmt.Errorf("name is too long")
	}
	return nil
}

func (b *Blueprint) processBusinessLogic(ctx context.Context, req *pb.CallRequest, resp *pb.CallResponse) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if b.DB != nil {
		tx := b.DB.WithContext(ctx)
		
		var count int64
		if err := tx.Raw("SELECT COUNT(*) FROM information_schema.tables").Count(&count).Error; err != nil {
			b.Log.WithError(err).Warn("Database query failed")
		}
	}

	return nil
}

func (b *Blueprint) checkRateLimit(ctx context.Context, identifier string) bool {
	b.rateLimiter.mu.Lock()
	defer b.rateLimiter.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-b.rateLimiter.window)

	requests, exists := b.rateLimiter.requests[identifier]
	if !exists {
		b.rateLimiter.requests[identifier] = []time.Time{now}
		return true
	}

	var validRequests []time.Time
	for _, t := range requests {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}

	if len(validRequests) >= b.rateLimiter.limit {
		return false
	}

	validRequests = append(validRequests, now)
	b.rateLimiter.requests[identifier] = validRequests

	return true
}

func (b *Blueprint) recordMetrics(duration time.Duration, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.metrics.TotalRequests++
	if err == nil {
		b.metrics.SuccessfulCalls++
	} else {
		b.metrics.FailedCalls++
	}

	if b.metrics.AvgResponseTime == 0 {
		b.metrics.AvgResponseTime = duration
	} else {
		b.metrics.AvgResponseTime = (b.metrics.AvgResponseTime + duration) / 2
	}
}

func (b *Blueprint) incrementCacheHit() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.metrics.CacheHits++
}

func (b *Blueprint) incrementCacheMiss() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.metrics.CacheMisses++
}

func (b *Blueprint) GetMetrics() Metrics {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.metrics
}

func (b *Blueprint) ResetMetrics() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.metrics = Metrics{}
}

func (b *Blueprint) HealthCheck(ctx context.Context) error {
	if b.DB != nil {
		if err := b.DB.WithContext(ctx).Exec("SELECT 1").Error; err != nil {
			return fmt.Errorf("database connection failed: %w", err)
		}
	}

	if b.Cache != nil {
		if err := b.Cache.Ping(ctx); err != nil {
			return fmt.Errorf("cache connection failed: %w", err)
		}
	}

	return nil
}