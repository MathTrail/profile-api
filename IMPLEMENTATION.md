# Profile Service Implementation Plan

## Technology Stack

**Core:**
- Go 1.25.7
- Gin (HTTP framework)
- GORM (ORM for PostgreSQL)
- PostgreSQL (Database)
- Redis (Profile cache)

**Event-Driven Architecture:**
- Dapr (service mesh & pub/sub)
- Kafka (message broker)
- CloudEvents (event standard)

**API & Documentation:**
- Swagger/OpenAPI
- Gin Swagger UI

**Development & Deployment:**
- Docker & Dev Containers
- Skaffold (local dev loop against k3d cluster)
- k3d (local Kubernetes)
- Helm (package manager)
- justfile (task automation)

**Testing:**
- Go testing (standard library)
- testify (assertions & mocking)
- Grafana k6 (scenario & load tests via REST API)

---

## Overview

Implement a Profile Service that manages user profiles in the MathTrail ecosystem through REST API and event-driven architecture (Dapr pub/sub & Kafka).

**Key Features:**
- Automatic profile creation via Ory Kratos webhook events (CloudEvents)
- Profile retrieval via REST API (Gin + Swagger)
- Redis cache for profile reads with invalidation on every profile mutation
- Skill and progress tracking via task-solved events (Kafka pub/sub)
- PostgreSQL database backend with GORM ORM
- Full observability integration via Dapr
- Local development with Skaffold dev mode (depends on mathtrail-infra-local, mathtrail-infra-testing)

---

## Stage 1: Project Setup & Configuration

### Objectives
- [x] Create DevContainer with all dependencies
- [x] Set up Go project structure
- [x] Configure Skaffold with infra-local dependency
- [x] Configure Dapr components
- [x] Initialize dependencies

### Tasks

#### 1.1 DevContainer Setup (First Step)
Create the DevContainer with all development tools pre-installed so the developer can start coding immediately.

**File:** `.devcontainer/Dockerfile`
```dockerfile
FROM golang:1.25.7-bookworm

RUN apt-get update && apt-get install -y \
    git \
    curl \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

# Install Skaffold
RUN curl -Lo /tmp/skaffold "https://storage.googleapis.com/skaffold/releases/latest/skaffold-linux-amd64" \
    && install /tmp/skaffold /usr/local/bin/skaffold \
    && rm -f /tmp/skaffold

# Install Go dev tools
RUN go install golang.org/x/tools/gopls@latest && \
    go install github.com/go-delve/delve/cmd/dlv@latest && \
    go install honnef.co/go/tools/cmd/staticcheck@latest && \
    go install github.com/swaggo/swag/cmd/swag@latest

WORKDIR /workspace
```

**File:** `.devcontainer/devcontainer.json`
```jsonc
{
    "name": "MathTrail Profile DevContainer",
    "build": {
        "dockerfile": "Dockerfile",
        "context": ".."
    },
    "features": {
        "ghcr.io/devcontainers/features/docker-in-docker:2": {},
        "ghcr.io/devcontainers/features/kubectl-helm-minikube:1.2.2": {
            "helm": "3.14.0",
            "version": "1.31.0",
            "minikube": "none"
        },
        "ghcr.io/eitsupi/devcontainer-features/just:0.1.1": {},
        "ghcr.io/dapr/cli/dapr-cli:0": {}
    },
    "mounts": [
        "source=${localEnv:HOME}${localEnv:USERPROFILE}/.kube,target=/home/vscode/.kube,type=bind,consistency=cached"
    ],
    "remoteEnv": {
        "KUBECONFIG": "/home/vscode/.kube/k3d-mathtrail-dev.yaml"
    },
    "customizations": {
        "vscode": {
            "extensions": [
                "golang.go",
                "ms-azuretools.vscode-docker",
                "ms-kubernetes-tools.vscode-kubernetes-tools",
                "ms-kubernetes.helm",
                "redhat.vscode-yaml",
                "eamodio.gitlens",
                "sclu1034.justfile",
                "Tim-Koehler.helm-intellisense"
            ]
        }
    },
    "postCreateCommand": "bash .devcontainer/post-create.sh",
    "forwardPorts": [8080]
}
```

**File:** `.devcontainer/post-create.sh`
```bash
#!/bin/bash
set -e

mkdir -p /home/vscode/.kube
chmod 700 /home/vscode/.kube 2>/dev/null || true

# Download Go dependencies
go mod download

echo "Checking cluster connection..."
if kubectl cluster-info 2>/dev/null; then
    echo "✅ Connected to K3d cluster"
else
    echo "⚠️  Cluster not accessible. Run 'just create' in mathtrail-infra-local-k3s first"
fi
```

