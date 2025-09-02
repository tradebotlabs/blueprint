// Owner: JeelRupapara (zeelrupapara@gmail.com)
package db

import (
	"blueprint/config"
	model "blueprint/model/blueprint"
	"context"
	"database/sql"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
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

type MysqlDB struct {
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

func NewMysqlDB(cfg *config.Config) (*MysqlDB, error) {
	return NewMysqlDBWithOptions(cfg, DBOptions{
		MaxIdleConns:    maxIdleConns,
		MaxOpenConns:    maxOpenConns,
		ConnMaxLifetime: connMaxLifetime,
		ConnMaxIdleTime: connMaxIdleTime,
		LogLevel:        logger.Error,
	})
}

func NewMysqlDBWithOptions(cfg *config.Config, opts DBOptions) (*MysqlDB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s&readTimeout=30s&writeTimeout=30s",
		cfg.MySQL.MysqlUser,
		cfg.MySQL.MysqlPassword,
		cfg.MySQL.MysqlHost,
		cfg.MySQL.MysqlPort,
		cfg.MySQL.MysqlDBName,
	)

	gormConfig := &gorm.Config{
		PrepareStmt:                              true,
		DisableForeignKeyConstraintWhenMigrating: true,
		QueryFields:                              true,
		Logger:                                   logger.Default.LogMode(opts.LogLevel),
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "forex_",
			SingularTable: true,
			NoLowerCase:   false,
		},
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
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

	mysqlDB := &MysqlDB{
		DB:     db,
		sqlDB:  sqlDB,
		config: cfg,
	}

	if err := mysqlDB.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return mysqlDB, nil
}

func (m *MysqlDB) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	return m.sqlDB.PingContext(ctx)
}

func (m *MysqlDB) Close() error {
	if m.sqlDB != nil {
		return m.sqlDB.Close()
	}
	return nil
}

func (m *MysqlDB) Stats() sql.DBStats {
	if m.sqlDB != nil {
		return m.sqlDB.Stats()
	}
	return sql.DBStats{}
}

func (m *MysqlDB) BeginTx(ctx context.Context, opts *sql.TxOptions) *gorm.DB {
	return m.DB.WithContext(ctx).Begin(opts)
}

func (m *MysqlDB) WithContext(ctx context.Context) *gorm.DB {
	return m.DB.WithContext(ctx)
}

func (m *MysqlDB) HealthCheck(ctx context.Context) error {
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
	db, err := NewMysqlDB(cfg)
	if err != nil {
		return fmt.Errorf("failed to connect for migration: %w", err)
	}
	defer db.Close()

	if err := db.DB.AutoMigrate(&model.MyModel{}); err != nil {
		return fmt.Errorf("failed to auto-migrate: %w", err)
	}

	return nil
}

func (m *MysqlDB) EnableSlowQueryLog(threshold time.Duration) {
	m.DB.Config.Logger = m.DB.Config.Logger.LogMode(logger.Info)
	m.DB = m.DB.Session(&gorm.Session{
		Logger: logger.Default.LogMode(logger.Info),
	})
}