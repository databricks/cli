#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# ///

import os
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path

MENTION_REVIEWERS = True

CODEOWNERS_LINK = "[CODEOWNERS](.github/CODEOWNERS)"

MARKER = "<!-- REVIEWER_SUGGESTION -->"

_login_cache: dict[str, str | None] = {}


def classify_file(path: str) -> float:
    p = Path(path)
    if p.name.startswith("out.") or p.name == "output.txt":
        return 0.0
    if path.startswith(("cmd/workspace/", "cmd/account/")):
        return 0.0
    if path.startswith(("acceptance/", "integration/")):
        return 0.2
    if path.endswith("_test.go"):
        return 0.3
    if path.endswith(".go"):
        return 1.0
    return 0.5


def get_changed_files(pr_number: str) -> list[str]:
    result = subprocess.run(
        ["gh", "pr", "diff", "--name-only", pr_number],
        capture_output=True,
        encoding="utf-8",
    )
    if result.returncode != 0:
        print(f"gh pr diff failed: {result.stderr.strip()}", file=sys.stderr)
        sys.exit(1)
    return [f.strip() for f in result.stdout.splitlines() if f.strip()]


def git_log(path: str) -> list[tuple[str, str, datetime]]:
    result = subprocess.run(
        [
            "git",
            "log",
            "-50",
            "--no-merges",
            "--since=12 months ago",
            "--format=%H|%an|%aI",
            "--",
            path,
        ],
        capture_output=True,
        encoding="utf-8",
    )
    if result.returncode != 0:
        return []
    entries = []
    for line in result.stdout.splitlines():
        line = line.strip()
        if not line or "|" not in line:
            continue
        parts = line.split("|", 2)
        if len(parts) != 3:
            continue
        sha, name, date_str = parts
        try:
            entries.append((sha, name, datetime.fromisoformat(date_str)))
        except ValueError:
            continue
    return entries


def resolve_login(repo: str, sha: str, author_name: str) -> str | None:
    if author_name in _login_cache:
        return _login_cache[author_name]
    result = subprocess.run(
        ["gh", "api", f"repos/{repo}/commits/{sha}", "--jq", ".author.login"],
        capture_output=True,
        encoding="utf-8",
    )
    login = result.stdout.strip() if result.returncode == 0 else None
    if not login:
        login = None
    _login_cache[author_name] = login
    return login


def score_contributors(
    files: list[str], pr_author: str, now: datetime, repo: str
) -> tuple[dict[str, float], dict[str, dict[str, float]], int]:
    scores: dict[str, float] = {}
    dir_scores: dict[str, dict[str, float]] = {}
    scored_count = 0
    author_login = pr_author.lower()

    for filepath in files:
        weight = classify_file(filepath)
        if weight == 0.0:
            continue

        history = git_log(filepath)
        if not history:
            parent = str(Path(filepath).parent)
            if parent and parent != ".":
                history = git_log(parent)
        if not history:
            continue

        top_dir = str(Path(filepath).parent) or "."
        file_contributed = False

        for sha, name, commit_date in history:
            if name.endswith("[bot]"):
                continue
            login = resolve_login(repo, sha, name)
            if not login or login.lower() == author_login:
                continue

            days_ago = max(0, (now - commit_date).total_seconds() / 86400)
            recency = 0.5 ** (days_ago / 150)
            s = weight * recency

            scores[login] = scores.get(login, 0) + s
            dir_scores.setdefault(login, {})
            dir_scores[login][top_dir] = dir_scores[login].get(top_dir, 0) + s
            file_contributed = True

        if file_contributed:
            scored_count += 1

    return scores, dir_scores, scored_count


def top_dirs(ds: dict[str, float], n: int = 3) -> list[str]:
    return [d for d, _ in sorted(ds.items(), key=lambda x: -x[1])[:n]]


def format_reviewer(login: str, dirs: list[str]) -> str:
    mention = f"@{login}" if MENTION_REVIEWERS else login
    dir_str = ", ".join(f"`{d}/`" for d in dirs)
    return f"- {mention} -- recent work in {dir_str}"


def compute_confidence(sorted_scores: list[tuple[str, float]], scored_count: int) -> str:
    if scored_count < 3 or len(sorted_scores) < 2:
        return "low"
    if len(sorted_scores) >= 3 and sorted_scores[0][1] > 2 * sorted_scores[2][1]:
        return "high"
    if len(sorted_scores) >= 3 and sorted_scores[0][1] > 1.5 * sorted_scores[2][1]:
        return "medium"
    return "low"


def build_comment(
    sorted_scores: list[tuple[str, float]],
    dir_scores: dict[str, dict[str, float]],
    total_files: int,
    scored_count: int,
) -> str:
    if not sorted_scores:
        return (
            f"{MARKER}\n"
            "## Suggested reviewers\n\n"
            "Could not determine reviewers from git history. "
            f"Please pick from {CODEOWNERS_LINK}.\n"
        )

    reviewers = [sorted_scores[0]]
    if len(sorted_scores) >= 2 and sorted_scores[0][1] < 1.35 * sorted_scores[1][1]:
        reviewers.append(sorted_scores[1])

    confidence = compute_confidence(sorted_scores, scored_count)

    lines = [MARKER, "## Suggested reviewers", ""]
    for login, _ in reviewers:
        dirs = top_dirs(dir_scores.get(login, {}))
        lines.append(format_reviewer(login, dirs))
    lines.append("")
    lines.append(f"Confidence: {confidence}")
    lines.append("")
    lines.append(
        f"<sub>Based on git history of {total_files} changed files "
        f"({scored_count} scored). "
        f"See {CODEOWNERS_LINK} for path-specific owners.</sub>"
    )
    return "\n".join(lines) + "\n"


def find_existing_comment(repo: str, pr_number: str) -> str | None:
    result = subprocess.run(
        [
            "gh",
            "api",
            f"repos/{repo}/issues/{pr_number}/comments",
            "--paginate",
            "--jq",
            f'.[] | select(.body | contains("{MARKER}")) | .id',
        ],
        capture_output=True,
        encoding="utf-8",
    )
    if result.returncode != 0:
        print(f"gh api comments failed: {result.stderr.strip()}", file=sys.stderr)
        sys.exit(1)
    for comment_id in result.stdout.splitlines():
        comment_id = comment_id.strip()
        if comment_id:
            return comment_id
    return None


def main():
    repo = os.environ["GITHUB_REPOSITORY"]
    pr_number = os.environ["PR_NUMBER"]
    pr_author = os.environ["PR_AUTHOR"]

    files = get_changed_files(pr_number)
    if not files:
        print("No changed files found.")
        return

    now = datetime.now(timezone.utc)
    scores, dir_scores, scored_count = score_contributors(files, pr_author, now, repo)
    sorted_scores = sorted(scores.items(), key=lambda x: -x[1])
    comment = build_comment(sorted_scores, dir_scores, len(files), scored_count)

    print(comment)
    existing_id = find_existing_comment(repo, pr_number)
    if existing_id:
        subprocess.run(
            ["gh", "api", f"repos/{repo}/issues/comments/{existing_id}", "-X", "PATCH", "-f", f"body={comment}"],
            check=True,
        )
    else:
        subprocess.run(
            ["gh", "pr", "comment", pr_number, "--body", comment],
            check=True,
        )


if __name__ == "__main__":
    main()
