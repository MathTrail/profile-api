# MathTrail Profile Service

## Overview

User Profile service for the MathTrail platform. Manages user profiles, handles events
(user-registered, task-solved via Kafka pub/sub), and exposes profile data via REST API.

**Language:** Go 1.26.0
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

## Key Files

| File | Purpose |
|------|---------|
| `internal/config/config.go` | Config from env vars — NO DB passwords, NO Redis password |
| `internal/secrets/vault.go` | Vault HTTP API client for fetching secrets |
| `internal/database/database.go` | GORM connection (accepts explicit DSN string) |
| `internal/app/container.go` | DI container — fetches secrets via Vault before creating connections |
| `internal/profile/` | domain model, repository, service, controller |
| `internal/kafka/subscriber.go` | Pub/sub event handlers (user-registered, task-solved) |
| `internal/cache/profile.go` | Redis-backed profile cache |
| `infra/helm/mathtrail-profile/` | Helm chart (uses `mathtrail-service-lib`) |
| `infra/helm/values-dev.yaml` | Dev environment values — no hardcoded credentials |

## Architecture

- **DB access:** GORM via PgBouncer (`postgres-pgbouncer:6432`). DSN built at startup
  with credentials fetched from Vault at startup.
- **Migrations:** Direct PostgreSQL (`postgres-postgresql:5432`), using Bitnami superuser K8s Secret.
- **Cache:** Redis with password fetched from Vault KV at startup.
- **Secrets rule:** ONLY via Vault HTTP API (`internal/secrets/vault.go`). No env var passwords, no ESO ExternalSecrets.
  - DB creds: Vault Database Secrets Engine `creds/profile-api-role` → `{username, password}`
  - Redis password: Vault KV `local/mathtrail-profile` → `{redis-password}`
- **Pub/Sub:** Kafka consumer calls app HTTP endpoints for event handling.

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
  Always fetch via `secrets.GetVaultSecretWithRetry()`.

## External Dependencies

| Repo | Purpose |
|------|---------|
| `mathtrail-charts` | Hosts `mathtrail-service-lib` library chart |
| `mathtrail-infra-local-k3s` | k3d cluster setup |
| `mathtrail-infra` | Vault components deployed here |

## Vault Secret Paths (local dev)

| Secret | Vault path | Keys |
|--------|------------|------|
| DB creds | `creds/profile-api-role` | `username`, `password` |
| Redis password | `local/mathtrail-profile` | `redis-password` |

## Commit Convention

Use Conventional Commits: `feat(profile):`, `fix(profile):`, `chore(profile):`
