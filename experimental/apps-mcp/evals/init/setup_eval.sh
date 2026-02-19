#!/bin/bash
set -e

echo "=== Apps-MCP Eval Setup ==="
echo "Python version: $(python --version)"

# Install Node.js (required for local npm install/build/test)
echo "Installing Node.js..."
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt-get install -y nodejs

echo "Node version: $(node --version)"
echo "npm version: $(npm --version)"

# Install Python dependencies
pip install fire mlflow

echo "=== Setup complete ==="
