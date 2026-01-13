#!/usr/bin/env python3
"""
Discover available tools and data sources for Databricks agents.

This script scans for:
- Unity Catalog functions (potential tools)
- Unity Catalog tables (data sources)
- Vector search indexes (RAG data sources)
- Genie spaces (conversational data access)
- Custom MCP servers (mcp-* packages)
"""

import json
import subprocess
import sys
from pathlib import Path
from typing import Any, Dict, List

from databricks.sdk import WorkspaceClient


def run_databricks_cli(args: List[str]) -> str:
    """Run databricks CLI command and return output."""
    try:
        result = subprocess.run(
            ["databricks"] + args,
            capture_output=True,
            text=True,
            check=True,
        )
        return result.stdout
    except subprocess.CalledProcessError as e:
        print(f"Error running databricks CLI: {e.stderr}", file=sys.stderr)
        return ""


def discover_uc_functions(w: WorkspaceClient, catalog: str = None) -> List[Dict[str, Any]]:
    """Discover Unity Catalog functions that could be used as tools."""
    functions = []

    try:
        catalogs = [catalog] if catalog else [c.name for c in w.catalogs.list()]

        for cat in catalogs:
            try:
                schemas = list(w.schemas.list(catalog_name=cat))
                for schema in schemas:
                    schema_name = f"{cat}.{schema.name}"
                    try:
                        funcs = list(w.functions.list(catalog_name=cat, schema_name=schema.name))
                        for func in funcs:
                            functions.append({
                                "type": "uc_function",
                                "name": func.full_name,
                                "catalog": cat,
                                "schema": schema.name,
                                "function_name": func.name,
                                "comment": func.comment,
                                "routine_definition": getattr(func, "routine_definition", None),
                            })
                    except Exception as e:
                        # Skip schemas we can't access
                        continue
            except Exception as e:
                # Skip catalogs we can't access
                continue

    except Exception as e:
        print(f"Error discovering UC functions: {e}", file=sys.stderr)

    return functions


def discover_uc_tables(w: WorkspaceClient, catalog: str = None, schema: str = None) -> List[Dict[str, Any]]:
    """Discover Unity Catalog tables that could be queried."""
    tables = []

    try:
        catalogs = [catalog] if catalog else [c.name for c in w.catalogs.list()]

        for cat in catalogs:
            if cat in ["__databricks_internal", "system"]:
                continue

            try:
                schemas = [schema] if schema else [s.name for s in w.schemas.list(catalog_name=cat)]
                for sch in schemas:
                    if sch == "information_schema":
                        continue

                    try:
                        tbls = list(w.tables.list(catalog_name=cat, schema_name=sch))
                        for tbl in tbls:
                            # Get column info
                            columns = []
                            if hasattr(tbl, "columns") and tbl.columns:
                                columns = [
                                    {"name": col.name, "type": col.type_name.value if hasattr(col.type_name, "value") else str(col.type_name)}
                                    for col in tbl.columns
                                ]

                            tables.append({
                                "type": "uc_table",
                                "name": tbl.full_name,
                                "catalog": cat,
                                "schema": sch,
                                "table_name": tbl.name,
                                "table_type": tbl.table_type.value if tbl.table_type else None,
                                "comment": tbl.comment,
                                "columns": columns,
                            })
                    except Exception as e:
                        # Skip schemas we can't access
                        continue
            except Exception as e:
                # Skip catalogs we can't access
                continue

    except Exception as e:
        print(f"Error discovering UC tables: {e}", file=sys.stderr)

    return tables


def discover_vector_search_indexes(w: WorkspaceClient) -> List[Dict[str, Any]]:
    """Discover Vector Search indexes for RAG applications."""
    indexes = []

    try:
        # List all vector search endpoints
        endpoints = list(w.vector_search_endpoints.list_endpoints())

        for endpoint in endpoints:
            try:
                # List indexes for each endpoint
                endpoint_indexes = list(w.vector_search_indexes.list_indexes(endpoint_name=endpoint.name))
                for idx in endpoint_indexes:
                    indexes.append({
                        "type": "vector_search_index",
                        "name": idx.name,
                        "endpoint": endpoint.name,
                        "primary_key": idx.primary_key,
                        "index_type": idx.index_type.value if idx.index_type else None,
                        "status": idx.status.state.value if idx.status and idx.status.state else None,
                    })
            except Exception as e:
                # Skip endpoints we can't access
                continue

    except Exception as e:
        print(f"Error discovering vector search indexes: {e}", file=sys.stderr)

    return indexes