#### 1.2 Project Structure
```
mathtrail-profile/
├── cmd/
│   └── server/
│       └── main.go             # Entry point
├── internal/
│   ├── app/
│   │   └── container.go        # Dependency injection container
│   ├── config/
│   │   └── config.go           # Configuration management
│   ├── database/
│   │   └── database.go         # GORM DB connection & initialization
│   ├── server/
│   │   └── router.go           # Gin HTTP router and middleware
│   ├── cache/
│   │   └── profile.go          # Redis cache layer for profiles
│   ├── profile/
│   │   ├── model.go            # GORM models: Profile, Skill, Progress
│   │   ├── repository.go       # GORM repository (interface + impl)
│   │   ├── service.go          # Business logic with cache-aside
│   │   └── controller.go       # Gin HTTP controllers
│   └── dapr/
│       ├── subscriber.go       # Dapr pub/sub subscriptions
│       ├── events.go           # CloudEvents handlers
│       └── models.go           # Event payload models
├── migrations/
│   └── 001_init.sql            # PostgreSQL schema migration
├── k6/
│   ├── scenarios/              # API scenario tests (functional flows)
│   │   ├── profile-crud.js     # Profile create / read / update flow
│   │   ├── health.js           # Health check scenario
│   │   └── ...
│   └── load/                   # Load & stress tests
│       ├── profile-read.js     # GET /profile under load
│       ├── mixed-traffic.js    # Combined read/write workload
│       └── ...
├── justfile                    # Task automation (build, test, run, debug)
├── skaffold.yaml               # Skaffold config (depends on mathtrail-infra-local)
├── Dockerfile
├── .devcontainer/
│   ├── Dockerfile              # Dev container with Go, Skaffold, Delve, etc.
│   ├── devcontainer.json       # Dev container config with k8s tooling
│   └── post-create.sh          # Post-create setup script
├── helm/
│   └── mathtrail-profile/
│       ├── Chart.yaml
│       ├── values.yaml         # Helm values for k3d/Kubernetes
│       └── templates/
│           ├── deployment.yaml # k3d deployment with Dapr sidecar
│           ├── service.yaml
│           └── configmap.yaml
├── dapr/
│   └── components.yaml         # Dapr Kafka pub/sub component
├── go.mod
├── go.sum
└── README.md
```

#### 1.3 Go Module Initialization
```bash
go mod init github.com/MathTrail/profile-api
# Web Framework
go get github.com/gin-gonic/gin
# ORM & Database
go get gorm.io/gorm
go get gorm.io/driver/postgres
# UUID
go get github.com/google/uuid
# Dapr SDK
go get github.com/dapr/go-sdk
# Cloud Events
go get github.com/cloudevents/sdk-go/v2
# Logging
go get go.uber.org/zap
# Swagger & OpenAPI
go get github.com/swaggo/swag/cmd/swag
go get github.com/swaggo/gin-swagger
go get github.com/swaggo/files
# Redis
go get github.com/redis/go-redis/v9
# Testing
go get github.com/stretchr/testify
```

#### 1.4 Environment Configuration
Create `config/config.go` to handle:
- Database connection string
- Redis connection string (host, port, password, DB index)
- Redis cache TTL (default: 5 minutes)
- Server port
- Dapr sidecar host/port
- Log level
- Pub/sub topic names

#### 1.5 Production Dockerfile
```dockerfile
FROM golang:1.25.7-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app ./cmd/server

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app /app

EXPOSE 8080

ENTRYPOINT ["/app"]
```

#### 1.6 Skaffold Configuration
**File:** `skaffold.yaml`

The profile service depends on infrastructure from `mathtrail-infra-local` (PostgreSQL, Redis, Kafka, Strimzi) and testing infrastructure from `mathtrail-infra-testing` (Grafana k6 operator). Skaffold's `requires` field references both configs so that `skaffold dev` brings up everything.

```yaml
apiVersion: skaffold/v4beta13
kind: Config
metadata:
  name: mathtrail-profile

requires:
  - path: ../mathtrail-infra-local
  - path: ../mathtrail-infra-testing

build:
  artifacts:
    - image: profile-api
      docker:
        dockerfile: Dockerfile
  local:
    push: false

deploy:
  helm:
    releases:
      - name: mathtrail-profile
        chartPath: helm/mathtrail-profile
        namespace: mathtrail
        createNamespace: true
        valuesFiles:
          - helm/mathtrail-profile/values.yaml
        setValues:
          image.repository: mathtrail-profile
          image.tag: latest
```

