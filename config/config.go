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
	GPRC_HOST      = "GPRC_HOST"
	GRPC_PORT      = "GRPC_PORT"
	REDIS_URL      = "REDIS_URL"
	REDIS_PASSWORD = "REDIS_PASSWORD"
	MYSQL_HOST     = "MYSQL_HOST"
	MYSQL_PORT     = "MYSQL_PORT"
	MYSQL_USER     = "MYSQL_USER"
	MYSQL_PASSWORD = "MYSQL_PASSWORD"
	
)

// Config blueprint microservice
type Config struct {
	Setting Setting
	GRPC   GRPC
	Logger Logger
	Redis  Redis
	MySQL  MySQL
	
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

// MySQL config
type MySQL struct {
	MysqlHost     string
	MysqlPort     string
	MysqlUser     string
	MysqlPassword string
	MysqlDBName   string
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
	mysql := MySQL{}

 	c := &Config{
		GRPC:   gprc,
		Logger: logger,
		Redis:  redis,
		MySQL:  mysql,	 
	}

	parseError := map[string]string{
		GPRC_HOST:      "",
		GRPC_PORT:      "",
		REDIS_URL:      "",
		REDIS_PASSWORD: "",
		MYSQL_HOST:     "",
		MYSQL_PORT:     "",
		MYSQL_USER:     "",
		MYSQL_PASSWORD: "",
	 
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
 

	mysqlHost := os.Getenv(MYSQL_HOST)

	if mysqlHost != "" {
		c.MySQL.MysqlHost = mysqlHost
		parseError[MYSQL_HOST] = mysqlHost
	}

	mysqlPort := os.Getenv(MYSQL_PORT)
	if mysqlPort != "" {
		c.MySQL.MysqlPort = mysqlPort
		parseError[MYSQL_PORT] = mysqlPort

	}

	mysqlUser := os.Getenv(MYSQL_USER)

	if mysqlUser != "" {
		c.MySQL.MysqlUser = mysqlUser
		parseError[MYSQL_USER] = mysqlUser
	}

	mysqlPassword := os.Getenv(MYSQL_PASSWORD)

	if mysqlPassword != "" {
		c.MySQL.MysqlPassword = mysqlPassword
		parseError[MYSQL_PASSWORD] = mysqlPassword
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
