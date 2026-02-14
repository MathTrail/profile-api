# mathtrail-profile
Source of truth for student learning state — stores knowledge graph, skill history, XP, and preferences for personalized adaptive learning.

## Mission & Responsibilities
- Maintain student Knowledge Graph (skills, mastery levels, connections)
- Track attempt history (problems solved, time spent, errors)
- Calculate and update XP / level progression
- Provide profile data to Mentor service on demand
- Auto-create profiles on Ory Kratos registration webhook (CloudEvents)
- Cache profiles in Redis for fast reads

## Tech Stack
- **Language**: Go 1.25.7
- **Framework**: Gin (HTTP), GORM (ORM)
- **Database**: PostgreSQL (persistent), Redis (cache)
- **Events**: Dapr pub/sub over Kafka (CloudEvents)
- **API Docs**: Swagger/OpenAPI
- **Testing**: Go testing + testify, Grafana k6

## API & Communication (Dapr)
- **Inbound**: REST API (GET/PUT /profiles/{id}), Kratos webhook (POST /webhooks/registration)
- **Outbound**: None (passive data provider)
- **Subscribes**: `identity.registration.completed` (auto-create profile)
- **Subscribes**: `task.attempt.completed` (update skills/XP)

## Data Persistence
- **PostgreSQL**: profiles, skills, attempt_history tables
- **Redis**: Profile cache with TTL, invalidated on mutations

## Secrets
- `DB_PASSWORD` — PostgreSQL password
- `REDIS_PASSWORD` — Redis password
- Vault path: `secret/data/{env}/mathtrail-profile/`

## Infrastructure
Standard `infra/` layout:
- `infra/helm/` — Helm chart + environment overlays (dev, on-prem, cloud)
- `infra/terraform/` — Database module (K8s pod vs managed RDS)
- `infra/ansible/` — On-prem node preparation

## Development
- `just dev` — Skaffold dev loop
- `just test` — Go unit tests
- `just swagger` — regenerate API docs

## Roadmap
1. Implement Knowledge Graph data model (skills → topics → prerequisites)
2. Add Dapr pub/sub handler for `task.attempt.completed` events
3. Build Redis cache layer with write-through invalidation