Skaffold local mode (`push: false`) builds images directly into the k3d cluster's Docker daemon, avoiding registry pushes.

#### 1.7 Task Automation with justfile
**File:** `justfile`

```bash
# Development commands
just build                  # Build binary
just test                   # Run all tests
just test-integration       # Run integration tests
just test-coverage          # Generate coverage report

# k6
just k6-scenarios            # Run all scenario tests (k6/scenarios/)
just k6-load                 # Run all load tests (k6/load/)
just k6-run FILE              # Run a specific k6 test file

# Skaffold
just dev                    # skaffold dev (build + deploy + watch)
just deploy                 # skaffold run (one-shot build + deploy)
just delete                 # skaffold delete (remove from cluster)
just logs                   # View pod logs

# Docker
just docker-build           # Build Docker image locally
```

---

## Stage 2: Database Schema & Models

### Objectives
- [x] Define PostgreSQL schema
- [x] Create SQL migration files
- [x] Define GORM data models
- [x] Implement repository layer with interfaces

### Tasks

#### 2.1 Database Schema (PostgreSQL)
**File:** `migrations/001_init.sql`
```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS profiles (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     VARCHAR(255) NOT NULL UNIQUE,
    name        VARCHAR(255) NOT NULL,
    age         INT,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS skills (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id  UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    skill_name  VARCHAR(100) NOT NULL,
    level       INT NOT NULL DEFAULT 1,
    proficiency DECIMAL(3, 2),
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(profile_id, skill_name)
);

CREATE TABLE IF NOT EXISTS progress (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id      UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    tasks_completed INT NOT NULL DEFAULT 0,
    current_streak  INT NOT NULL DEFAULT 0,
    total_points    INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_profiles_user_id ON profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_skills_profile_id ON skills(profile_id);
CREATE INDEX IF NOT EXISTS idx_progress_profile_id ON progress(profile_id);
```

#### 2.2 GORM Data Models
**File:** `internal/profile/model.go`
```go
package profile

import (
    "time"

    "github.com/google/uuid"
)

type Profile struct {
    ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
    UserID    string    `gorm:"uniqueIndex;not null" json:"user_id"`
    Name      string    `gorm:"not null" json:"name"`
    Age       int       `json:"age"`
    Skills    []Skill   `gorm:"foreignKey:ProfileID" json:"skills"`
    Progress  *Progress `gorm:"foreignKey:ProfileID" json:"progress"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type Skill struct {
    ID          uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
    ProfileID   uuid.UUID `gorm:"type:uuid;index;not null" json:"profile_id"`
    SkillName   string    `gorm:"uniqueIndex:idx_profile_skill;not null" json:"skill_name"`
    Level       int       `gorm:"default:1;not null" json:"level"`
    Proficiency float64   `json:"proficiency"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type Progress struct {
    ID             uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
    ProfileID      uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"profile_id"`
    TasksCompleted int       `gorm:"default:0;not null" json:"tasks_completed"`
    CurrentStreak  int       `gorm:"default:0;not null" json:"current_streak"`
    TotalPoints    int       `gorm:"default:0;not null" json:"total_points"`
    CreatedAt      time.Time `json:"created_at"`
    UpdatedAt      time.Time `json:"updated_at"`
}
```

#### 2.3 GORM Repository Layer
**File:** `internal/profile/repository.go`

Interface-based repository pattern (following `tmp.service` conventions):

```go
package profile

import (
    "context"
    "fmt"

    "github.com/google/uuid"
    "gorm.io/gorm"
)

type Repository interface {
    Create(ctx context.Context, profile *Profile) error
    GetByUserID(ctx context.Context, userID string) (*Profile, error)
    UpdateSkills(ctx context.Context, profileID uuid.UUID, skills []Skill) error
    UpdateProgress(ctx context.Context, profileID uuid.UUID, progress *Progress) error
    GetOrCreate(ctx context.Context, profile *Profile) (*Profile, error)
}

type repositoryImpl struct {
    db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
    return &repositoryImpl{db: db}
}

func (r *repositoryImpl) Create(ctx context.Context, profile *Profile) error {
    if err := r.db.WithContext(ctx).Create(profile).Error; err != nil {
        return fmt.Errorf("failed to create profile: %w", err)
    }
    return nil
}

