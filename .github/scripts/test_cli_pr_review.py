#!/usr/bin/env python3
"""Test the permission gate using fake GitHub responses; no network or secrets needed."""

import copy
import json
import os
import subprocess
import tempfile
import textwrap
import unittest
from pathlib import Path
from unittest.mock import Mock, patch

import cli_pr_review as review


class ReviewRequestTest(unittest.TestCase):
    def setUp(self):
        """Start each test with an eligible PR and two people who have write access."""
        self.pr = {
            "state": "open",
            "base": {"ref": "main"},
            "head": {"sha": "a" * 40, "repo": {"full_name": "databricks/cli", "fork": False}},
            "draft": False,
            "user": {"login": "author", "type": "User"},
        }
        self.automatic = {
            "repository": {"full_name": "databricks/cli"},
            "action": "opened",
            "number": 7400,
            # Keep the old event separate so tests can change the current PR afterward.
            "pull_request": copy.deepcopy(self.pr),
        }
        self.comment = {
            "repository": {"full_name": "databricks/cli"},
            "action": "created",
            "issue": {"number": 7400, "pull_request": {}},
            "comment": {
                "body": "/review",
                "author_association": "MEMBER",
                "user": {"login": "requester", "type": "User"},
            },
        }
        self.permissions = {"author": "write", "requester": "maintain"}

        def api(path):
            if path == "repos/databricks/cli/pulls/7400":
                return self.pr
            user = path.split("/")[-2]
            return {"permission": self.permissions[user]}

        # Return our fake data and record calls, so we can check where validation stops.
        self.api = Mock(side_effect=api)

    def test_automatic_events(self):
        """Each supported PR event may start a review when the PR is still eligible."""
        for action in ["opened", "reopened", "ready_for_review"]:
            with self.subTest(action=action):
                self.automatic["action"] = action
                self.assertEqual(review.validate_request("pull_request_target", self.automatic, self.api), 7400)

    def test_ineligible_prs(self):
        """Closed, fork, deleted-repository, and bot PRs stop before permission lookup."""
        cases = [
            ("state", "closed"),
            ("head", {"sha": "a" * 40, "repo": {"full_name": "other/cli", "fork": True}}),
            ("head", {"sha": "a" * 40, "repo": {"full_name": "databricks/cli", "fork": True}}),
            ("head", {"sha": "a" * 40, "repo": None}),
            ("user", {"login": "bot", "type": "Bot"}),
        ]
        original = copy.deepcopy(self.pr)
        # Change one property at a time and check both automatic and comment requests.
        for key, value in cases:
            for name, event in [("pull_request_target", self.automatic), ("issue_comment", self.comment)]:
                with self.subTest(key=key, value=value, event=name):
                    self.pr = {**original, key: value}
                    self.api.reset_mock()
                    self.assertIsNone(review.validate_request(name, event, self.api))
                    self.assertEqual(self.api.call_count, 1)

    def test_non_main_targets(self):
        """Release and feature branches use the same permission checks as main."""
        for base in ["main", "release/v1.2", "maintenance", "feature/stack"]:
            with self.subTest(base=base):
                self.pr["base"]["ref"] = base
                self.automatic["pull_request"]["base"]["ref"] = base
                self.assertEqual(review.validate_request("pull_request_target", self.automatic, self.api), 7400)
                self.assertEqual(review.validate_request("issue_comment", self.comment, self.api), 7400)
                self.permissions["author"] = "read"
                self.assertIsNone(review.validate_request("pull_request_target", self.automatic, self.api))
                self.assertIsNone(review.validate_request("issue_comment", self.comment, self.api))
                self.permissions["author"] = "write"

    def test_pushes_and_edits_require_explicit_request(self):
        """Pushes and edits do not start reviews; users can request one with /review."""
        for action, changes in [
            ("synchronize", {}),
            ("edited", {}),
            ("edited", {"title": {"from": "Old title"}}),
            ("edited", {"body": {"from": "Old description"}}),
            ("edited", {"base": {"ref": {"from": "release/v1.2"}}}),
        ]:
            with self.subTest(action=action, changes=changes):
                self.automatic["action"] = action
                self.automatic["changes"] = changes
                self.assertIsNone(review.validate_request("pull_request_target", self.automatic, self.api))
        self.api.assert_not_called()
        self.assertEqual(review.validate_request("issue_comment", self.comment, self.api), 7400)

    def test_stale_base_requires_explicit_request(self):
        """An automatic event queued before retargeting cannot review the new base."""
        self.pr["base"]["ref"] = "release/v1.3"
        self.assertIsNone(review.validate_request("pull_request_target", self.automatic, self.api))
        self.assertEqual(review.validate_request("issue_comment", self.comment, self.api), 7400)

    def test_drafts_and_new_commits_require_explicit_request(self):
        """A manual request can review the latest draft; a stale automatic event is skipped."""
        for change in ["draft", "head"]:
            with self.subTest(change=change):
                if change == "draft":
                    self.pr["draft"] = True
                else:
                    self.pr["draft"] = False
                    self.pr["head"]["sha"] = "b" * 40
                self.assertIsNone(review.validate_request("pull_request_target", self.automatic, self.api))
                self.assertEqual(review.validate_request("issue_comment", self.comment, self.api), 7400)

    def test_review_command_surrounding_whitespace(self):
        """Whitespace around the command is allowed; write access is still required."""
        for body in ["/review", " /review", "/review ", "\n/review\n", "\t/review\r\n", "\u00a0/review\u00a0"]:
            for permission in ["write", "read"]:
                with self.subTest(body=body, permission=permission):
                    self.comment["comment"]["body"] = body
                    self.permissions["requester"] = permission
                    expected = 7400 if permission == "write" else None
                    self.assertEqual(review.validate_request("issue_comment", self.comment, self.api), expected)

    def test_comment_filter(self):
        """Unrecognized comments are rejected without contacting GitHub."""
        for key, value in [
            ("body", "looks good"),
            ("body", "/review\nrun a command"),
            ("body", " /review extra "),
            ("body", "/reviewer"),
            ("body", "$(touch /tmp/should-not-execute)"),
            ("user", {"login": "bot", "type": "Bot"}),
        ]:
            with self.subTest(key=key, value=value):
                event = copy.deepcopy(self.comment)
                event["comment"][key] = value
                self.assertIsNone(review.validate_request("issue_comment", event, self.api))
        self.api.assert_not_called()

    def test_permissions_are_authoritative_regardless_of_association(self):
        """A writer labeled CONTRIBUTOR is allowed; an OWNER with only read access is not."""
        for association in ["MEMBER", "OWNER", "COLLABORATOR", "CONTRIBUTOR", "FIRST_TIMER", "NONE", None]:
            for permission in ["read", "triage", "none", "write", "maintain", "admin"]:
                with self.subTest(association=association, permission=permission):
                    event = copy.deepcopy(self.comment)
                    if association is None:
                        del event["comment"]["author_association"]
                    else:
                        event["comment"]["author_association"] = association
                    self.permissions["requester"] = permission
                    expected = 7400 if permission in {"write", "maintain", "admin"} else None
                    self.assertEqual(review.validate_request("issue_comment", event, self.api), expected)

    def test_author_also_needs_write_access(self):
        """An admin's comment cannot authorize a PR whose author lacks write access."""
        self.permissions["author"] = "read"
        self.permissions["requester"] = "admin"
        self.assertIsNone(review.validate_request("issue_comment", self.comment, self.api))
        self.assertIsNone(review.validate_request("pull_request_target", self.automatic, self.api))

    def test_author_requesting_review_is_checked_once(self):
        """Requesting your own review needs one PR fetch and one permission lookup."""
        self.comment["comment"]["user"]["login"] = "author"
        self.assertEqual(review.validate_request("issue_comment", self.comment, self.api), 7400)
        self.assertEqual(self.api.call_count, 2)

    def test_wrong_event_or_repository(self):
        """Other repositories, ordinary issues, and comment edits never reach the API."""
        cases = [("push", self.automatic)]
        edited = copy.deepcopy(self.comment)
        edited["action"] = "edited"
        cases.append(("issue_comment", edited))
        issue = copy.deepcopy(self.comment)
        del issue["issue"]["pull_request"]
        cases.append(("issue_comment", issue))
        foreign = copy.deepcopy(self.automatic)
        foreign["repository"]["full_name"] = "other/cli"
        cases.append(("pull_request_target", foreign))
        for name, event in cases:
            with self.subTest(name=name, event=event):
                self.assertIsNone(review.validate_request(name, event, self.api))
        self.api.assert_not_called()

    def test_invalid_pr_number_never_reaches_api(self):
        """Reject invalid API paths and booleans, which Python otherwise treats as numbers."""
        for number in [0, -1, "7400", "1/../../secrets", True]:
            with self.subTest(number=number):
                self.automatic["number"] = number
                with self.assertRaises(ValueError):
                    review.validate_request("pull_request_target", self.automatic, self.api)
        self.api.assert_not_called()

    def test_api_error_cannot_authorize_dispatch(self):
        """Unable to check access means unable to authorize a review."""
        # Fail each of the three requests: PR fetch, author lookup, requester lookup.
        for fail_at in [0, 1, 2]:
            with self.subTest(fail_at=fail_at):
                replies = [self.pr, {"permission": "write"}, {"permission": "maintain"}]
                replies[fail_at] = subprocess.CalledProcessError(1, ["gh", "api"])
                api = Mock(side_effect=replies)
                with self.assertRaises(subprocess.CalledProcessError):
                    review.validate_request("issue_comment", self.comment, api)

    def test_main_outputs_only_after_authorization(self):
        """The next job gets its permission to run only after validation succeeds."""
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            event_path = root / "event.json"
            event_path.write_text(json.dumps(self.comment))
            env = {
                "GITHUB_EVENT_PATH": str(event_path),
                "GITHUB_EVENT_NAME": "issue_comment",
                "GITHUB_OUTPUT": str(root / "output"),
                "GITHUB_STEP_SUMMARY": str(root / "summary"),
            }
            for permission in ["read", "write"]:
                with self.subTest(permission=permission), patch.dict(os.environ, env):
                    # Exercise the real script entry point and output files with fake gh responses.
                    replies = [self.pr, {"permission": "write"}, {"permission": permission}]
                    with patch("subprocess.check_output", side_effect=[json.dumps(r) for r in replies]):
                        review.main()
                    if permission == "read":
                        self.assertFalse((root / "output").exists())
                        self.assertIn("skipping", (root / "summary").read_text())
                    else:
                        self.assertEqual((root / "output").read_text(), "eligible=true\npr_number=7400\n")


