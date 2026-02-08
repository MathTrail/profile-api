# MathTrail Profile Service

set shell := ["bash", "-c"]

NAMESPACE := "mathtrail"
SERVICE := "mathtrail-profile"

# Build the Go binary
build:
    go build -o bin/server ./cmd/server

# Run all tests
test:
    go test ./... -v

# Run integration tests
test-integration:
    go test ./... -v -tags=integration

# Generate test coverage report
test-coverage:
    go test ./... -coverprofile=coverage.out
    go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report: coverage.html"

# Run all k6 scenario tests
k6-scenarios:
    #!/bin/bash
    set -e
    for f in k6/scenarios/*.js; do
        echo "▶ Running $f..."
        k6 run "$f"
    done

# Run all k6 load tests
k6-load:
    #!/bin/bash
    set -e
    for f in k6/load/*.js; do
        echo "▶ Running $f..."
        k6 run "$f"
    done

# Run a specific k6 test file
k6-run FILE:
    k6 run {{ FILE }}

# Skaffold dev mode (build + deploy + watch)
dev:
    skaffold dev

# One-shot build and deploy
deploy:
    skaffold run

# Remove from cluster
delete:
    skaffold delete

# View pod logs
logs:
    kubectl logs -l app={{ SERVICE }} -n {{ NAMESPACE }} -f

# Build Docker image locally
docker-build:
    docker build -t {{ SERVICE }}:latest .

# Generate Swagger docs
swagger:
    swag init -g cmd/server/main.go

# Run database migration
migrate:
    #!/bin/bash
    set -e
    echo "Running migrations..."
    kubectl exec -n {{ NAMESPACE }} deploy/postgres-postgresql -- \
        psql -U postgres -d profile -f /dev/stdin < migrations/001_init.sql
    echo "✅ Migrations applied!"
