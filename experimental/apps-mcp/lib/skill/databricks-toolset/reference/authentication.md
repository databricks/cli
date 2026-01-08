# Authentication

## Check Current Auth

```bash
scripts/db auth profiles
```

Shows configured profiles and their status.

## Configure New Profile

```bash
scripts/db configure --profile <name>
```

Interactive setup for new profile.

## OAuth Login (U2M)

```bash
scripts/db auth login --profile <name> --host <workspace-url>
```

Browser-based OAuth flow. Recommended for development.

## Profile Switching

Temporary switch for single command:
```bash
DATABRICKS_CONFIG_PROFILE=<name> scripts/db <command>
```

Or use `--profile` flag:
```bash
scripts/db --profile <name> <command>
```

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `DATABRICKS_HOST` | Workspace URL |
| `DATABRICKS_CONFIG_PROFILE` | Profile name from ~/.databrickscfg |
| `DATABRICKS_WAREHOUSE_ID` | Default warehouse for SQL queries |

## Troubleshooting

| Issue | Solution |
|-------|----------|
| No profiles configured | Run `scripts/db configure --profile <name>` |
| Token expired | Run `scripts/db auth login --profile <name> --host <url>` |
| Wrong workspace | Check `DATABRICKS_CONFIG_PROFILE` or use `--profile` flag |
| Auth fails silently | Run `scripts/db auth profiles` to check status |

## New Account Setup

Don't have a Databricks account? Set up a free account at:
https://docs.databricks.com/getting-started/free-edition
