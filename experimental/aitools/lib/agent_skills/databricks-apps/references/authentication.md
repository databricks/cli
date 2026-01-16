# Authentication

## Check Status

```bash
databricks auth profiles
```

## Configure Profile

```bash
databricks configure --profile <name>
```

## OAuth Login

```bash
databricks auth login --profile <name> --host <workspace-url>
```

Browser-based OAuth. Recommended for development.

## Profile Switching

```bash
# single command
DATABRICKS_CONFIG_PROFILE=<name> databricks <command>

# or flag
databricks --profile <name> <command>
```

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `DATABRICKS_HOST` | Workspace URL |
| `DATABRICKS_CONFIG_PROFILE` | Profile name |
| `DATABRICKS_WAREHOUSE_ID` | Default warehouse |

## Troubleshooting

| Issue | Solution |
|-------|----------|
| No profiles | `databricks configure --profile <name>` |
| Token expired | `databricks auth login --profile <name> --host <url>` |
| Wrong workspace | Check `DATABRICKS_CONFIG_PROFILE` or use `--profile` |
| Silent auth fail | `databricks auth profiles` to check status |

## New Account

Free account: https://docs.databricks.com/getting-started/free-edition
