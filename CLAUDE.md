# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a **Forex Platform Blueprint Service** - a template microservice that serves as a blueprint for creating new gRPC-based microservices in the Forex Platform ecosystem. When creating a new service, this entire repository should be renamed and customized according to the JIRA project requirements.

**Owner:** JeelRupapara (zeelrupapara@gmail.com)
**Lead Architect:** Emran A. Hamdan

## Critical Rules

1. **Follow the established code structure** - This blueprint defines the standard architecture for all Forex Platform microservices
2. **Always run the service first** before making changes: `go run cmd/main.go`
3. **If the service doesn't run, STOP and fix or ask for help** - never proceed with changes if the baseline is broken
4. **MUST source environment variables** before running: `source export.sh`
5. **Remove all template comments** before deploying a new service based on this blueprint

## Environment Setup

### Required Environment Variables

All environment variables are defined in `export.sh`. Source this file before development:

```bash
source export.sh
```

Configuration follows a dual-mode pattern:
- **Static values**: Used when running locally (GRPC_HOST=127.0.0.1)
- **Dynamic values**: Used in docker-compose network (MYSQL_HOST=mysql)

The config package (`config/config.go`) validates that all required environment variables are set on startup and panics if any are missing.

### Development Stack

Start supporting services (Redis, MySQL) for local development:

```bash
sudo docker compose -f stack.yaml up -d
```

The stack includes:
- **Redis** (port 6379) - Used for caching via the cache package
- **MySQL 8.0.19** (port 3306) - Database with GORM integration
  - Database: `vfxcore`
  - User: `vfxuser` / Password: `root@12345`
  - Network name: `blueprint`
  - Table prefix: `forex_` (configured in db package)

## Common Development Commands

### Running the Service

```bash
# Source environment variables
source export.sh

# Run the service
go run cmd/main.go
```

### Building

```bash
make build  # Builds to blueprint-srv binary
```

### Testing

```bash
make test  # Runs all tests with coverage
```

To run a single test:

```bash
go test -v -run TestName ./path/to/package
```

### Protocol Buffers

When modifying `.proto` files:

```bash
# First time setup (installs protoc generators)
make init

# Regenerate Go code from proto files
make proto
```

This generates `blueprint.pb.go` and `blueprint_grpc.pb.go` in `proto/blueprint/`.

### Docker

```bash
# Build Docker image
make docker

# Run full service stack with docker-compose
sudo docker compose up -d
```

## Architecture

### Service Initialization Flow

1. **`cmd/main.go`**: Entry point that calls `app.Start()`
2. **`app/app.go`**: Orchestrates service startup in this order:
   - Load configuration from environment variables
   - Initialize logger (Zap with file rotation via Lumberjack)
   - Initialize i18n for multi-language support (en-US, el-GR, zh-CN)
   - Start gRPC server with keepalive and recovery middleware
   - Connect to Redis and wrap with cache client
   - Connect to MySQL and run migrations
   - Register gRPC handlers
   - Enable Prometheus metrics
   - Start graceful shutdown handler (listens for SIGTERM/SIGINT)

### Key Packages

**`pkg/logger`** (logger.go:50)
- Structured logging with Zap
- Dual output: JSON to file + console output
- Log rotation with Lumberjack (max 100MB, 30 backups, 30 day retention)
- Specialized logging methods: `LogGRPCRequest()`, `LogDatabaseQuery()`, `LogCacheOperation()`

**`pkg/cache`** (cache.go:44)
- Redis-backed caching with automatic JSON marshaling
- Key prefixing (`blueprint:` by default)
- Retry logic with exponential backoff (3 retries, 100ms base delay)
- Batch operations via Redis pipelining: `SetBatch()`, `GetBatch()`
- Built-in statistics tracking (hits, misses, sets, deletes)

**`pkg/db`** (mysql.go:39)
- GORM wrapper with MySQL driver
- Connection pooling (10 idle, 100 max open connections)
- Table naming strategy: `forex_` prefix, singular table names
- Auto-migration support via `Migrate()` function
- Health check with 2-second timeout

**`pkg/redis`** (redis.go)
- Redis client initialization with password support
- Configured from environment variables

**`pkg/i18n`** (i18n.go)
- Multi-language support using kataras/i18n
- Locales stored in `./locales/*/*` path pattern

**`pkg/errors`** (errors.go)
- Centralized error handling utilities

### Handler Pattern

Handlers (`handler/blueprint.go`) follow this pattern:

1. **Embed `UnimplementedBlueprintServer`** for forward compatibility
2. **Inject dependencies** via constructor: i18n, logger, cache, DB
3. **Request flow**:
   - Validate request (nil checks, required fields, length limits)
   - Check rate limiting (100 requests/minute per identifier)
   - Set request timeout (30 seconds default)
   - Check cache for existing response
   - Process business logic
   - Cache response with TTL (5 minutes default)
   - Record metrics

Handlers include built-in:
- Rate limiting with sliding window
- Metrics tracking (requests, cache hits/misses, response times)
- Health check functionality

### gRPC Configuration

The gRPC server is configured with (app.go:61-93):
- **Keepalive enforcement**: 5-second minimum, permits streams
- **Server parameters**: 15s idle, 30s max age, 5s grace period
- **Recovery middleware**: Catches panics and logs them
- **Prometheus metrics**: All requests tracked via `grpc_prometheus`
- **Reflection enabled**: For tools like grpcurl

### Models

Domain models are organized in `model/` by entity:
- `model/blueprint/model.go` - Example model with GORM struct tags
- `model/other/other.go` - Additional models

Models use the `forex_` table prefix and singular naming (configured in db package).

## Development Workflow

### Creating a New Service from Blueprint

1. Clone/rename repository according to JIRA ticket (e.g., `blueprint-svc-STORY-XXX`)
2. Update service name and version in `app/app.go:31-32`
3. Modify proto file in `proto/blueprint/blueprint.proto`
4. Run `make proto` to regenerate gRPC code
5. Update handler logic in `handler/`
6. Update models in `model/`
7. Remove all blueprint-specific comments
8. Test locally with `source export.sh && go run cmd/main.go`

### Git Branching Strategy

- **Branch naming**: `blueprint-svc-BUG-XXX` or `blueprint-svc-STORY-XXX`
- **Testing requirement**: Test approval required before merge to master
- **Merge process**:
  ```bash
  git checkout master
  git merge blueprint-svc-BUG-XXX
  ```

### Database Migrations

Auto-migrations run on startup via `db.Migrate()` in `app.go:120`. Add new models to the migration list in `pkg/db/mysql.go:153`.

### Testing Strategy

Each handler should have a corresponding `*_test.go` file (e.g., `handler/blueprint_test.go`). Tests should verify:
- Request validation
- Cache behavior
- Error handling
- Business logic

## Important Notes

- **Database timezone**: Set to `Asia/Amman` in stack.yaml
- **Default authentication**: MySQL uses `mysql_native_password` plugin
- **Data persistence**: MySQL data stored in `${MYSQL_DATA}` path from export.sh
- **Log output**: Combined file + console logging, file located at `blueprint.log` by default
- **Service discovery**: Uses Docker network name `blueprint` for inter-service communication
