#!/bin/bash
set -e

# Create skaffold.env symlink so Skaffold picks up shared variables
ln -sf "$HOME/.env.shared" "$PWD/skaffold.env"

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
