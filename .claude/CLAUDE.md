# MathTrail Profile Service

## Overview

User Profile service for the MathTrail platform. Manages user profiles, handles events
(user-registered, task-solved via Dapr pub/sub), and exposes profile data via REST API.

**Language:** Go 1.25.7
**Port:** 8080
**Cluster:** k3d `mathtrail-dev`, namespace `mathtrail`
**KUBECONFIG:** `/home/vscode/.kube/k3d-mathtrail-dev.yaml`

## Tech Stack

| Layer | Library |
|-------|---------|
| HTTP | `github.com/gin-gonic/gin` |
| Database | GORM + pgx driver via PgBouncer |
| Cache | Redis (`github.com/redis/go-redis/v9`) |
| Config | `os.LookupEnv` (no external config library) |
| Logging | `go.uber.org/zap` |
| Dapr | HTTP pub/sub subscriptions (sidecar calls app; no Dapr Go SDK) |

## Key Files

| File | Purpose |
|------|---------|
| `internal/config/config.go` | Config from env vars — NO DB passwords, NO Redis password |
| `internal/secrets/dapr.go` | Dapr HTTP API client for fetching secrets from Vault |
| `internal/database/database.go` | GORM connection (accepts explicit DSN string) |
| `internal/app/container.go` | DI container — fetches secrets via Dapr before creating connections |
| `internal/profile/` | domain model, repository, service, controller |
| `internal/dapr/subscriber.go` | Pub/sub event handlers (user-registered, task-solved) |
| `internal/cache/profile.go` | Redis-backed profile cache |
| `infra/helm/mathtrail-profile/` | Helm chart (uses `mathtrail-service-lib`) |
| `infra/helm/values-dev.yaml` | Dev environment values — no hardcoded credentials |

## Architecture

- **DB access:** GORM via PgBouncer (`postgres-pgbouncer:6432`). DSN built at startup
  with credentials fetched from Vault via the Dapr sidecar (`vault-db` component).
- **Migrations:** Direct PostgreSQL (`postgres-postgresql:5432`), using Bitnami superuser K8s Secret.
- **Cache:** Redis with password fetched from Vault KV (`vault` component) at startup.
- **Secrets rule:** ONLY via Dapr HTTP API (`internal/secrets/dapr.go`). No env var passwords, no ESO ExternalSecrets.
  - DB creds: `GetDaprSecretWithRetry(ctx, daprAddr, "vault-db", "creds/profile-api-role", 10)` → `{username, password}`
  - Redis password: `GetDaprSecretWithRetry(ctx, daprAddr, "vault", "local/mathtrail-profile", 10)` → `{redis-password}`
- **Dapr App ID:** `mathtrail-profile`
- **Pub/Sub:** Dapr sidecar calls app HTTP endpoints — no Dapr Go SDK needed for subscriptions.

## Development Workflow

```bash
just dev         # Skaffold dev mode: hot-reload + port-forward
just deploy      # One-time build and deploy
just delete      # Remove from cluster
just test        # go test ./... -v
just logs        # View pod logs
just status      # Check pods and services
```

## Development Standards

- Follow Clean Architecture: Domain → Repository → Service → Controller
- Handle errors explicitly — never ignore error returns
- All comments in English
- **Secrets rule:** Credentials are NEVER hardcoded in values files or passed as env var passwords.
  Always fetch via `secrets.GetDaprSecretWithRetry()`.

## External Dependencies

| Repo | Purpose |
|------|---------|
| `mathtrail-charts` | Hosts `mathtrail-service-lib` library chart |
| `mathtrail-infra-local-k3s` | k3d cluster setup |
| `mathtrail-infra` | Vault + Dapr components (`vault-db`, `vault`) deployed here |

## Vault / Dapr Secret Paths (local dev)

| Secret | Dapr component | Vault path | Keys |
|--------|---------------|------------|------|
| DB creds | `vault-db` | `creds/profile-api-role` | `username`, `password` |
| Redis password | `vault` | `local/mathtrail-profile` | `redis-password` |

## Commit Convention

Use Conventional Commits: `feat(profile):`, `fix(profile):`, `chore(profile):`
