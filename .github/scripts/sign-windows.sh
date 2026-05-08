#!/usr/bin/env bash
#
# Sign a Windows binary using jsign with Azure Key Vault.
# Called as a goreleaser post-hook for every built binary.
#
# Skips signing when:
#   - The binary is not a .exe (unix builds)
#   - Not running in CI (local builds)
#
# Expected environment variables (set by the "cli" job in release-build.yml):
#   JSIGN_JAR          - Path to the jsign jar file
#   AZURE_VAULT_TOKEN  - Azure Key Vault access token
#
# https://github.com/ebourg/jsign

set -euo pipefail

binary="$1"

# Skip non-Windows binaries.
[[ "$binary" == *.exe ]] || exit 0

# Skip when not running in CI.
[[ "${CI:-}" == "true" ]] || exit 0

# Verify required environment variables.
if [[ -z "${JSIGN_JAR:-}" ]]; then
  echo "ERROR: JSIGN_JAR is not set" >&2
  exit 1
fi
if [[ -z "${AZURE_VAULT_TOKEN:-}" ]]; then
  echo "ERROR: AZURE_VAULT_TOKEN is not set" >&2
  exit 1
fi

java -jar "${JSIGN_JAR}" \
  --storetype AZUREKEYVAULT \
  --keystore deco-sign \
  --storepass "${AZURE_VAULT_TOKEN}" \
  --alias deco-sign \
  --tsaurl http://timestamp.digicert.com \
  --tsmode RFC3161 \
  "$binary"