func (r *repositoryImpl) GetByUserID(ctx context.Context, userID string) (*Profile, error) {
    var profile Profile
    err := r.db.WithContext(ctx).
        Preload("Skills").
        Preload("Progress").
        Where("user_id = ?", userID).
        First(&profile).Error
    if err != nil {
        return nil, fmt.Errorf("failed to get profile: %w", err)
    }
    return &profile, nil
}
```

Additional methods:
- `UpdateSkills` — upsert skill array via GORM `Save`
- `UpdateProgress` — update progress row
- `GetOrCreate` — GORM `FirstOrCreate` pattern

#### 2.4 Database Connection & DI Container
**File:** `internal/database/database.go`

```go
package database

import (
    "log"

    "gorm.io/driver/postgres"
    "gorm.io/gorm"

    "github.com/MathTrail/profile-api/internal/config"
)

func NewConnection(cfg *config.Config) *gorm.DB {
    db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
    if err != nil {
        log.Fatalf("failed to connect to database: %v", err)
    }

    sqlDB, _ := db.DB()
    sqlDB.SetMaxOpenConns(25)
    sqlDB.SetMaxIdleConns(5)

    return db
}
```

**File:** `internal/app/container.go`

Dependency injection container wiring all layers together:

```go
package app

import (
    "time"

    "github.com/redis/go-redis/v9"
    "gorm.io/gorm"

    "github.com/MathTrail/profile-api/internal/cache"
    "github.com/MathTrail/profile-api/internal/config"
    "github.com/MathTrail/profile-api/internal/database"
    "github.com/MathTrail/profile-api/internal/profile"
)

type Container struct {
    DB                *gorm.DB
    ProfileController *profile.Controller
}

func NewContainer(cfg *config.Config) *Container {
    // Database
    db := database.NewConnection(cfg)

    // Redis
    rdb := redis.NewClient(&redis.Options{
        Addr:     cfg.RedisAddr,
        Password: cfg.RedisPassword,
        DB:       cfg.RedisDB,
    })
    profileCache := cache.NewProfileCache(rdb, time.Duration(cfg.CacheTTLSeconds)*time.Second, nil)

    // Profile domain
    profileRepo := profile.NewRepository(db)
    profileService := profile.NewService(profileRepo, profileCache)
    profileController := profile.NewController(profileService)

    return &Container{
        DB:                db,
        ProfileController: profileController,
    }
}

func (c *Container) Close() {
    sqlDB, _ := c.DB.DB()
    sqlDB.Close()
}
```

---

## Stage 3: Core API Implementation

### Objectives
- [x] Implement GET /profile/{userId} endpoint
- [x] Add request validation
- [x] Implement error handling
- [x] Add Swagger/OpenAPI documentation

### Tasks

#### 3.1 HTTP Controller
**File:** `internal/profile/controller.go`

```go
package profile

import (
    "net/http"

    "github.com/gin-gonic/gin"
)

type Controller struct {
    service Service
}

func NewController(service Service) *Controller {
    return &Controller{service: service}
}

func (c *Controller) RegisterRoutes(router *gin.RouterGroup) {
    profiles := router.Group("/profile")
    {
        profiles.GET("/:userId", c.GetByUserID)
    }
}

// GetByUserID godoc
// @Summary Get user profile
// @Description Retrieve complete user profile by user ID
// @Tags profiles
// @Produce json
// @Param userId path string true "User ID from Kratos"
// @Success 200 {object} Profile "Profile retrieved successfully"
// @Failure 404 {object} ErrorResponse "Profile not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /profile/{userId} [get]
func (c *Controller) GetByUserID(ctx *gin.Context) {
    userID := ctx.Param("userId")

    profile, err := c.service.GetProfile(ctx.Request.Context(), userID)
    if err != nil {
        ctx.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
        return
    }

    ctx.JSON(http.StatusOK, profile)
}
```

#### 3.2 Redis Cache Layer
**File:** `internal/cache/profile.go`

```go
import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
    "go.uber.org/zap"
)

type ProfileCache struct {
    client *redis.Client
    ttl    time.Duration
    logger *zap.Logger
}

func NewProfileCache(client *redis.Client, ttl time.Duration, logger *zap.Logger) *ProfileCache {
    return &ProfileCache{client: client, ttl: ttl, logger: logger}
}

func (c *ProfileCache) cacheKey(userID string) string {
    return fmt.Sprintf("profile:%s", userID)
}

