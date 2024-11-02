# Bugbash

The script in this directory can be used to conveniently exec into a shell
where a CLI build for a specific branch is made available.

## Usage

This script prompts if you do NOT have at least Bash 5 installed,
but works without command completion with earlier versions.

```shell
bash <(curl -fsSL https://raw.githubusercontent.com/databricks/cli/main/internal/bugbash/exec.sh) my-branch
```
