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
  --jq ".[] | select(.head.sha == \"$HEAD_SHA\") | .number" | head -n 1)
if [ -z "$pr" ]; then
  echo "No open PR has $HEAD_SHA as its head."
  exit 0
fi

# download RUN DIR: the coverage files of a run, one per cell, as -head or -base flags.
download() {
  gh run download "$1" -R "$REPO" -p 'coverage-linux-*' -D "$2" >/dev/null 2>&1 || true
  find "$2" -name coverage.bin.zst 2>/dev/null | sort
}

mapfile -t head_files < <(download "$RUN_ID" "$work/head")
if [ ${#head_files[@]} -eq 0 ]; then
  echo "Run $RUN_ID has no coverage."
  exit 0
fi

# The PR's build tests GitHub's merge of the PR into main, so its lines are those of that
# commit: the PR's change is that commit against its first parent.
merge=$("$coverpack" -commit "${head_files[0]}")
git fetch -q --no-tags origin "$merge"
read -r _ parent1 parent2 rest < <(git rev-list --parents -n 1 "$merge")
if [ "$parent2" != "$HEAD_SHA" ] || [ -n "${rest:-}" ] || ! git merge-base --is-ancestor "$parent1" origin/main; then
  echo "Commit $merge is not a merge of $HEAD_SHA into main." >&2
  exit 1
fi

# The base: the newest main build at or before the PR's starting point that has coverage left.
base_args=()
base_desc=""
tries=0
while read -r run sha; do
  [[ $sha =~ $sha_re ]] && git merge-base --is-ancestor "$sha" "$parent1" 2>/dev/null || continue
  mapfile -t base_files < <(download "$run" "$work/base-$run")
  if [ ${#base_files[@]} -gt 0 ]; then
    for f in "${base_files[@]}"; do base_args+=(-base "$f"); done
    base_desc="main at [\`${sha:0:9}\`](https://github.com/$REPO/commit/$sha) ([run](https://github.com/$REPO/actions/runs/$run))"
    break
  fi
  tries=$((tries + 1))
  [ "$tries" -lt 10 ] || break
done < <(gh run list -R "$REPO" --workflow push.yml --branch main --event push --limit 200 \
  --json databaseId,headSha,status --jq '.[] | select(.status == "completed") | "\(.databaseId) \(.headSha)"')

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
  --jq ".[] | select(.user.login == \"github-actions[bot]\" and (.body | startswith(\"$marker\"))) | .id" | head -n 1)
if [ -n "$existing" ]; then
  gh api -X PATCH "repos/$REPO/issues/comments/$existing" -F body=@"$work/comment.md" >/dev/null
  echo "Updated the comment on PR #$pr."
else
  gh api "repos/$REPO/issues/$pr/comments" -F body=@"$work/comment.md" >/dev/null
  echo "Commented on PR #$pr."
fi