// Get retrieves a cached profile; returns nil on cache miss
func (c *ProfileCache) Get(ctx context.Context, userID string) (*models.Profile, error) {
    data, err := c.client.Get(ctx, c.cacheKey(userID)).Bytes()
    if err == redis.Nil {
        return nil, nil // cache miss
    }
    if err != nil {
        return nil, err
    }
    var profile models.Profile
    if err := json.Unmarshal(data, &profile); err != nil {
        return nil, err
    }
    return &profile, nil
}

// Set stores a profile in cache with TTL
func (c *ProfileCache) Set(ctx context.Context, userID string, profile *models.Profile) error {
    data, err := json.Marshal(profile)
    if err != nil {
        return err
    }
    return c.client.Set(ctx, c.cacheKey(userID), data, c.ttl).Err()
}

// Invalidate removes a profile from cache
func (c *ProfileCache) Invalidate(ctx context.Context, userID string) error {
    return c.client.Del(ctx, c.cacheKey(userID)).Err()
}
```

#### 3.3 Service Layer
**File:** `internal/profile/service.go`

Interface-based service with cache-aside pattern:

```go
package profile

import (
    "context"

    "github.com/google/uuid"
    "go.uber.org/zap"

    "github.com/MathTrail/profile-api/internal/cache"
)

type Service interface {
    GetProfile(ctx context.Context, userID string) (*Profile, error)
    CreateProfile(ctx context.Context, profile *Profile) error
    UpdateSkills(ctx context.Context, userID string, profileID uuid.UUID, skills []Skill) error
    UpdateProgress(ctx context.Context, userID string, profileID uuid.UUID, progress *Progress) error
}

type serviceImpl struct {
    repo   Repository
    cache  *cache.ProfileCache
    logger *zap.Logger
}

func NewService(repo Repository, cache *cache.ProfileCache) Service {
    return &serviceImpl{repo: repo, cache: cache}
}
```

Implement cache-aside:
- Profile retrieval with cache-aside pattern:
  1. Check Redis cache → return on hit
  2. Query PostgreSQL on cache miss
  3. Populate Redis cache before returning
- Cache invalidation on every mutation:
  - `CreateProfile` → invalidate cache for userID
  - `UpdateSkills` → invalidate cache for userID
  - `UpdateProgress` → invalidate cache for userID
- Business rule validation

```go
func (s *serviceImpl) GetProfile(ctx context.Context, userID string) (*Profile, error) {
    // 1. Try cache
    cached, err := s.cache.Get(ctx, userID)
    if err != nil {
        s.logger.Warn("cache read failed, falling back to DB", zap.Error(err))
    }
    if cached != nil {
        return cached, nil
    }

    // 2. Query DB
    profile, err := s.repo.GetProfileByUserID(ctx, userID)
    if err != nil {
        return nil, err
    }

    // 3. Populate cache (best-effort)
    if err := s.cache.Set(ctx, userID, profile); err != nil {
        s.logger.Warn("cache write failed", zap.Error(err))
    }

    return profile, nil
}

func (s *serviceImpl) UpdateSkills(ctx context.Context, userID string, profileID uuid.UUID, skills []Skill) error {
    if err := s.repo.UpdateSkills(ctx, profileID, skills); err != nil {
        return err
    }
    // Invalidate cache so next read picks up fresh data
    return s.cache.Invalidate(ctx, userID)
}
```

#### 3.4 Router Setup
**File:** `internal/server/router.go`

```go
package server

import (
    "github.com/gin-gonic/gin"

    "github.com/MathTrail/profile-api/internal/profile"
)

func NewRouter(profileController *profile.Controller) *gin.Engine {
    router := gin.Default()

    // Register routes
    api := router.Group("/api/v1")
    profileController.RegisterRoutes(api)

    // Health check
    router.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "ok"})
    })

    // Swagger
    router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

    return router
}
```

#### 3.5 Error Handling
**File:** `internal/handler/response.go`

```go
type ErrorResponse struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}

// Standard error codes
const (
    ErrProfileNotFound = "PROFILE_NOT_FOUND"
    ErrInvalidUserID   = "INVALID_USER_ID"
    ErrInternal        = "INTERNAL_ERROR"
)
```

#### 3.6 Swagger Documentation
- Run: `swag init -g cmd/server/main.go`
- Endpoint: `http://localhost:8080/swagger/index.html`
- Update comments with proper Swagger annotations

---

## Stage 4: Dapr/Event Integration

### Objectives
- [x] Set up Dapr pub/sub subscriber
- [x] Implement user-registered event handler
- [x] Implement task-solved event handler
- [x] Configure Dapr components

