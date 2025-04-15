#!/usr/bin/env python3
"""
Start integration test jobs for PRs by non-team members that are approved by team members.
"""

import argparse
import json
import subprocess
import sys
from pathlib import Path
import re


CLI_REPO = "databricks/cli"
CODE_OWNERS = "pietern andrewnester shreyas-goenka denik anton-107".split()
ALLOWED_HEAD_REPOSITORY = {"id": "R_kgDOHVGMwQ", "name": "cli"}
ALLOWED_HEAD_OWNER = {"id": "MDEyOk9yZ2FuaXphdGlvbjQ5OTgwNTI=", "login": "databricks"}


def run(cmd):
    sys.stderr.write("+ " + " ".join(cmd) + "\n")
    return subprocess.run(cmd, check=True)


def run_json(cmd):
    sys.stderr.write("+ " + " ".join(cmd) + "\n")
    result = subprocess.run(cmd, stdout=subprocess.PIPE, encoding="utf-8", check=True)

    try:
        return json.loads(result.stdout)
    except Exception:
        sys.stderr.write(f"Failed to JSON parse:\n{result.stdout}\n")
        raise


def get_approved_prs_by_non_team():
    prs = run_json(
        ["gh", "pr", "-R", CLI_REPO, "list", "--limit", "300", "--json", "number,author,reviews,headRefOid,headRepository,headRepositoryOwner"]
    )
    result = []

    for pr in prs:
        pr_number = pr["number"]
        author = pr["author"]["login"]

        if author in CODE_OWNERS:
            continue

        head_repo = pr["headRepository"]
        if head_repo != ALLOWED_HEAD_REPOSITORY:
            print(f"#{pr_number} by {author} skipped due to headRepository: {head_repo}")
            continue

        head_owner = pr["headRepositoryOwner"]
        if head_owner != ALLOWED_HEAD_OWNER:
            print(f"#{pr_number} by {author} skipped due to headRepositoryOwner: {head_owner}")
            continue

        approved_by = []
        for review in pr.get("reviews", []):
            approver = review["author"]["login"]
            if review["state"] == "APPROVED" and approver in CODE_OWNERS:
                approved_by.append(approver)

        if not approved_by:
            continue

        result.append(
            {
                "number": pr_number,
                "commit": pr["headRefOid"],
                "author": author,
                "approved_by": approved_by,
            }
        )

    return prs, result


def start_job(pr_number, commit_sha, author, approved_by, workflow, repo, force=False):
    pr_details = run_json(["gh", "pr", "-R", CLI_REPO, "view", str(pr_number), "--json", "title,url"])
    pr_title = pr_details.get("title", "")
    commit_url = f"https://github.com/{CLI_REPO}/pull/{pr_number}/commits/{commit_sha}"
    approvers = ", ".join(approved_by)

    print(f"PR:        #{pr_number} {pr_title}")
    print(f"Author:    {author}")
    print(f"Approvers: {approvers}")
    print(f"Commit:    {commit_url}")

    if force:
        response = "y"
        print("Starting integration tests.")
    else:
        response = input("Start integration tests? (y/n): ")

    if response.lower() == "y":
        result = run(
            [
                "gh",
                "workflow",
                "run",
                workflow,
                "-R",
                repo,
                "-F",
                f"pull_request_number={pr_number}",
                "-F",
                f"commit_sha={commit_sha}",
            ],
        )
        print(f"Started integration tests for PR #{pr_number}")


def get_status(commit_sha):
    statuses = run_json(["gh", "api", f"repos/{CLI_REPO}/commits/{commit_sha}/statuses"])
    result = []
    for st in statuses:
        if st["context"] != "Integration Tests Check":
            continue
        result.append(f"{st['state']} {st['target_url']}")
    return result


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--yes", action="store_true", default=False)
    parser.add_argument("--workflow", default="cli-isolated-pr")
    parser.add_argument("-R", "--repo")

    args = parser.parse_args()
    assert args.repo, "Must provide repo where workflow is run with -R"

    all_prs, approved_prs = get_approved_prs_by_non_team()

    if not approved_prs:
        print(f"Fetched {len(all_prs)} PRs. No approved PRs from non-team members found.")
        return

    print(f"Fetched {len(all_prs)} PRs.")

    for pr in approved_prs:
        pr_number = pr["number"]
        commit_sha = pr["commit"]

        status = get_status(commit_sha)

        if not status:
            start_job(pr_number, commit_sha, pr["author"], approved_by=pr["approved_by"], workflow=args.workflow, repo=args.repo, force=args.yes)
        else:
            commit_url = f"https://github.com/{CLI_REPO}/pull/{pr_number}/commits/{commit_sha}"
            print(f"Tests already running for PR #{pr_number} {commit_url}")
            print("\n".join(status))
        print()


if __name__ == "__main__":
    main()
