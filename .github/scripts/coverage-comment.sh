#!/usr/bin/env bash
# Posts, or updates, the coverage comment of a PR (.github/workflows/coverage-comment.yml): the
# coverage that its build run uploaded, against that of the newest main build its change starts
# from. Exits quietly when the run has no coverage, or its PR has moved on to another commit.
#
# The run's artifacts and their meta come from the PR's author: they are only read as data, by
# `coverpack -compare`, and what they name is checked against git and the GitHub API.
#
# Env: GH_TOKEN, REPO (owner/name), RUN_ID (the PR's build run), HEAD_SHA (the PR commit it
# built), HEAD_OWNER and HEAD_BRANCH (where that commit is). With DRY_RUN=1, prints the comment.
# Run from the root of a full clone of the default branch.
set -euo pipefail

sha_re='^[0-9a-f]{40}$'
[[ $HEAD_SHA =~ $sha_re ]] || { echo "bad HEAD_SHA" >&2; exit 1; }
[[ $RUN_ID =~ ^[0-9]+$ ]] || { echo "bad RUN_ID" >&2; exit 1; }

work=$(mktemp -d)
coverpack="$work/coverpack"
go -C tools build -o "$coverpack" ./coverpack

# The PR whose head is still the commit the run built.
pr=$(gh api -X GET "repos/$REPO/pulls" -f state=open -f head="$HEAD_OWNER:$HEAD_BRANCH" \
  --jq "[.[] | select(.head.sha == \"$HEAD_SHA\") | .number][0] // empty")
if [ -z "$pr" ]; then
  echo "No open PR has $HEAD_SHA as its head."
  exit 0
fi

# cells RUN: the cells of a run's coverage artifacts, sorted; a failure to list them fails the
# script. A run of a PR names its artifacts as it likes: only plain names of a bounded size
# are taken, as coverpack bounds what it reads of each.
cells() {
  gh api "repos/$REPO/actions/runs/$1/artifacts?per_page=100" --jq '.artifacts[]
    | select(.expired | not) | select(.name | test("^coverage-linux-[a-z0-9]{1,32}$"))
    | select(.size_in_bytes < 16777216) | .name | ltrimstr("coverage-linux-")' | sort | sed -n 1,8p
}

# lines VAR TEXT: the non-empty lines of TEXT as the array VAR. (The functions here run in a
# command substitution, not a process one, so that their failure fails the script.)
lines() {
  local -n into=$1
  into=()
  [ -z "$2" ] || mapfile -t into <<<"$2"
}

# download RUN DIR CELL...: the coverage file of each cell, or the script fails.
download() {
  local run=$1 dir=$2
  shift 2
  for cell in "$@"; do
    gh run download "$run" -R "$REPO" -n "coverage-linux-$cell" -D "$dir/$cell" >/dev/null
    [ -f "$dir/$cell/coverage.bin.zst" ] || { echo "Run $run: no coverage.bin.zst in coverage-linux-$cell" >&2; exit 1; }
    echo "$dir/$cell/coverage.bin.zst"
  done
}

out=$(cells "$RUN_ID")
lines head_cells "$out"
if [ ${#head_cells[@]} -eq 0 ]; then
  echo "Run $RUN_ID has no coverage."
  exit 0
fi
out=$(download "$RUN_ID" "$work/head" "${head_cells[@]}")
lines head_files "$out"

# The PR's build tests GitHub's merge of the PR into main, so its lines are those of that
# commit: the PR's change is that commit against its first parent.
merge=$("$coverpack" -commit "${head_files[0]}")
git fetch -q --no-tags origin "$merge"
parents=$(git rev-list --parents -n 1 "$merge")
read -r _ parent1 parent2 rest <<<"$parents"
if [ "$parent2" != "$HEAD_SHA" ] || [ -n "${rest:-}" ] || ! git merge-base --is-ancestor "$parent1" origin/main; then
  echo "Commit $merge is not a merge of $HEAD_SHA into main." >&2
  exit 1
fi

# The base: the newest main build at or before the PR's starting point that has the coverage of
# the same cells (a build whose cell failed early lacks it).
main_runs=$(gh run list -R "$REPO" --workflow push.yml --branch main --event push --limit 200 \
  --json databaseId,headSha,status --jq '.[] | select(.status == "completed") | "\(.databaseId) \(.headSha)"')
base_args=()
base_desc=""
tries=0
while read -r run sha; do
  [[ $sha =~ $sha_re ]] && git merge-base --is-ancestor "$sha" "$parent1" 2>/dev/null || continue
  out=$(cells "$run")
  lines base_cells "$out"
  if [ "${base_cells[*]}" = "${head_cells[*]}" ]; then
    out=$(download "$run" "$work/base-$run" "${base_cells[@]}")
    lines base_files "$out"
    for f in "${base_files[@]}"; do base_args+=(-base "$f"); done
    base_desc="main at [\`${sha:0:9}\`](https://github.com/$REPO/commit/$sha) ([run](https://github.com/$REPO/actions/runs/$run))"
    behind=$(git rev-list --count --first-parent "$sha..$parent1")
    if [ "$behind" -gt 0 ]; then
      base_desc+=", $behind commits before the PR's base: the totals include their changes"
    fi
    break
  fi
  tries=$((tries + 1))
  [ "$tries" -lt 10 ] || break
done <<<"$main_runs"

head_args=()
for f in "${head_files[@]}"; do head_args+=(-head "$f"); done
"$coverpack" -compare -repo . -from "$parent1" -to "$merge" "${head_args[@]}" "${base_args[@]}" \
  -head-desc "[\`${HEAD_SHA:0:9}\`](https://github.com/$REPO/commit/$HEAD_SHA) ([run](https://github.com/$REPO/actions/runs/$RUN_ID))" \
  -base-desc "$base_desc" -o "$work/comment.md"

if [ "${DRY_RUN:-}" = 1 ]; then
  echo "PR #$pr:"
  cat "$work/comment.md"
  exit 0
fi

marker=$(head -n 1 "$work/comment.md")
existing=$(gh api --paginate "repos/$REPO/issues/$pr/comments" \
  --jq ".[] | select(.user.login == \"github-actions[bot]\" and (.body | startswith(\"$marker\"))) | .id" | sed -n 1p)
if [ -n "$existing" ]; then
  gh api -X PATCH "repos/$REPO/issues/comments/$existing" -F body=@"$work/comment.md" >/dev/null
  echo "Updated the comment on PR #$pr."
else
  gh api "repos/$REPO/issues/$pr/comments" -F body=@"$work/comment.md" >/dev/null
  echo "Commented on PR #$pr."
fi