def discover_genie_spaces(w: WorkspaceClient) -> List[Dict[str, Any]]:
    """Discover Genie spaces for conversational data access."""
    spaces = []

    try:
        # Use CLI since SDK may not have full Genie support
        output = run_databricks_cli(["genie", "list-spaces", "--output", "json"])
        if output:
            data = json.loads(output)
            spaces_list = data.get("spaces", [])
            for space in spaces_list:
                spaces.append({
                    "type": "genie_space",
                    "id": space.get("space_id"),
                    "name": space.get("display_name"),
                    "description": space.get("description"),
                })
    except Exception as e:
        print(f"Error discovering Genie spaces: {e}", file=sys.stderr)

    return spaces


def discover_mcp_servers() -> List[Dict[str, Any]]:
    """Discover custom MCP servers (Python packages starting with mcp-)."""
    mcp_servers = []

    try:
        # Check if uv is available
        result = subprocess.run(
            ["uv", "pip", "list", "--format", "json"],
            capture_output=True,
            text=True,
        )

        if result.returncode == 0:
            packages = json.loads(result.stdout)
            for pkg in packages:
                name = pkg.get("name", "")
                if name.startswith("mcp-") or "mcp" in name.lower():
                    mcp_servers.append({
                        "type": "mcp_server_package",
                        "package": name,
                        "version": pkg.get("version"),
                    })
    except Exception as e:
        print(f"Error discovering MCP servers: {e}", file=sys.stderr)

    return mcp_servers


def discover_custom_mcp_servers(w: WorkspaceClient) -> List[Dict[str, Any]]:
    """Discover custom MCP servers deployed as Databricks apps."""
    custom_servers = []

    try:
        # List all apps and filter for those starting with mcp-
        apps = w.apps.list()
        for app in apps:
            if app.name and app.name.startswith("mcp-"):
                custom_servers.append({
                    "type": "custom_mcp_server",
                    "name": app.name,
                    "url": app.url,
                    "status": app.app_status.state.value if app.app_status and app.app_status.state else None,
                    "description": app.description,
                })
    except Exception as e:
        print(f"Error discovering custom MCP servers: {e}", file=sys.stderr)

    return custom_servers


def discover_external_mcp_servers(w: WorkspaceClient) -> List[Dict[str, Any]]:
    """Discover external MCP servers configured via Unity Catalog connections."""
    external_servers = []

    try:
        # List all connections and filter for MCP connections
        connections = w.connections.list()
        for conn in connections:
            # Check if this is an MCP connection
            if conn.options and conn.options.get("is_mcp_connection") == "true":
                external_servers.append({
                    "type": "external_mcp_server",
                    "name": conn.name,
                    "connection_type": conn.connection_type.value if hasattr(conn.connection_type, "value") else str(conn.connection_type),
                    "comment": conn.comment,
                    "full_name": conn.full_name,
                })
    except Exception as e:
        print(f"Error discovering external MCP servers: {e}", file=sys.stderr)

    return external_servers


def format_output_markdown(results: Dict[str, List[Dict[str, Any]]]) -> str:
    """Format discovery results as markdown."""
    lines = ["# Agent Tools and Data Sources Discovery\n"]

    # UC Functions
    functions = results.get("uc_functions", [])
    if functions:
        lines.append(f"## Unity Catalog Functions ({len(functions)})\n")
        lines.append("These can be used as agent tools via MCP servers.\n")
        for func in functions[:10]:  # Show first 10
            lines.append(f"- `{func['name']}`")
            if func.get("comment"):
                lines.append(f"  - {func['comment']}")
        if len(functions) > 10:
            lines.append(f"\n*...and {len(functions) - 10} more*\n")
        lines.append("")

    # UC Tables
    tables = results.get("uc_tables", [])
    if tables:
        lines.append(f"## Unity Catalog Tables ({len(tables)})\n")
        lines.append("These can be queried by agents for structured data.\n")
        for table in tables[:10]:  # Show first 10
            lines.append(f"- `{table['name']}` ({table['table_type']})")
            if table.get("comment"):
                lines.append(f"  - {table['comment']}")
            if table.get("columns"):
                col_names = [c["name"] for c in table["columns"][:5]]
                lines.append(f"  - Columns: {', '.join(col_names)}")
        if len(tables) > 10:
            lines.append(f"\n*...and {len(tables) - 10} more*\n")
        lines.append("")

    # Vector Search Indexes
    indexes = results.get("vector_search_indexes", [])
    if indexes:
        lines.append(f"## Vector Search Indexes ({len(indexes)})\n")
        lines.append("These can be used for RAG applications.\n")
        for idx in indexes:
            lines.append(f"- `{idx['name']}`")
            lines.append(f"  - Endpoint: {idx['endpoint']}")
            lines.append(f"  - Status: {idx['status']}")
        lines.append("")

    # Genie Spaces
    spaces = results.get("genie_spaces", [])
    if spaces:
        lines.append(f"## Genie Spaces ({len(spaces)})\n")
        lines.append("These provide conversational data access.\n")
        for space in spaces:
            lines.append(f"- `{space['name']}` (ID: {space['id']})")
            if space.get("description"):
                lines.append(f"  - {space['description']}")
        lines.append("")

    # MCP Server Packages
    packages = results.get("mcp_server_packages", [])
    if packages:
        lines.append(f"## MCP Server Packages ({len(packages)})\n")
        lines.append("Installed Python packages that provide MCP tools.\n")
        for pkg in packages:
            lines.append(f"- `{pkg['package']}` (v{pkg['version']})")
        lines.append("")

    # Custom MCP Servers (Databricks Apps)
    custom_servers = results.get("custom_mcp_servers", [])
    if custom_servers:
        lines.append(f"## Custom MCP Servers ({len(custom_servers)})\n")
        lines.append("MCP servers deployed as Databricks Apps (names starting with mcp-).\n")
        for server in custom_servers:
            lines.append(f"- `{server['name']}`")
            if server.get("url"):
                lines.append(f"  - URL: {server['url']}")
            if server.get("status"):
                lines.append(f"  - Status: {server['status']}")
            if server.get("description"):
                lines.append(f"  - {server['description']}")
        lines.append("")

    # External MCP Servers (UC Connections)
    external_servers = results.get("external_mcp_servers", [])
    if external_servers:
        lines.append(f"## External MCP Servers ({len(external_servers)})\n")
        lines.append("External MCP servers configured via Unity Catalog connections.\n")
        for server in external_servers:
            lines.append(f"- `{server['name']}`")
            if server.get("full_name"):
                lines.append(f"  - Full name: {server['full_name']}")
            if server.get("comment"):
                lines.append(f"  - {server['comment']}")
        lines.append("")

    return "\n".join(lines)


