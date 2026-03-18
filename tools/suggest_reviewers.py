#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# ///

import fnmatch
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
    return 1.0 if path.endswith(".go") else 0.5


def get_changed_files(pr_number: str) -> list[str]:
    r = subprocess.run(
        ["gh", "pr", "diff", "--name-only", pr_number],
        capture_output=True,
        encoding="utf-8",
    )
    if r.returncode != 0:
        print(f"gh pr diff failed: {r.stderr.strip()}", file=sys.stderr)
        sys.exit(1)
    return [f.strip() for f in r.stdout.splitlines() if f.strip()]


def git_log(path: str) -> list[tuple[str, str, datetime]]:
    r = subprocess.run(
        ["git", "log", "-50", "--no-merges", "--since=12 months ago", "--format=%H|%an|%aI", "--", path],
        capture_output=True,
        encoding="utf-8",
    )
    if r.returncode != 0:
        return []
    entries = []
    for line in r.stdout.splitlines():
        parts = line.strip().split("|", 2)
        if len(parts) != 3:
            continue
        try:
            entries.append((parts[0], parts[1], datetime.fromisoformat(parts[2])))
        except ValueError:
            continue
    return entries


def resolve_login(repo: str, sha: str, author_name: str) -> str | None:
    if author_name in _login_cache:
        return _login_cache[author_name]
    r = subprocess.run(
        ["gh", "api", f"repos/{repo}/commits/{sha}", "--jq", ".author.login"],
        capture_output=True,
        encoding="utf-8",
    )
    login = r.stdout.strip() or None if r.returncode == 0 else None
    _login_cache[author_name] = login
    return login


def _codeowners_match(pattern: str, filepath: str) -> bool:
    if pattern.startswith("/"):
        pattern = pattern[1:]
        if pattern.endswith("/"):
            return filepath.startswith(pattern)
        return fnmatch.fnmatch(filepath, pattern) or filepath == pattern
    return fnmatch.fnmatch(filepath, pattern) or fnmatch.fnmatch(Path(filepath).name, pattern)


def parse_codeowners(changed_files: list[str]) -> list[str]:
    path = Path(".github/CODEOWNERS")
    if not path.exists():
        return []
    rules: list[tuple[str, list[str]]] = []
    for line in path.read_text().splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        parts = line.split()
        owners = [p for p in parts[1:] if p.startswith("@")]
        if len(parts) >= 2 and owners:
            rules.append((parts[0], owners))

    all_owners: set[str] = set()
    for filepath in changed_files:
        matched = []
        for pattern, owners in rules:
            if _codeowners_match(pattern, filepath):
                matched = owners
        all_owners.update(matched)
    return sorted(all_owners)


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
            s = weight * (0.5 ** (days_ago / 150))
            scores[login] = scores.get(login, 0) + s
            dir_scores.setdefault(login, {})
            dir_scores[login][top_dir] = dir_scores[login].get(top_dir, 0) + s
            file_contributed = True
        if file_contributed:
            scored_count += 1
    return scores, dir_scores, scored_count


def top_dirs(ds: dict[str, float], n: int = 3) -> list[str]:
    return [d for d, _ in sorted(ds.items(), key=lambda x: -x[1])[:n]]


def fmt_reviewer(login: str, dirs: list[str]) -> str:
    mention = f"@{login}" if MENTION_REVIEWERS else login
    return f"- {mention} -- recent work in {', '.join(f'`{d}/`' for d in dirs)}"


def select_reviewers(ss: list[tuple[str, float]]) -> list[tuple[str, float]]:
    if not ss:
        return []
    out = [ss[0]]
    if len(ss) >= 2 and ss[0][1] < 1.5 * ss[1][1]:
        out.append(ss[1])
        if len(ss) >= 3 and ss[1][1] < 1.5 * ss[2][1]:
            out.append(ss[2])
    return out


def compute_confidence(ss: list[tuple[str, float]], scored_count: int) -> str:
    if scored_count < 3 or len(ss) < 2:
        return "low"
    if len(ss) >= 3 and ss[0][1] > 2 * ss[2][1]:
        return "high"
    if len(ss) >= 3 and ss[0][1] > 1.5 * ss[2][1]:
        return "medium"
    return "low"


def fmt_eligible(owners: list[str]) -> str:
    if MENTION_REVIEWERS:
        return ", ".join(owners)
    return ", ".join(o.lstrip("@") for o in owners)


def build_comment(
    sorted_scores: list[tuple[str, float]],
    dir_scores: dict[str, dict[str, float]],
    total_files: int,
    scored_count: int,
    eligible_owners: list[str],
    pr_author: str,
) -> str:
    reviewers = select_reviewers(sorted_scores)
    suggested_logins = {login.lower() for login, _ in reviewers}
    eligible = [
        o
        for o in eligible_owners
        if o.lstrip("@").lower() != pr_author.lower() and o.lstrip("@").lower() not in suggested_logins
    ]

    lines = [MARKER]
    if reviewers:
        lines += [
            "## Suggested reviewers",
            "",
            "Based on git history of the changed files, these people are best suited to review:",
            "",
        ]
        for login, _ in reviewers:
            lines.append(fmt_reviewer(login, top_dirs(dir_scores.get(login, {}))))
        lines += ["", f"Confidence: {compute_confidence(sorted_scores, scored_count)}"]
        if eligible:
            lines += [
                "",
                "## Eligible reviewers",
                "",
                "Based on CODEOWNERS, these people or teams could also review:",
                "",
                fmt_eligible(eligible),
            ]
    elif eligible:
        lines += [
            "## Eligible reviewers",
            "",
            "Could not determine reviewers from git history. Based on CODEOWNERS, these people or teams could review:",
            "",
            fmt_eligible(eligible),
        ]
    else:
        lines += [
            "## Suggested reviewers",
            "",
            f"Could not determine reviewers from git history. Please pick from {CODEOWNERS_LINK}.",
        ]

    lines += [
        "",
        f"<sub>Suggestions based on git history of {total_files} changed files "
        f"({scored_count} scored). "
        f"See {CODEOWNERS_LINK} for path-specific ownership rules.</sub>",
    ]
    return "\n".join(lines) + "\n"


def find_existing_comment(repo: str, pr_number: str) -> str | None:
    r = subprocess.run(
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
    if r.returncode != 0:
        print(f"gh api comments failed: {r.stderr.strip()}", file=sys.stderr)
        sys.exit(1)
    for cid in r.stdout.splitlines():
        cid = cid.strip()
        if cid:
            return cid
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
    eligible = parse_codeowners(files)
    comment = build_comment(sorted_scores, dir_scores, len(files), scored_count, eligible, pr_author)

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
