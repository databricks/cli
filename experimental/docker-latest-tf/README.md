# Docker with Latest Terraform

Docker image using Terraform 1.14.0 and Databricks CLI v0.279.0.

## Build

```bash
./build.sh
```

Images are output to `build/` directory.

## Upload to GitHub Container Registry

```bash
# Login to ghcr.io
docker login ghcr.io -u <username>

# Load and tag images
docker load -i build/cli-amd64.tar
docker load -i build/cli-arm64.tar

# Tag for registry
docker tag databricks-cli:amd64 ghcr.io/databricks/cli:latest-tf-amd64
docker tag databricks-cli:arm64 ghcr.io/databricks/cli:latest-tf-arm64

# Push
docker push ghcr.io/databricks/cli:<cli-version>-amd64-tf-<tf-version>-experimental
docker push ghcr.io/databricks/cli:<cli-version>-amd64-tf-<tf-version>-experimental


## Test

```bash
# Load the image
docker load -i build/cli-arm64.tar  # or cli-amd64.tar

# Run a bundle deploy (mount your bundle directory and credentials)
docker run --rm \
  -v ~/.databrickscfg:/root/.databrickscfg:ro \
  -v /path/to/your/bundle:/bundle \
  -w /bundle \
  databricks-cli:arm64 bundle deploy
```