### Tasks

#### 4.1 Dapr Component Configuration
**File:** `dapr/components.yaml`

```yaml
apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: pubsub
spec:
  type: pubsub.kafka
  version: v1
  metadata:
  - name: brokers
    value: kafka:9092
  - name: consumerGroup
    value: profile-service
  - name: topics
    value: "user-registered,task-solved"
```

Note: Dapr state store not required; Profile Service uses GORM + PostgreSQL for persistence and Redis for read-through caching.

#### 4.2 Dapr Event Subscriber Handler
**File:** `internal/dapr/subscriber.go`

```go
import (
    "context"
    daprclient "github.com/dapr/go-sdk/client"
    "github.com/cloudevents/sdk-go/v2/event"
    "go.uber.org/zap"
)

type EventHandler struct {
    profileService profile.Service
    logger         *zap.Logger
    daprClient     daprclient.Client
}

// Subscribe registers handlers for topics
func (eh *EventHandler) Subscribe(ctx context.Context) error {
    // Register handlers with Dapr client
    // Subscribe to user-registered topic
    // Subscribe to task-solved topic
}

// HandleUserRegistered processes CloudEvents from user-registered topic
func (eh *EventHandler) HandleUserRegistered(ctx context.Context, e *event.Event) error {
    // Parse CloudEvent data
    // Extract UserRegisteredEvent payload
    // Create profile via service (cache invalidated automatically)
    // Log event with correlation ID
}

// HandleTaskSolved processes CloudEvents from task-solved topic
func (eh *EventHandler) HandleTaskSolved(ctx context.Context, e *event.Event) error {
    // Parse CloudEvent data
    // Extract TaskSolvedEvent payload
    // Update skills and progress via service (cache invalidated automatically)
    // Log event
}
```

#### 4.3 CloudEvents Models
**File:** `internal/dapr/models.go`

```go
import ce "github.com/cloudevents/sdk-go/v2/event"

type UserRegisteredEvent struct {
    UserID    string    `json:"user_id"`
    Name      string    `json:"name"`
    Age       int       `json:"age"`
    Email     string    `json:"email"`
    Timestamp time.Time `json:"timestamp"`
}

type TaskSolvedEvent struct {
    UserID        string        `json:"user_id"`
    TaskID        string        `json:"task_id"`
    SkillsGained  map[string]int `json:"skills_gained"`
    PointsEarned  int           `json:"points_earned"`
    Timestamp     time.Time     `json:"timestamp"`
}

// Helper to create CloudEvent envelope
func NewCloudEvent(eventType, source string, data interface{}) *ce.Event {
    e := ce.New()
    e.SetType(eventType)
    e.SetSource(source)
    e.SetData(ce.ApplicationJSON, data)
    return &e
}
```

#### 4.4 Pub/Sub Endpoint Setup
**File:** `cmd/server/main.go`

```go
package main

import (
    "fmt"
    "log"

    "github.com/MathTrail/profile-api/internal/app"
    "github.com/MathTrail/profile-api/internal/config"
    "github.com/MathTrail/profile-api/internal/server"
)

func main() {
    // Load configuration
    cfg := config.Load()

    // Initialize dependencies
    container := app.NewContainer(cfg)
    defer container.Close()

    // Setup router
    router := server.NewRouter(container.ProfileController)

    // Register Dapr pub/sub endpoints
    // POST /dapr/subscribe - Dapr calls this to discover subscriptions
    // POST /user-registered - Handle user-registered events
    // POST /task-solved - Handle task-solved events

    // Start server
    addr := fmt.Sprintf(":%s", cfg.ServerPort)
    log.Printf("server starting on %s", addr)
    if err := router.Run(addr); err != nil {
        log.Fatalf("failed to start server: %v", err)
    }
}
```

#### 4.5 Local Development Setup
Infrastructure (PostgreSQL, Redis, Kafka) is deployed via Skaffold's `requires` dependency on `mathtrail-infra-local`. Running `just dev` (i.e. `skaffold dev`) automatically deploys the infra first, then builds and deploys the profile service.

---

## Stage 5: Observability

### Objectives
- [x] Configure logging and tracing
- [x] Baseline metrics via Dapr sidecar
- [x] (skip) Custom business metrics (deferred until SLOs defined)

### Tasks

#### 5.1 Logging
Use `zap` for structured logging:

```go
logger, _ := zap.NewProduction()
logger.Info("profile created",
    zap.String("user_id", userID),
    zap.Time("timestamp", time.Now()),
)
```

