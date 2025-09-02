// Owner: JeelRupapara (zeelrupapara@gmail.com)
package logger

import (
	"blueprint/config"
	"context"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Logger struct {
	*zap.SugaredLogger
	atomicLevel zap.AtomicLevel
	config      *config.Config
	mu          sync.RWMutex
	fields      map[string]any
}

type LoggerOptions struct {
	Level          string
	OutputPath     string
	MaxSize        int
	MaxBackups     int
	MaxAge         int
	Compress       bool
	DisableCaller  bool
	DisableStacktrace bool
	Sampling       bool
}

var (
	defaultOptions = LoggerOptions{
		Level:          "info",
		OutputPath:     "../logs/blueprint.log",
		MaxSize:        100,
		MaxBackups:     30,
		MaxAge:         30,
		Compress:       true,
		DisableCaller:  false,
		DisableStacktrace: true,
		Sampling:       true,
	}
)

func NewLogger(cfg *config.Config) (*Logger, error) {
	opts := buildLoggerOptions(cfg)
	return NewLoggerWithOptions(cfg, opts)
}

func NewLoggerWithOptions(cfg *config.Config, opts LoggerOptions) (*Logger, error) {
	atomicLevel := zap.NewAtomicLevel()
	
	if err := atomicLevel.UnmarshalText([]byte(opts.Level)); err != nil {
		atomicLevel.SetLevel(zapcore.InfoLevel)
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	fileWriter := &lumberjack.Logger{
		Filename:   opts.OutputPath,
		MaxSize:    opts.MaxSize,
		MaxBackups: opts.MaxBackups,
		MaxAge:     opts.MaxAge,
		Compress:   opts.Compress,
	}

	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
	fileEncoder := zapcore.NewJSONEncoder(encoderConfig)

	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, zapcore.AddSync(fileWriter), atomicLevel),
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), atomicLevel),
	)

	if opts.Sampling {
		core = zapcore.NewSamplerWithOptions(
			core,
			time.Second,
			100,
			10,
		)
	}

	zapLogger := zap.New(
		core,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

	if opts.DisableCaller {
		zapLogger = zapLogger.WithOptions(zap.WithCaller(false))
	}

	if opts.DisableStacktrace {
		zapLogger = zapLogger.WithOptions(zap.AddStacktrace(zapcore.DPanicLevel))
	}

	return &Logger{
		SugaredLogger: zapLogger.Sugar(),
		atomicLevel:   atomicLevel,
		config:        cfg,
		fields:        make(map[string]interface{}),
	}, nil
}

func buildLoggerOptions(cfg *config.Config) LoggerOptions {
	opts := defaultOptions
	
	if cfg.Logger.LogFile != "" {
		opts.OutputPath = cfg.Logger.LogFile
	}
	
	return opts
}

func (l *Logger) SetLevel(level string) error {
	return l.atomicLevel.UnmarshalText([]byte(level))
}

func (l *Logger) GetLevel() string {
	return l.atomicLevel.String()
}

func (l *Logger) WithContext(ctx context.Context) *Logger {
	newLogger := &Logger{
		SugaredLogger: l.SugaredLogger,
		atomicLevel:   l.atomicLevel,
		config:        l.config,
		fields:        make(map[string]interface{}),
	}
	
	if traceID := ctx.Value("trace_id"); traceID != nil {
		newLogger = newLogger.WithField("trace_id", traceID)
	}
	
	if userID := ctx.Value("user_id"); userID != nil {
		newLogger = newLogger.WithField("user_id", userID)
	}
	
	return newLogger
}

func (l *Logger) WithField(key string, value interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	l.fields[key] = value
	
	return &Logger{
		SugaredLogger: l.SugaredLogger.With(key, value),
		atomicLevel:   l.atomicLevel,
		config:        l.config,
		fields:        l.fields,
	}
}

func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	var args []interface{}
	for k, v := range fields {
		l.fields[k] = v
		args = append(args, k, v)
	}
	
	return &Logger{
		SugaredLogger: l.SugaredLogger.With(args...),
		atomicLevel:   l.atomicLevel,
		config:        l.config,
		fields:        l.fields,
	}
}

func (l *Logger) WithError(err error) *Logger {
	return l.WithField("error", err.Error())
}

func (l *Logger) GetFields() map[string]interface{} {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	fields := make(map[string]interface{})
	for k, v := range l.fields {
		fields[k] = v
	}
	
	return fields
}

func (l *Logger) Flush() error {
	return l.Sync()
}

func (l *Logger) Close() error {
	return l.Sync()
}

func (l *Logger) LogRequest(method, path string, statusCode int, duration time.Duration) {
	l.WithFields(map[string]interface{}{
		"method":      method,
		"path":        path,
		"status_code": statusCode,
		"duration_ms": duration.Milliseconds(),
	}).Info("HTTP Request")
}

func (l *Logger) LogGRPCRequest(method string, statusCode int, duration time.Duration) {
	l.WithFields(map[string]interface{}{
		"method":      method,
		"status_code": statusCode,
		"duration_ms": duration.Milliseconds(),
	}).Info("gRPC Request")
}

func (l *Logger) LogDatabaseQuery(query string, duration time.Duration, err error) {
	fields := map[string]interface{}{
		"query":       query,
		"duration_ms": duration.Milliseconds(),
	}
	
	if err != nil {
		fields["error"] = err.Error()
		l.WithFields(fields).Error("Database query failed")
	} else {
		l.WithFields(fields).Debug("Database query executed")
	}
}

func (l *Logger) LogCacheOperation(operation, key string, hit bool, duration time.Duration) {
	l.WithFields(map[string]interface{}{
		"operation":   operation,
		"key":         key,
		"cache_hit":   hit,
		"duration_ms": duration.Milliseconds(),
	}).Debug("Cache operation")
}