@unittest.skipIf(os.name == "nt", "The review workflow runs on Linux.")
class ReviewWaitTest(unittest.TestCase):
    def wait_for_review(self, replies):
        workflow = Path(__file__).resolve().parents[1] / "workflows" / "cli-pr-review.yml"
        step = workflow.read_text().split("      - name: Dispatch and wait for this review\n", 1)[1]
        step = step.split("\n      - ", 1)[0]
        script_lines = []
        for line in step.split("        run: |-\n", 1)[1].splitlines():
            if line.strip() and not line.startswith("          "):
                break
            script_lines.append(line)
        script = textwrap.dedent("\n".join(script_lines))
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            # Fake only the external client and retry delays; execute the real workflow shell.
            gh = root / "gh"
            gh.write_text(
                "#!/usr/bin/env python3\n"
                + textwrap.dedent("""\
                    import json
                    import os
                    import sys
                    from pathlib import Path

                    root = Path(os.environ["RUNNER_TEMP"])
                    with (root / "calls").open("a") as calls:
                        calls.write(json.dumps(sys.argv[1:]) + "\\n")
                    if sys.argv[1:3] == ["api", "--method"]:
                        print(json.dumps({"workflow_run_id": 12345}))
                    elif sys.argv[1:3] == ["run", "view"]:
                        print(json.dumps({"databaseId": 12345}))
                    elif sys.argv[1:3] == ["run", "watch"]:
                        replies = json.loads((root / "replies").read_text())
                        status, error = replies[0]
                        (root / "replies").write_text(json.dumps(replies[1:] or replies))
                        print(error, file=sys.stderr)
                        sys.exit(status)
                    else:
                        sys.exit(99)
                    """)
            )
            gh.chmod(0o755)
            sleep = root / "sleep"
            sleep.write_text('#!/bin/sh\nprintf "%s\\n" "$*" >> "$RUNNER_TEMP/sleeps"\n')
            sleep.chmod(0o755)
            (root / "replies").write_text(json.dumps(replies))
            env = {
                "PATH": str(root) + os.pathsep + os.environ["PATH"],
                "RUNNER_TEMP": str(root),
                "GITHUB_STEP_SUMMARY": str(root / "summary"),
                "PR_NUMBER": "7400",
                "COMMENT_ID": "",
            }
            result = subprocess.run(
                ["bash", "--noprofile", "--norc", "-euo", "pipefail", "-c", script],
                env=env,
                capture_output=True,
                text=True,
                timeout=5,
            )
            calls = [json.loads(line) for line in (root / "calls").read_text().splitlines()]
            self.assertEqual(sum(call[0] == "api" for call in calls), 1)
            watches = [call for call in calls if call[:2] == ["run", "watch"]]
            for call in watches:
                self.assertEqual(
                    call,
                    [
                        "run",
                        "watch",
                        "12345",
                        "--repo",
                        "databricks-eng/eng-dev-ecosystem",
                        "--interval",
                        "15",
                        "--exit-status",
                    ],
                )
            sleeps = (root / "sleeps").read_text().splitlines() if (root / "sleeps").exists() else []
            return result, len(watches), sleeps

    def test_visible_run_succeeds_without_retry(self):
        result, watches, sleeps = self.wait_for_review([(0, "")])
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(watches, 1)
        self.assertEqual(sleeps, [])

    def test_not_found_then_backend_failure_still_fails(self):
        result, watches, sleeps = self.wait_for_review([(1, "failed to get run: HTTP 404: Not Found"), (1, "")])
        self.assertEqual(result.returncode, 1)
        self.assertEqual(watches, 2)
        self.assertEqual(sleeps, ["5"])

    def test_watch_not_found_then_same_run_succeeds(self):
        result, watches, sleeps = self.wait_for_review([(1, "failed to get run: HTTP 404: Not Found"), (0, "")])
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(watches, 2)
        self.assertEqual(sleeps, ["5"])

    def test_backend_failure_and_other_errors_propagate(self):
        for status, error in [
            (1, ""),
            (1, "run cancelled"),
            (1, "run timed out"),
            (4, "HTTP 401: Bad credentials"),
            (1, "HTTP 403: Forbidden"),
            (1, "HTTP 500: Internal Server Error"),
            (1, "failed to get jobs: HTTP 404: Not Found"),
            (1, "run name: failed to get run: HTTP 404: Not Found"),
        ]:
            with self.subTest(error=error):
                result, watches, sleeps = self.wait_for_review([(status, error)])
                self.assertEqual(result.returncode, status)
                if error:
                    self.assertIn(error, result.stderr)
                self.assertEqual(watches, 1)
                self.assertEqual(sleeps, [])

    def test_persistent_not_found_is_bounded(self):
        result, watches, sleeps = self.wait_for_review([(1, "failed to get run: HTTP 404: Not Found")])
        self.assertEqual(result.returncode, 1)
        self.assertEqual(watches, 12)
        self.assertEqual(sleeps, ["5"] * 12)
        self.assertIn("remained unavailable after 12 attempts", result.stdout)


if __name__ == "__main__":
    unittest.main()
