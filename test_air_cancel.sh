#!/usr/bin/env bash
#
# Manual / e2e smoke test for `air cancel` against a REAL workspace.
# Run from ~/cli-wt-air-cancel (branch air-cancel-port = m2-3 + cancel).
#
#   PROFILE=riddhi.bhagwat ./test_air_cancel.sh
#
# Optional, opt-in (these ACTUALLY CANCEL runs):
#   RUN_ID=<jobRunId> [RUN_ID2=<jobRunId>] PROFILE=... ./test_air_cancel.sh
#   ALLOW_CANCEL_ALL=1 PROFILE=... ./test_air_cancel.sh
#
# What runs unconditionally is non-destructive:
#   - argument validation (no API call)
#   - invalid id / not-found (the cancel just fails)
#   - `--all` preview then ABORT (answers "n", cancels nothing)
#
# set -e is intentionally OFF: most cases are expected to exit non-zero and we
# assert on that.
set -u

PROFILE="${PROFILE:-riddhi.bhagwat}"
CLI="${CLI:-./cli}"
BOGUS_ID="999999999999999"   # a syntactically valid id that should not exist

pass=0 fail=0
hr(){ printf -- '------------------------------------------------------------\n'; }
section(){ printf '\n=========================  %s  =========================\n' "$1"; }

# check "<desc>" <expected_exit|any> "<expected_substr|>" -- cmd...
check(){
  local desc="$1" expcode="$2" substr="$3"; shift 3; [ "$1" = "--" ] && shift
  hr; printf '· %s\n$ %s\n' "$desc" "$*"
  local out code ok=1
  out="$("$@" 2>&1)"; code=$?
  printf '%s\n' "$out"
  if [ "$expcode" != "any" ] && [ "$code" -ne "$expcode" ]; then
    printf '  \xE2\x9C\x97 exit %s (expected %s)\n' "$code" "$expcode"; ok=0
  fi
  if [ -n "$substr" ] && ! grep -qF -- "$substr" <<<"$out"; then
    printf '  \xE2\x9C\x97 missing text: %s\n' "$substr"; ok=0
  fi
  if [ "$ok" = 1 ]; then printf '  \xE2\x9C\x93 PASS (exit %s)\n' "$code"; pass=$((pass+1));
  else fail=$((fail+1)); fi
}

# ----------------------------------------------------------------------------
section "0. preflight"
if [ ! -x "$CLI" ]; then
  echo "Building CLI binary..."; go build -o "$CLI" . || { echo "build failed"; exit 1; }
fi
echo "CLI:     $CLI"
echo "PROFILE: $PROFILE"
echo "Active runs in this workspace (for picking RUN_IDs):"
preflight="$("$CLI" experimental air list --active -p "$PROFILE" 2>&1)"; printf '%s\n' "$preflight"
if grep -qiE "refresh token is invalid|auth login|could not be retrieved|default auth" <<<"$preflight"; then
  hr
  echo "*** Authentication failed for profile '$PROFILE'. Re-authenticate, then re-run this script: ***"
  echo "    databricks auth login --profile $PROFILE"
  exit 1
fi

# ----------------------------------------------------------------------------
section "1. argument validation (no API call)"
check "no args -> error"                 1 "provide at least one JOB_RUN_ID, or use --all" -- "$CLI" experimental air cancel -p "$PROFILE"
check "ids + --all -> error"             1 "cannot combine JOB_RUN_ID arguments with --all" -- "$CLI" experimental air cancel 123 --all -p "$PROFILE"
check "--help -> usage, no error"        0 "Cancel one or more runs" -- "$CLI" experimental air cancel --help

# ----------------------------------------------------------------------------
section "2. invalid / not-found (cancels nothing)"
check "non-numeric id"                   1 "invalid run ID" -- "$CLI" experimental air cancel abc -p "$PROFILE"
check "zero id"                          1 "invalid run ID" -- "$CLI" experimental air cancel 0 -p "$PROFILE"
check "not-found (text)"                 1 "not found" -- "$CLI" experimental air cancel "$BOGUS_ID" -p "$PROFILE"
check "not-found summary"                1 "1 run(s) failed to cancel." -- "$CLI" experimental air cancel "$BOGUS_ID" -p "$PROFILE"
check "not-found (json) has failed"      1 '"failed"' -- "$CLI" experimental air cancel "$BOGUS_ID" -o json -p "$PROFILE"

# ----------------------------------------------------------------------------
section "3. --all preview + ABORT (answers 'n', cancels nothing)"
hr; printf '· --all then answer n (preview shown, then aborted)\n$ printf "n\\n" | %s experimental air cancel --all -p %s\n' "$CLI" "$PROFILE"
out="$(printf 'n\n' | "$CLI" experimental air cancel --all -p "$PROFILE" 2>&1)"; code=$?
printf '%s\n' "$out"
if grep -qF "Cancellation aborted." <<<"$out" || grep -qF "No active runs found." <<<"$out"; then
  printf '  \xE2\x9C\x93 PASS (exit %s)\n' "$code"; pass=$((pass+1))
else
  printf '  \xE2\x9C\x97 expected "Cancellation aborted." or "No active runs found."\n'; fail=$((fail+1))
fi

# ----------------------------------------------------------------------------
section "4. by-id cancellation (opt-in: set RUN_ID / RUN_ID2)  *** DESTRUCTIVE ***"
if [ -n "${RUN_ID:-}" ]; then
  check "cancel single (text)"           0 "Successfully requested cancellation for run ${RUN_ID}" -- "$CLI" experimental air cancel "$RUN_ID" -p "$PROFILE"
  check "cancel single (json)"           0 '"cancelled"' -- "$CLI" experimental air cancel "$RUN_ID" -o json -p "$PROFILE"
  check "valid + bogus -> partial fail"  1 "1 run(s) failed to cancel." -- "$CLI" experimental air cancel "$RUN_ID" "$BOGUS_ID" -p "$PROFILE"
  if [ -n "${RUN_ID2:-}" ]; then
    check "cancel two -> count summary"  0 "Successfully requested cancellation for 2 run(s)." -- "$CLI" experimental air cancel "$RUN_ID" "$RUN_ID2" -p "$PROFILE"
  fi
else
  echo "skipped (set RUN_ID=<id> [RUN_ID2=<id>] to enable)"
fi

# ----------------------------------------------------------------------------
section "5. --all -y cancellation (opt-in: ALLOW_CANCEL_ALL=1)  *** DESTRUCTIVE ***"
if [ "${ALLOW_CANCEL_ALL:-0}" = "1" ]; then
  check "--all -y (text)"                any "" -- "$CLI" experimental air cancel --all -y -p "$PROFILE"
  check "--all -y (json) all:true"       any '"all"' -- "$CLI" experimental air cancel --all -y -o json -p "$PROFILE"
else
  echo "skipped (set ALLOW_CANCEL_ALL=1 to enable — this cancels every active run you own)"
fi

# ----------------------------------------------------------------------------
hr; printf 'SUMMARY: %s passed, %s failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
