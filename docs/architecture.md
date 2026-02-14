# MathTrail Profile Service Architecture

## Overview

The MathTrail Profile Service is a microservice responsible for managing user profiles within the MathTrail ecosystem. It is designed for cloud-native, event-driven environments and integrates with other MathTrail services via REST APIs and Dapr pub/sub (Kafka).

## Key Components

- **Gin**: HTTP web framework for REST API endpoints.
- **GORM**: ORM for PostgreSQL database access.
- **PostgreSQL**: Primary data store for user profiles, skills, and progress.
- **Redis**: Caching layer for fast profile reads, using a cache-aside pattern.
- **Dapr**: Service mesh for pub/sub, sidecar observability, and distributed tracing.
- **Kafka**: Message broker for event-driven communication (user-registered, task-solved events).
- **CloudEvents**: Event format for interoperability.
- **Swagger/OpenAPI**: API documentation and interactive UI.

## Architecture Diagram

```
sequenceDiagram
    participant User
    participant API as Profile API (Gin)
    participant Redis
    participant DB as PostgreSQL
    participant Dapr
    participant Kafka
    participant EventSrc as Event Source

    User->>API: REST API call (GET /profile/{userId})
    API->>Redis: Check cache
    alt Cache hit
        Redis-->>API: Return profile
        API-->>User: Return profile
    else Cache miss
        API->>DB: Query profile
        DB-->>API: Return profile
        API->>Redis: Populate cache
        API-->>User: Return profile
    end

    EventSrc->>Dapr: Publish user-registered event (Kafka)
    Dapr->>API: Deliver event (CloudEvent)
    API->>DB: Create profile
    API->>Redis: Invalidate cache
    API-->>Dapr: Ack event
```

## Data Flow

1. **Profile Retrieval**: API checks Redis cache first; on miss, queries PostgreSQL and populates cache.
2. **Profile Creation/Update**: Triggered by user-registered or task-solved events via Dapr pub/sub. Mutations invalidate the cache.
3. **Observability**: Structured logging (zap), metrics and tracing via Dapr sidecar, GORM query logging.

## Deployment

- Runs in Kubernetes (k3d for local dev)
- Managed via Helm charts
- Local dev loop with Skaffold and DevContainer
- Dapr sidecar for pub/sub, metrics, and tracing

## Extensibility

- New events can be handled by adding Dapr subscriptions and event handlers.
- Additional endpoints can be added via Gin controllers.
- Business logic is encapsulated in the service layer for testability.

---

For more details, see the main README and IMPLEMENTATION.md.