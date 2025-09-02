// Owner: JeelRupapara (zeelrupapara@gmail.com)
package app

import (
	"blueprint/config"
	"blueprint/handler"
	"blueprint/pkg/cache"
	"blueprint/pkg/logger"
	"blueprint/pkg/redis"
	"blueprint/pkg/db"
	"blueprint/pkg/i18n"
	
	"context"
	"fmt"	
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "blueprint/proto/blueprint"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/grpc-ecosystem/go-grpc-prometheus"
)

var (
	service = "forex-platform-blueprint"
	version = "v2.0.0"
)

func Start() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.NewConfig()		
	
	fmt.Printf("Starting %s %s\n", service, version)
	
	log, err := logger.NewLogger(cfg)
	if err != nil {
		panic(fmt.Errorf("failed to initialize logger: %w", err))
	}
	defer log.Flush()

	local, err := i18n.New(cfg, "en-US", "el-GR", "zh-CN")
	if err != nil {
		log.Errorf("failed to init i18n package: %v", err)
	}
	
	log.Infof("Starting service: %s@%s", service, version)
	
	lis, err := net.Listen("tcp", ":" + cfg.GRPC.Port)
	if err != nil {
		log.Fatalf("failed to listen on port %s: %v", cfg.GRPC.Port, err)
	}

	kaep := keepalive.EnforcementPolicy{
		MinTime:             5 * time.Second,
		PermitWithoutStream: true,
	}

	kasp := keepalive.ServerParameters{
		MaxConnectionIdle:     15 * time.Second,
		MaxConnectionAge:      30 * time.Second,
		MaxConnectionAgeGrace: 5 * time.Second,
		Time:                  5 * time.Second,
		Timeout:               1 * time.Second,
	}

	// Recovery options for panic handling
	recoveryOpts := []recovery.Option{
		recovery.WithRecoveryHandler(func(p interface{}) error {
			log.Errorf("panic recovered: %v", p)
			return fmt.Errorf("internal server error")
		}),
	}

	s := grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(kaep),
		grpc.KeepaliveParams(kasp),
		grpc.ChainUnaryInterceptor(
			recovery.UnaryServerInterceptor(recoveryOpts...),
			grpc_prometheus.UnaryServerInterceptor,
		),
		grpc.ChainStreamInterceptor(
			recovery.StreamServerInterceptor(recoveryOpts...),
			grpc_prometheus.StreamServerInterceptor,
		),
	)

	reflection.Register(s)

	log.Infof("gRPC server listening on %v", lis.Addr())
	
	redisClient, err := redis.NewRedisClient(cfg)
	if err != nil {
		log.Fatalf("Error connecting to Redis at %v: %v", cfg.Redis.RedisAddr, err)
	}
	defer redisClient.Close()

	log.Infof("Connected to Redis at %s", cfg.Redis.RedisAddr)

	cacheClient := cache.NewCache(redisClient.GetClient())
	if cacheClient == nil {
		panic("Could not initialize cache client")
	}

	dbSess, err := db.NewMysqlDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbSess.Close()

	log.Info("Connected to MySQL database")

	if err := db.Migrate(cfg); err != nil {
		log.Warnf("Migration failed: %v", err)
	}

	blueprintHandler := handler.NewBlueprint(local, log, cacheClient, dbSess.DB)

	pb.RegisterBlueprintServer(s, blueprintHandler)

	grpc_prometheus.Register(s)

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	select {
	case sig := <-quit:
		log.Infof("Received shutdown signal: %v", sig)
	case <-ctx.Done():
		log.Info("Context cancelled")
	}
	
	log.Info("Shutting down gracefully...")
	
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	stopped := make(chan struct{})
	go func() {
		s.GracefulStop()
		close(stopped)
	}()

	select {
	case <-shutdownCtx.Done():
		log.Warn("Graceful shutdown timed out, forcing stop")
		s.Stop()
	case <-stopped:
		log.Info("Server stopped gracefully")
	}

	log.Info("Shutdown complete")
}