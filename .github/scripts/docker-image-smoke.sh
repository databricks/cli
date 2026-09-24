#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: $0 IMAGE PLATFORM" >&2
  exit 2
fi

image=$1
platform=$2
repo=$(pwd)
commit=$(git rev-parse HEAD)
config=$(mktemp)
trap 'rm -f "$config"' EXIT
cat > "$config" <<EOF
experiment_name: docker-image-smoke
command: python train.py
compute:
  accelerator_type: GPU_1xH100
  num_accelerators: 1
code_source:
  type: snapshot
  snapshot:
    root_path: /repo/acceptance/air/run-submit
    git:
      commit: $commit
EOF

docker run --rm --platform "linux/$platform" \
  -v "$repo:/repo:ro" \
  -v "$config:/run.yaml:ro" \
  --entrypoint /app/databricks \
  "$image" air run -f /run.yaml --dry-run

docker run --rm --platform "linux/$platform" \
  -v "$repo:/repo:ro" \
  --entrypoint /bin/sh \
  "$image" -ec '
    git -C /repo archive --format=tar HEAD >/dev/null
    test -z "$(git -C /repo status --porcelain)"
  '
