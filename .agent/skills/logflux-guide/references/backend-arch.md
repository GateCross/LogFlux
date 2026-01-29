# LogFlux Backend Architecture

LogFlux uses [Go-Zero](https://go-zero.dev/) framework.

## Directory Structure (`backend/internal`)

- **config/**: Configuration struct definition (`config.go`). Maps to `etc/logflux.yaml`.
- **svc/**: Service Context (`ServiceContext`). Dependency injection container.
    - All models, clients (Redis, DB), and background tasks are initialized here.
- **handler/**: HTTP Connectors.
    - Unmarshals requests and calls Logic.
    - **Do NOT** put business logic here.
- **logic/**: Business Logic Layer.
    - Contains the core application logic.
    - One file per API endpoint usually.
- **model/**: Database Models (GORM or sqlx).
    - Database interactions happen here.
- **types/**: Request/Response structs generated from `.api` file.

## Key Components

### 1. Log Ingestion (`internal/ingest`)
- Handles parsing of log lines.
- Communicates with Caddy or reads file streams.

### 2. Notification System (`internal/notification`)
- Manages alerts and notifications.

### 3. Task Scheduler (`internal/tasks`)
- Background jobs (cron).

## Development Flow
1. **Define API**: Modify `.api` file.
2. **Generate Code**: Run `goctl api go -api logflux.api -dir .`
3. **Implement Logic**: Edit files in `internal/logic`.
4. **Register Dependencies**: Add new resources to `ServiceContext` in `internal/svc`.