#### 5.2 Metrics & Tracing
Integrate with Dapr sideca & observability stack:
- Profile creation count (counter metric)
- API response times (histogram)
- Event processing latency (histogram)
- Database query duration via GORM logger
- Trace correlation IDs across services
- Export metrics/traces to observability backend (Prometheus, Jaeger, etc.)

#### 5.3 Tracing
Integration with distributed tracing:
- Trace profile creation events
- Trace API calls
- Correlation IDs for request tracking

---

## Stage 6: Documentation & Deployment

### Objectives
- [ ] Complete OpenAPI/Swagger documentation
- [ ] Write API documentation
- [ ] Create deployment guides
- [ ] Prepare example payloads

### Tasks

#### 6.1 API Documentation
**File:** `API.md`

Document:
- Base URL
- Authentication (if applicable)
- Endpoints with examples
- Error codes
- Rate limiting (if applicable)

#### 6.2 Example Payloads

**Profile Response Example:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "kratos-user-123",
  "name": "John Doe",
  "age": 25,
  "skills": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "skill_name": "Algebra",
      "level": 2,
      "proficiency": 0.75
    }
  ],
  "progress": {
    "id": "550e8400-e29b-41d4-a716-446655440002",
    "tasks_completed": 5,
    "current_streak": 3,
    "total_points": 150
  },
  "created_at": "2024-02-07T10:00:00Z",
  "updated_at": "2024-02-07T10:00:00Z"
}
```

**User Registered Event Example:**
```json
{
  "user_id": "kratos-user-123",
  "name": "John Doe",
  "age": 25,
  "email": "john@example.com",
  "timestamp": "2024-02-07T10:00:00Z"
}
```

#### 6.3 Deployment Guide

**Prerequisites:**
- k3d cluster running (via `just create` in `mathtrail-infra-local-k3s`)
- `~/.kube/k3d-mathtrail-dev.yaml` present

**Local Development with Skaffold (recommended):**
```bash
# Deploy infra + profile service in watch mode (rebuilds on file changes)
just dev

# One-shot deploy (no watch)
just deploy

# View pod logs
just logs

# Remove everything from cluster
just delete

# Run tests
just test
just test-coverage

# View Swagger UI (after port-forward or service exposure)
http://localhost:8080/swagger/index.html
```

`just dev` runs `skaffold dev` which:
1. Deploys `mathtrail-infra-local` (PostgreSQL, Redis, Kafka) via the `requires` dependency
2. Builds the profile service Docker image locally (no registry push)
3. Deploys the profile service Helm chart to k3d
4. Watches for file changes and redeploys automatically

#### 6.4 Helm Configuration for k3d
**File:** `helm/mathtrail-profile/values.yaml`

Configure:
- Image: mathtrail-profile (built locally by Skaffold, no registry push)
- Resource limits (CPU, memory)
- Environment variables (DB_HOST, REDIS_HOST, REDIS_PORT, CACHE_TTL, DAPR_HOST, etc.)
- Service type: ClusterIP (for k3d internal service discovery)
- Replicas: 1 (single instance for k3d)
- Dapr annotations:
  - `dapr.io/enabled: "true"`
  - `dapr.io/id: "profile-service"`
  - `dapr.io/port: "8080"`

**File:** `helm/mathtrail-profile/templates/configmap.yaml`
- Dapr pub/sub configuration
- Kafka broker addresses
- Log levels

#### 6.5 README Updates
Update main `README.md` with:
- **Quick Start** — Using justfile: `just dev` (Skaffold dev loop), `just test`
- **Architecture Diagram** — Show GORM, Gin, Dapr, CloudEvents flow
- **Configuration Options** — Environment variables for PostgreSQL, Redis, Dapr, Kafka
- **Tech Stack** — Go, Gin, GORM, PostgreSQL, Redis, Dapr, Kafka, CloudEvents, Swagger, Skaffold, Docker, k3d, Helm
- **Local Development** — DevContainer with Skaffold dev loop against k3d
- **Debugging** — Delve remote debugging with Skaffold debug
- **Testing** — testify, unit vs integration tests, k6 scenario & load tests, coverage
- **Deployment** — Skaffold + Helm to k3d, infra-local dependency
- **Troubleshooting** — Common issues and solutions
- **Contributing** — Guidelines for adding features

---

## Stage 7: k6 Integration Testing

### Objectives
- [ ] Create k6 scenario & load tests

### Tasks

#### 7.1 k6 Scenario Tests (REST API)
**Directory:** `k6/scenarios/`

Each file is a self-contained scenario test. Example:

**File:** `k6/scenarios/profile-crud.js`

```javascript
import http from 'k6/http';
import { check, group, sleep } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export const options = {
    scenarios: {
        smoke: {
            executor: 'shared-iterations',
            vus: 1,
            iterations: 1,
        },
    },
    thresholds: {
        http_req_failed: ['rate==0'],
        http_req_duration: ['p(95)<500'],
    },
};

