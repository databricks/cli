#!/usr/bin/env bash
#
# Query Databricks SDK documentation from the command line.
# Usage: ./query-sdk-docs.sh <query> [category] [service] [limit]
#
# Examples:
#   ./query-sdk-docs.sh "create job"
#   ./query-sdk-docs.sh "JobSettings" types
#   ./query-sdk-docs.sh "list" methods jobs
#   ./query-sdk-docs.sh "cluster" methods compute 20
#
# Categories: methods, types, enums, services
# Services: jobs, clusters, pipelines, workspace, etc.

set -euo pipefail

QUERY="${1:-}"
CATEGORY="${2:-}"
SERVICE="${3:-}"
LIMIT="${4:-10}"

if [[ -z "$QUERY" ]]; then
    echo "Usage: $0 <query> [category] [service] [limit]"
    echo ""
    echo "Examples:"
    echo "  $0 'create job'              # Search for 'create job'"
    echo "  $0 'JobSettings' types       # Search types for 'JobSettings'"
    echo "  $0 'list' methods jobs       # Search jobs service methods for 'list'"
    echo ""
    echo "Categories: methods, types, enums, services"
    exit 1
fi

# Build the JSON input for the MCP tool
build_json_input() {
    local json="{\"query\": \"$QUERY\""

    if [[ -n "$CATEGORY" ]]; then
        json+=", \"category\": \"$CATEGORY\""
    fi

    if [[ -n "$SERVICE" ]]; then
        json+=", \"service\": \"$SERVICE\""
    fi

    json+=", \"limit\": $LIMIT}"
    echo "$json"
}

# Try to find the SDK docs index file for direct search
SDK_DOCS_INDEX="${SDK_DOCS_INDEX:-}"
if [[ -z "$SDK_DOCS_INDEX" ]]; then
    # Look for the index in common locations
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    CLI_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

    POSSIBLE_PATHS=(
        "$CLI_ROOT/experimental/aitools/lib/providers/sdkdocs/sdk_docs_index.json"
        "./sdk_docs_index.json"
    )

    for path in "${POSSIBLE_PATHS[@]}"; do
        if [[ -f "$path" ]]; then
            SDK_DOCS_INDEX="$path"
            break
        fi
    done
fi

# If we have jq and the index file, do a direct search
if command -v jq &>/dev/null && [[ -n "$SDK_DOCS_INDEX" && -f "$SDK_DOCS_INDEX" ]]; then
    echo "Searching SDK docs for: $QUERY"
    echo "---"

    QUERY_LOWER=$(echo "$QUERY" | tr '[:upper:]' '[:lower:]')

    # Search methods
    if [[ -z "$CATEGORY" || "$CATEGORY" == "methods" ]]; then
        echo ""
        echo "## Methods"
        jq -r --arg q "$QUERY_LOWER" --arg svc "$SERVICE" '
            .services | to_entries[] |
            select($svc == "" or .key == $svc) |
            .key as $service |
            .value.methods // {} | to_entries[] |
            select(
                (.key | ascii_downcase | contains($q)) or
                (.value.description // "" | ascii_downcase | contains($q))
            ) |
            "- \($service).\(.key): \(.value.description // "No description")[signature: \(.value.signature // "N/A")]"
        ' "$SDK_DOCS_INDEX" 2>/dev/null | head -n "$LIMIT" || echo "  (no matches)"
    fi

    # Search types
    if [[ -z "$CATEGORY" || "$CATEGORY" == "types" ]]; then
        echo ""
        echo "## Types"
        jq -r --arg q "$QUERY_LOWER" '
            .types // {} | to_entries[] |
            select(
                (.key | ascii_downcase | contains($q)) or
                (.value.description // "" | ascii_downcase | contains($q))
            ) |
            "- \(.key): \(.value.description // "No description")"
        ' "$SDK_DOCS_INDEX" 2>/dev/null | head -n "$LIMIT" || echo "  (no matches)"
    fi

    # Search enums
    if [[ -z "$CATEGORY" || "$CATEGORY" == "enums" ]]; then
        echo ""
        echo "## Enums"
        jq -r --arg q "$QUERY_LOWER" '
            .enums // {} | to_entries[] |
            select(
                (.key | ascii_downcase | contains($q)) or
                (.value.description // "" | ascii_downcase | contains($q))
            ) |
            "- \(.key): \(.value.values // [] | join(", "))"
        ' "$SDK_DOCS_INDEX" 2>/dev/null | head -n "$LIMIT" || echo "  (no matches)"
    fi
else
    # Fallback: show how to use the MCP tool
    echo "SDK docs index not found locally. Use the MCP tool instead:"
    echo ""
    echo "databricks_query_sdk_docs with input:"
    build_json_input
    echo ""
    echo "Or set SDK_DOCS_INDEX environment variable to point to sdk_docs_index.json"
fi
