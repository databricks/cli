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


def get_team_members():
    codeowners_path = Path(".github/CODEOWNERS")
    with open(codeowners_path, "r") as f:
        first_line = f.readline().strip()

    team_members = re.findall(r"@([a-zA-Z0-9-.]+)", first_line)
    print("Team members:", team_members)
    return team_members


def get_approved_prs_by_non_team():
    team_members = get_team_members()

    prs = run_json(["gh", "pr", "list", "--json", "number,author,reviews,headRefOid"])
    result = []

    for pr in prs:
        pr_number = pr["number"]
        author = pr["author"]["login"]

        if author in team_members:
            # print(f"Skipping #{pr_number} by {author}")
            continue

        is_approved = False
        for review in pr.get("reviews", []):
            if review["state"] == "APPROVED" and review["author"]["login"] in team_members:
                is_approved = True
                break

        if not is_approved:
            # print(f"Skipping #{pr_number} by {author} -- not approved yet.")
            continue

        print(f"Needs tests: #{pr_number} by {author}")

        result.append(
            {
                "number": pr_number,
                "commit": pr["headRefOid"],
                "author": author,
            }
        )

    return result


def start_job(pr_number, commit_sha, author, force=False):
    pr_details = run_json(["gh", "pr", "view", str(pr_number), "--json", "title,url"])
    pr_title = pr_details.get("title", "")
    pr_url = pr_details.get("url", "")

    print(f"PR: {pr_url}")
    print(f'PR: #{pr_number}: "{pr_title}" by {author} (commit {commit_sha[:7]})')

    # Get approver information
    pr_view = run_json(["gh", "pr", "view", str(pr_number), "--json", "reviews"])
    reviews = pr_view.get("reviews", [])
    approvers = [review["author"]["login"] for review in reviews if review["state"] == "APPROVED"]

    commit_url = f"https://github.com/databricks/cli/pull/{pr_number}/commits/{commit_sha}"
    print(f"This PR is approved by {', '.join(approvers)} but has no integration tests for {commit_url}")

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
                "cli-isolated-pr",
                "-R",
                "databricks-eng/eng-dev-ecosystem",
                "-F",
                f"pull_request_number={pr_number}",
                "-F",
                f"commit_sha={commit_sha}",
            ],
            check=True,
        )
        print(f"Started integration tests for PR #{pr_number}")


def get_status(commit_sha):
    statuses = run_json(["gh", "api", f"repos/databricks/cli/commits/{commit_sha}/statuses"])
    result = []
    for st in statuses:
        if st["context"] != "Integration Tests Check":
            continue
        result.append(f"{st['state']} {st['target_url']}")
    return result


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--yes", action="store_true", default=False)
    args = parser.parse_args()

    approved_prs = get_approved_prs_by_non_team()

    if not approved_prs:
        print("No approved PRs from non-team members found.")
        return

    for pr in approved_prs:
        pr_number = pr["number"]
        commit_sha = pr["commit"]

        status = get_status(commit_sha)

        if not status:
            start_job(pr_number, commit_sha, pr["author"], force=args.yes)
        else:
            commit_url = f"https://github.com/databricks/cli/pull/{pr_number}/commits/{commit_sha}"
            print(f"Tests already running for PR #{pr_number} {commit_url}")
            print("\n".join(status))
        print()


if __name__ == "__main__":
    main()
