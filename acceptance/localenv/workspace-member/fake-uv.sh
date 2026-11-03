#!/bin/bash
# The workspace root is the test folder, where this script is installed as uv.
# `uv sync` creates the shared venv there with a fake interpreter that answers
# the validate probe (prefixes from libs/localenv/uv.go).
root="$(cd "$(dirname "$0")" && pwd)"
case "$1 $2" in
  "--version ") echo "uv 0.12.9 (fake)"; exit 0 ;;
  "workspace dir") echo "$root"; exit 0 ;;
  "python install") exit 0 ;;
  "pip install") exit 0 ;;
  "sync --python")
    mkdir -p "$root/.venv/bin"
    cat > "$root/.venv/bin/python" <<'PY'
#!/bin/bash
echo "PYVER:3.12"
echo "DBCVER:17.2.0"
echo "PYSPARKVER:"
echo "DBCIMPORT:"
PY
    chmod +x "$root/.venv/bin/python"
    exit 0 ;;
esac
echo "fake uv: unexpected call: $*" >&2
exit 1
