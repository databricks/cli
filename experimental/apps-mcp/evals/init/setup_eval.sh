#!/bin/bash
set -e

echo "=== Apps-MCP Eval Setup ==="
echo "Python version: $(python --version)"

# Install Node.js (required for klaudbiusz eval)
echo "Installing Node.js..."
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt-get install -y nodejs

echo "Node version: $(node --version)"
echo "npm version: $(npm --version)"

# Install Docker (required for --no-dagger mode)
echo "Installing Docker..."
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
rm get-docker.sh

# Configure Docker to use vfs storage driver (works without privileged mode)
echo "Configuring Docker with vfs storage driver..."
sudo mkdir -p /etc/docker
cat <<EOF | sudo tee /etc/docker/daemon.json
{
  "storage-driver": "vfs"
}
EOF

# Stop any existing Docker daemon
sudo systemctl stop docker 2>/dev/null || true
sudo pkill dockerd 2>/dev/null || true
sleep 2

# Start Docker daemon
echo "Starting Docker daemon..."
sudo dockerd --storage-driver=vfs &
sleep 10

# Verify Docker is running
echo "Docker version: $(docker --version)"
sudo docker info || echo "Warning: Docker daemon may not be fully started"

# Allow non-root user to run docker
sudo usermod -aG docker $(whoami) || true
sudo chmod 666 /var/run/docker.sock || true

# Pre-pull the node image to speed up evaluation
echo "Pre-pulling node:20-alpine image..."
docker pull node:20-alpine || echo "Warning: Could not pre-pull image"

# Install Python dependencies
pip install fire mlflow

echo "=== Setup complete ==="
