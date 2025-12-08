#!/bin/bash
set -e

echo "=== Setting up generation environment ==="

# Install Dagger (required for klaudbiusz container orchestration)
echo "Installing Dagger..."
curl -fsSL https://dl.dagger.io/dagger/install.sh | sh
export PATH=$PATH:/root/.local/bin

# Install Python dependencies for klaudbiusz
echo "Installing Python dependencies..."
pip install --quiet dagger-io fire tqdm python-dotenv claude-agent-sdk litellm joblib tenacity

echo "=== Setup complete ==="