export default function () {
    group('Health check', () => {
        const res = http.get(`${BASE_URL}/health`);
        check(res, {
            'health status 200': (r) => r.status === 200,
            'health body ok': (r) => r.json().status === 'ok',
        });
    });

    group('Get profile', () => {
        const res = http.get(`${BASE_URL}/api/v1/profile/kratos-user-123`);
        check(res, {
            'profile status 200 or 404': (r) => [200, 404].includes(r.status),
        });
    });

    group('Get profile - not found', () => {
        const res = http.get(`${BASE_URL}/api/v1/profile/nonexistent-user`);
        check(res, {
            'not found status 404': (r) => r.status === 404,
        });
    });

    sleep(0.5);
}
```

#### 7.2 k6 Load Tests
**Directory:** `k6/load/`

Each file targets a specific workload pattern. Example:

**File:** `k6/load/profile-read.js`

```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export const options = {
    stages: [
        { duration: '30s', target: 20 },   // Ramp up to 20 VUs
        { duration: '1m',  target: 20 },   // Hold at 20 VUs
        { duration: '30s', target: 50 },   // Ramp up to 50 VUs
        { duration: '1m',  target: 50 },   // Hold at 50 VUs
        { duration: '30s', target: 0 },    // Ramp down
    ],
    thresholds: {
        http_req_duration: ['p(95)<200', 'p(99)<500'],
        http_req_failed: ['rate<0.01'],
    },
};

export default function () {
    const res = http.get(`${BASE_URL}/api/v1/profile/kratos-user-123`);
    check(res, {
        'status is 200 or 404': (r) => [200, 404].includes(r.status),
        'response time < 500ms': (r) => r.timings.duration < 500,
    });

    sleep(0.1);
}
```

Run against the service deployed in k3d:
```bash
# Port-forward the service
kubectl port-forward svc/mathtrail-profile 8080:8080 -n mathtrail &

# Run all scenario tests
just k6-scenarios

# Run all load tests
just k6-load

# Run a specific test file
just k6-run k6/scenarios/profile-crud.js
just k6-run k6/load/profile-read.js

# Run with custom base URL
k6 run -e BASE_URL=http://localhost:8080 k6/load/profile-read.js
```

---

## Summary Checklist

### Stage 1: Project Setup
- [x] DevContainer created with Go, Skaffold, kubectl, Helm, Dapr CLI, Delve
- [x] Go project structure created
- [x] Dependencies installed
- [x] Skaffold config with mathtrail-infra-local dependency
- [x] Configuration system implemented

### Stage 2: Database
- [x] SQL migration file created (`migrations/001_init.sql`)
- [x] GORM data models defined (`internal/profile/model.go`)
- [x] Repository interface + implementation (`internal/profile/repository.go`)
- [x] Database connection (`internal/database/database.go`)
- [x] DI container wiring (`internal/app/container.go`)

### Stage 3: API
- [x] GET /profile/{userId} endpoint implemented
- [x] Redis cache-aside pattern for profile reads
- [x] Cache invalidation on profile create/update
- [x] Error handling in place
- [x] Request validation implemented
- [x] Swagger documentation added

### Stage 4: Events
- [x] Dapr components configured
- [x] user-registered handler implemented
- [x] task-solved handler implemented
- [x] Event models defined

### Stage 5: Observability
- [x] Logging configured
- [x] Baseline metrics collection enabled (Dapr)
- [x] (skip) Custom business metrics (deferred)

### Stage 6: Documentation & Deployment
- [ ] API documentation complete
- [ ] Example payloads provided
- [ ] Deployment guides written
- [ ] Helm charts configured
- [ ] README updated

### Stage 7: k6 Integration Testing
- [ ] k6 scenario tests passing
- [ ] k6 load tests passing (p95 < 20ms)

---

## Next Steps

1. Start with **Stage 1: Project Setup**
2. Move through stages sequentially
3. Each stage should be independently testable
4. Commit changes after completing each stage
5. Run tests before moving to the next stage
