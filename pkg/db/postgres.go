// Owner: JeelRupapara (zeelrupapara@gmail.com)
package db

import (
	"blueprint/config"
	model "blueprint/model/blueprint"
	"context"
	"database/sql"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

const (
	maxIdleConns    = 10
	maxOpenConns    = 100
	connMaxLifetime = time.Hour
	connMaxIdleTime = time.Minute * 10
)

type PostgresDB struct {
	DB     *gorm.DB
	sqlDB  *sql.DB
	config *config.Config
}

type DBOptions struct {
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	LogLevel        logger.LogLevel
}

func NewPostgresDB(cfg *config.Config) (*PostgresDB, error) {
	return NewPostgresDBWithOptions(cfg, DBOptions{
		MaxIdleConns:    maxIdleConns,
		MaxOpenConns:    maxOpenConns,
		ConnMaxLifetime: connMaxLifetime,
		ConnMaxIdleTime: connMaxIdleTime,
		LogLevel:        logger.Error,
	})
}

func NewPostgresDBWithOptions(cfg *config.Config, opts DBOptions) (*PostgresDB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		cfg.Postgres.PostgresHost,
		cfg.Postgres.PostgresUser,
		cfg.Postgres.PostgresPassword,
		cfg.Postgres.PostgresDBName,
		cfg.Postgres.PostgresPort,
	)

	gormConfig := &gorm.Config{
		PrepareStmt:                              true,
		DisableForeignKeyConstraintWhenMigrating: true,
		QueryFields:                              true,
		Logger:                                   logger.Default.LogMode(opts.LogLevel),
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "platform_",
			SingularTable: true,
			NoLowerCase:   false,
		},
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying SQL database: %w", err)
	}

	sqlDB.SetMaxIdleConns(opts.MaxIdleConns)
	sqlDB.SetMaxOpenConns(opts.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(opts.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(opts.ConnMaxIdleTime)

	postgresDB := &PostgresDB{
		DB:     db,
		sqlDB:  sqlDB,
		config: cfg,
	}

	if err := postgresDB.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return postgresDB, nil
}

func (m *PostgresDB) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	return m.sqlDB.PingContext(ctx)
}

func (m *PostgresDB) Close() error {
	if m.sqlDB != nil {
		return m.sqlDB.Close()
	}
	return nil
}

func (m *PostgresDB) Stats() sql.DBStats {
	if m.sqlDB != nil {
		return m.sqlDB.Stats()
	}
	return sql.DBStats{}
}

func (m *PostgresDB) BeginTx(ctx context.Context, opts *sql.TxOptions) *gorm.DB {
	return m.DB.WithContext(ctx).Begin(opts)
}

func (m *PostgresDB) WithContext(ctx context.Context) *gorm.DB {
	return m.DB.WithContext(ctx)
}

func (m *PostgresDB) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	var result int
	if err := m.DB.WithContext(ctx).Raw("SELECT 1").Scan(&result).Error; err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("unexpected health check result: %d", result)
	}

	return nil
}

func Migrate(cfg *config.Config) error {
	db, err := NewPostgresDB(cfg)
	if err != nil {
		return fmt.Errorf("failed to connect for migration: %w", err)
	}
	defer db.Close()

	if err := db.DB.AutoMigrate(&model.MyModel{}); err != nil {
		return fmt.Errorf("failed to auto-migrate: %w", err)
	}

	return nil
}

func (m *PostgresDB) EnableSlowQueryLog(threshold time.Duration) {
	m.DB.Config.Logger = m.DB.Config.Logger.LogMode(logger.Info)
	m.DB = m.DB.Session(&gorm.Session{
		Logger: logger.Default.LogMode(logger.Info),
	})
}