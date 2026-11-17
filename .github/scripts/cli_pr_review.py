#!/usr/bin/env python3

import json
import os
import subprocess
from pathlib import Path
from urllib.parse import quote

REPOSITORY = "databricks/cli"
PR_ACTIONS = {"opened", "reopened", "synchronize", "ready_for_review"}
COMMENT_ASSOCIATIONS = {"MEMBER", "OWNER", "COLLABORATOR"}
WRITE_PERMISSIONS = {"write", "maintain", "admin"}


def github_api(path):
    return json.loads(subprocess.check_output(["gh", "api", path], text=True))


def validate_request(event_name, event, api=github_api):
    if event["repository"]["full_name"] != REPOSITORY:
        return None

    requester = None
    if event_name == "pull_request_target":
        if event["action"] not in PR_ACTIONS:
            return None
        number = event["number"]
        expected_head = event["pull_request"]["head"]["sha"]
    elif event_name == "issue_comment":
        comment = event["comment"]
        if (
            event["action"] != "created"
            or "pull_request" not in event["issue"]
            or comment["body"] != "/review"
            or comment["user"]["type"] != "User"
            or comment["author_association"] not in COMMENT_ASSOCIATIONS
        ):
            return None
        number = event["issue"]["number"]
        requester = comment["user"]["login"]
    else:
        return None

    if type(number) is not int or number <= 0:
        raise ValueError("Expected a positive PR number")
    pr = api(f"repos/{REPOSITORY}/pulls/{number}")
    head_repo = pr["head"]["repo"]
    if (
        pr["state"] != "open"
        or pr["base"]["ref"] != "main"
        or head_repo is None
        or head_repo["full_name"] != REPOSITORY
        or head_repo["fork"]
        or pr["user"]["type"] != "User"
        or (event_name == "pull_request_target" and (pr["draft"] or pr["head"]["sha"] != expected_head))
    ):
        return None

    # Association labels do not establish current write access.
    # https://docs.github.com/en/rest/collaborators/collaborators#get-repository-permissions-for-a-user
    users = {pr["user"]["login"]}
    if requester:
        users.add(requester)
    for user in sorted(users):
        permission = api(f"repos/{REPOSITORY}/collaborators/{quote(user, safe='')}/permission")
        if permission["permission"] not in WRITE_PERMISSIONS:
            return None
    return number


def main():
    event = json.loads(Path(os.environ["GITHUB_EVENT_PATH"]).read_text())
    number = validate_request(os.environ["GITHUB_EVENT_NAME"], event)
    if number is None:
        with Path(os.environ["GITHUB_STEP_SUMMARY"]).open("a") as summary:
            summary.write("Review request is ineligible or superseded; skipping review.\n")
        return
    with Path(os.environ["GITHUB_OUTPUT"]).open("a") as output:
        output.write(f"eligible=true\npr_number={number}\n")


if __name__ == "__main__":
    main()
