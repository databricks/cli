#!/bin/bash
# PreToolUse gate: block `gh pr create` and `git pp` until the
# pr-checklist skill has been invoked in the current session.
# Fail closed: any internal error denies the tool call.

set -uo pipefail

deny() {
  jq -n --arg reason "$1" '{
    hookSpecificOutput: {
      hookEventName: "PreToolUse",
      permissionDecision: "deny",
      permissionDecisionReason: $reason
    }
  }'
  exit 0
}

INPUT=$(cat) || deny "pr-checklist gate: failed to read hook input"

COMMAND=$(jq -r '.tool_input.command // empty' <<<"$INPUT") \
  || deny "pr-checklist gate: failed to parse tool_input.command"
TRANSCRIPT=$(jq -r '.transcript_path // empty' <<<"$INPUT") \
  || deny "pr-checklist gate: failed to parse transcript_path"

# Only gate PR-creation commands. Match `gh pr create` or `git pp`
# (the project's PR-push alias) as standalone tokens.
if ! grep -qE '(^|[[:space:]])(gh[[:space:]]+pr[[:space:]]+create|git[[:space:]]+pp)([[:space:]]|$)' <<<"$COMMAND"; then
  exit 0
fi

if [[ -z "$TRANSCRIPT" || ! -f "$TRANSCRIPT" ]]; then
  deny "Run /pr-checklist before creating a PR (no session transcript available to verify)."
fi

# Skill tool_use blocks appear one-per-line in the JSONL transcript.
# Require both substrings on the same line to tolerate field-order variance.
if grep -E '"name"[[:space:]]*:[[:space:]]*"Skill"' "$TRANSCRIPT" \
   | grep -qE '"skill"[[:space:]]*:[[:space:]]*"pr-checklist"'; then
  exit 0
fi

deny "Run /pr-checklist before \`gh pr create\` or \`git pp\`. The pre-PR gate is mandatory."
