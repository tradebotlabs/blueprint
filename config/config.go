// By Emran A. Hamdan, Lead Architect 
package config
// Config will use .ENV for docker-compose and load into config
import (
	"fmt"
	"os"
	"time"
)

// Env vars gose here so we don't change names by mistake
const (
	GPRC_HOST         = "GPRC_HOST"
	GRPC_PORT         = "GRPC_PORT"

	REDIS_URL         = "REDIS_URL"
	REDIS_PASSWORD    = "REDIS_PASSWORD"
	
	POSTGRES_HOST     = "POSTGRES_HOST"
	POSTGRES_PORT     = "POSTGRES_PORT"
	POSTGRES_USER     = "POSTGRES_USER"
	POSTGRES_PASSWORD = "POSTGRES_PASSWORD"
	POSTGRES_DB       = "POSTGRES_DB"

)

// Config blueprint microservice
type Config struct {
	Setting  Setting
	GRPC     GRPC
	Logger   Logger
	Redis    Redis
	Postgres Postgres
}

type Setting struct {
	Version string 
	LocalPath string 
	
}

// Logger config
type Logger struct {
	DisableCaller     bool
	DisableStacktrace bool
	Encoding          string
	Level             string
	LogFile           string
}

// Redis config
type Redis struct {
	RedisAddr      string
	RedisPassword  string
	RedisDB        string
	RedisDefaultDB string
	MinIdleConn    int
	PoolSize       int
	PoolTimeout    int
	DB             int
}

// Mongo

type Mongo struct {
	URI         string
	PoolTimeout int
}

// Postgres config
type Postgres struct {
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDBName   string
}

// GRPC gRPC service config
type GRPC struct {
	Host              string
	Port              string
	MaxConnectionIdle time.Duration
	Timeout           time.Duration
	MaxConnectionAge  time.Duration
}

// NewConfig get config from env
func NewConfig() *Config {

	// init config 
	setting := Setting{}
	setting.LocalPath = "./locales/*/*"
	setting.Version = "1.0.0"
	logger := Logger{}
	logger.LogFile = "blueprint.log"
	redis := Redis{}
	gprc := GRPC{}
	postgres := Postgres{}

	c := &Config{
		GRPC:     gprc,
		Logger:   logger,
		Redis:    redis,
		Postgres: postgres,
	}

	parseError := map[string]string{
		GPRC_HOST:         "",
		GRPC_PORT:         "",
		REDIS_URL:         "",
		REDIS_PASSWORD:    "",
		POSTGRES_HOST:     "",
		POSTGRES_PORT:     "",
		POSTGRES_USER:     "",
		POSTGRES_PASSWORD: "",
		POSTGRES_DB:       "",
	}

	redisURL := os.Getenv(REDIS_URL)

	if redisURL != "" {
		c.Redis.RedisAddr = redisURL
		parseError[REDIS_URL] = redisURL		
		//fmt.Println(parseError[redisURL] ,redisURL)
	}

	redisPassword := os.Getenv(REDIS_PASSWORD)
	if redisPassword != "" {
		c.Redis.RedisPassword = redisPassword
		parseError[REDIS_PASSWORD] = redisPassword
	}

	gRPCHost := os.Getenv(GPRC_HOST)
	if gRPCHost != "" {
		c.GRPC.Host = gRPCHost
		parseError[GPRC_HOST] = gRPCHost
	}

	gRPCPort := os.Getenv(GRPC_PORT)
	if gRPCPort != "" {
		c.GRPC.Port = gRPCPort
		parseError[GRPC_PORT] = gRPCPort

	}
 

	postgresHost := os.Getenv(POSTGRES_HOST)
	if postgresHost != "" {
		c.Postgres.PostgresHost = postgresHost
		parseError[POSTGRES_HOST] = postgresHost
	}

	postgresPort := os.Getenv(POSTGRES_PORT)
	if postgresPort != "" {
		c.Postgres.PostgresPort = postgresPort
		parseError[POSTGRES_PORT] = postgresPort
	}

	postgresUser := os.Getenv(POSTGRES_USER)
	if postgresUser != "" {
		c.Postgres.PostgresUser = postgresUser
		parseError[POSTGRES_USER] = postgresUser
	}

	postgresPassword := os.Getenv(POSTGRES_PASSWORD)
	if postgresPassword != "" {
		c.Postgres.PostgresPassword = postgresPassword
		parseError[POSTGRES_PASSWORD] = postgresPassword
	}

	postgresDB := os.Getenv(POSTGRES_DB)
	if postgresDB != "" {
		c.Postgres.PostgresDBName = postgresDB
		parseError[POSTGRES_DB] = postgresDB
	}

	exitParse :=false
	for k, v := range parseError {
			if v=="" {
				exitParse=true
				fmt.Printf("%s = %s\n",k,v)
			}		
	}
	
	// one faild
	if exitParse {
		panic("Env vars not set see list")
	}
	return c
}
