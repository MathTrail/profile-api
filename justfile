# MathTrail Profile Service

set shell := ["bash", "-c"]
set dotenv-load := true
set dotenv-path := env("HOME") + "/.env.shared"

NAMESPACE := "mathtrail"
SERVICE := "mathtrail-profile"
CHART_NAME := "mathtrail-profile"

# -- Portable Image Build (buildctl → buildah) --------------------------------

# Build and push a container image using the best available builder
# CI (K3s runner): buildctl talks to buildkitd sidecar
# Local dev: buildah builds rootlessly
[private]
build-image tag:
    #!/bin/bash
    set -e
    if command -v buildctl &>/dev/null; then
        echo "🔨 Building with BuildKit..."
        buildctl build \
            --frontend=dockerfile.v0 \
            --local context=. \
            --local dockerfile=. \
            --output type=image,name={{tag}},push=true,registry.insecure=true \
            --export-cache type=inline \
            --import-cache type=registry,ref={{tag}}
    else
        echo "🔨 Building with Buildah..."
        buildah bud -t {{tag}} .
        buildah push {{tag}}
    fi

# -- Development ---------------------------------------------------------------

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
    kubectl logs -l app.kubernetes.io/name={{ SERVICE }} -n {{ NAMESPACE }} -f

# Check deployment status
status:
    kubectl get pods -n {{ NAMESPACE }} -l app.kubernetes.io/name={{ SERVICE }}

# Build container image locally (uses kaniko → buildah → docker fallback)
image-build:
    just build-image {{ SERVICE }}:latest

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
    echo "Migrations applied!"

# -- K6 Tests ------------------------------------------------------------------

# Run all k6 scenario tests
k6-scenarios:
    #!/bin/bash
    set -e
    for f in k6/scenarios/*.js; do
        echo "Running $f..."
        k6 run "$f"
    done

# Run all k6 load tests
k6-load:
    #!/bin/bash
    set -e
    for f in k6/load/*.js; do
        echo "Running $f..."
        k6 run "$f"
    done

# Run a specific k6 test file
k6-run FILE:
    k6 run {{ FILE }}

# -- Chart Release -------------------------------------------------------------

# Package and publish chart to mathtrail-charts
release-chart:
    #!/bin/bash
    set -e
    CHART_DIR="infra/helm/{{ CHART_NAME }}"
    VERSION=$(grep '^version:' "$CHART_DIR/Chart.yaml" | awk '{print $2}')
    echo "Packaging {{ CHART_NAME }} v${VERSION}..."
    helm package "$CHART_DIR" --destination /tmp/mathtrail-charts

    CHARTS_REPO_DIR="/tmp/mathtrail-charts-repo"
    rm -rf "$CHARTS_REPO_DIR"
    git clone git@github.com:MathTrail/charts.git "$CHARTS_REPO_DIR"
    cp /tmp/mathtrail-charts/{{ CHART_NAME }}-*.tgz "$CHARTS_REPO_DIR/charts/"
    cd "$CHARTS_REPO_DIR"
    helm repo index ./charts \
        --url ${CHARTS_REPO}
    git add charts/
    git commit -m "chore: release {{ CHART_NAME }} v${VERSION}"
    git push
    echo "Published {{ CHART_NAME }} v${VERSION} to mathtrail-charts"

# -- Terraform -----------------------------------------------------------------

# Initialize Terraform for an environment
tf-init ENV:
    cd infra/terraform/environments/{{ ENV }} && terraform init

# Plan Terraform changes
tf-plan ENV:
    cd infra/terraform/environments/{{ ENV }} && terraform plan

# Apply Terraform changes
tf-apply ENV:
    cd infra/terraform/environments/{{ ENV }} && terraform apply

# -- On-prem Node Preparation -------------------------------------------------

# Prepare an Ubuntu node for on-prem deployment
prepare-node IP:
    cd infra/ansible && ansible-playbook \
        -i "{{ IP }}," \
        playbooks/setup.yml
