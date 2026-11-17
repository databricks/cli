#!/usr/bin/env python3
"""Allow a review only when the PR and the people requesting it pass every check.

This script only reads GitHub data. The next workflow job starts the review.
"""

import json
import os
import subprocess
from pathlib import Path
from urllib.parse import quote

REPOSITORY = "databricks/cli"
PR_ACTIONS = {"opened", "reopened", "ready_for_review"}
WRITE_PERMISSIONS = {"write", "maintain", "admin"}


def github_api(path):
    """Use gh's existing login; an API failure stops validation instead of allowing access."""
    return json.loads(subprocess.check_output(["gh", "api", path], text=True))


def validate_request(event_name, event, api=github_api):
    """Return the PR number if allowed, or None to skip. Tests supply a fake API."""
    if event["repository"]["full_name"] != REPOSITORY:
        return None

    # 1. Identify the request: an automatic PR event or an explicit /review comment.
    requester = None
    if event_name == "pull_request_target":
        if event["action"] not in PR_ACTIONS:
            return None
        number = event["number"]
        expected_head = event["pull_request"]["head"]["sha"]
        expected_base = event["pull_request"]["base"]["ref"]
    elif event_name == "issue_comment":
        comment = event["comment"]
        # Comments on ordinary issues, edited commands, and extra instructions do not count.
        if (
            event["action"] != "created"
            or "pull_request" not in event["issue"]
            or comment["body"].strip() != "/review"
            or comment["user"]["type"] != "User"
        ):
            return None
        number = event["issue"]["number"]
        requester = comment["user"]["login"]
    else:
        return None

    if type(number) is not int or number <= 0:
        raise ValueError("Expected a positive PR number")
    # 2. Fetch the current PR. It may have changed while this event waited for a runner.
    pr = api(f"repos/{REPOSITORY}/pulls/{number}")
    head_repo = pr["head"]["repo"]
    if (
        pr["state"] != "open"
        or head_repo is None
        or head_repo["full_name"] != REPOSITORY
        or head_repo["fork"]
        or pr["user"]["type"] != "User"
        # A manual /review can review a draft or the latest commit; an old automatic event cannot.
        or (
            event_name == "pull_request_target"
            and (pr["draft"] or pr["head"]["sha"] != expected_head or pr["base"]["ref"] != expected_base)
        )
    ):
        return None

    # 3. Check actual write access for the author and, for /review, the commenter too.
    # GitHub can label a writer CONTRIBUTOR, so author_association is not a permission gate.
    # https://docs.github.com/en/rest/collaborators/collaborators#get-repository-permissions-for-a-user
    users = {pr["user"]["login"]}
    if requester:
        # A set checks the author only once when they request their own review.
        users.add(requester)
    for user in sorted(users):
        permission = api(f"repos/{REPOSITORY}/collaborators/{quote(user, safe='')}/permission")
        if permission["permission"] not in WRITE_PERMISSIONS:
            return None
    return number


def main():
    """Read GitHub's event file and tell the next job whether it may proceed."""
    event = json.loads(Path(os.environ["GITHUB_EVENT_PATH"]).read_text())
    number = validate_request(os.environ["GITHUB_EVENT_NAME"], event)
    if number is None:
        # An ineligible request is a normal skip, not a broken workflow.
        with Path(os.environ["GITHUB_STEP_SUMMARY"]).open("a") as summary:
            summary.write("Review request is ineligible or superseded; skipping review.\n")
        return
    # These outputs unlock the credential-bearing job. Write them only after all checks pass.
    with Path(os.environ["GITHUB_OUTPUT"]).open("a") as output:
        output.write(f"eligible=true\npr_number={number}\n")


if __name__ == "__main__":
    main()