def main():
    """Main discovery function."""
    import argparse

    parser = argparse.ArgumentParser(description="Discover available agent tools and data sources")
    parser.add_argument("--catalog", help="Limit discovery to specific catalog")
    parser.add_argument("--schema", help="Limit discovery to specific schema (requires --catalog)")
    parser.add_argument("--format", choices=["json", "markdown"], default="markdown", help="Output format")
    parser.add_argument("--output", help="Output file (default: stdout)")

    args = parser.parse_args()

    if args.schema and not args.catalog:
        print("Error: --schema requires --catalog", file=sys.stderr)
        sys.exit(1)

    print("Discovering available tools and data sources...", file=sys.stderr)

    # Initialize Databricks workspace client
    w = WorkspaceClient()

    results = {}

    # Discover each type
    print("- UC Functions...", file=sys.stderr)
    results["uc_functions"] = discover_uc_functions(w, catalog=args.catalog)

    print("- UC Tables...", file=sys.stderr)
    results["uc_tables"] = discover_uc_tables(w, catalog=args.catalog, schema=args.schema)

    print("- Vector Search Indexes...", file=sys.stderr)
    results["vector_search_indexes"] = discover_vector_search_indexes(w)

    print("- Genie Spaces...", file=sys.stderr)
    results["genie_spaces"] = discover_genie_spaces(w)

    print("- MCP Server Packages...", file=sys.stderr)
    results["mcp_server_packages"] = discover_mcp_servers()

    print("- Custom MCP Servers (Apps)...", file=sys.stderr)
    results["custom_mcp_servers"] = discover_custom_mcp_servers(w)

    print("- External MCP Servers (Connections)...", file=sys.stderr)
    results["external_mcp_servers"] = discover_external_mcp_servers(w)

    # Format output
    if args.format == "json":
        output = json.dumps(results, indent=2)
    else:
        output = format_output_markdown(results)

    # Write output
    if args.output:
        Path(args.output).write_text(output)
        print(f"\nResults written to {args.output}", file=sys.stderr)
    else:
        print("\n" + output)

    # Print summary
    print("\n=== Discovery Summary ===", file=sys.stderr)
    print(f"UC Functions: {len(results['uc_functions'])}", file=sys.stderr)
    print(f"UC Tables: {len(results['uc_tables'])}", file=sys.stderr)
    print(f"Vector Search Indexes: {len(results['vector_search_indexes'])}", file=sys.stderr)
    print(f"Genie Spaces: {len(results['genie_spaces'])}", file=sys.stderr)
    print(f"MCP Server Packages: {len(results['mcp_server_packages'])}", file=sys.stderr)
    print(f"Custom MCP Servers: {len(results['custom_mcp_servers'])}", file=sys.stderr)
    print(f"External MCP Servers: {len(results['external_mcp_servers'])}", file=sys.stderr)


if __name__ == "__main__":
    main()
