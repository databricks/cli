#!/bin/bash
set -euo pipefail

cd "$(dirname "$0")"

mkdir -p build

for arch in amd64 arm64; do
    echo "Building for $arch..."
    docker build --build-arg ARCH=$arch -t databricks-cli:$arch -f Dockerfile .
    docker save databricks-cli:$arch -o build/cli-$arch.tar
done

echo "Done. Images saved to build/"
