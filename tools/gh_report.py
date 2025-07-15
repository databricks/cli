#!/usr/bin/env python3
"""
Download integration logs artifacts for a given run id (--run RUNID) or commit (--commit) and call gh_parse.py on those to print the report.

If neither --commit nor --run are passed, will use either current PR or HEAD.
"""

import sys
import os
import subprocess
import argparse
import json
import pprint
from pathlib import Path


CLI_REPO = "databricks/cli"
DECO_REPO = os.environ.get("DECO_REPO") or os.environ.get("GITHUB_REPOSITORY")
DECO_TESTS_PREFIX = "https://go/deco-tests/"
CLI_TESTS_PREFIX = "https://github.com/databricks/cli/actions/runs/"
DIRECTORY = Path(__file__).parent
PARSE_SCRIPT = DIRECTORY / "gh_parse.py"

try:
    PARSE_SCRIPT = PARSE_SCRIPT.relative_to(os.getcwd())
except Exception:
    pass  # keep absolute


def run(cmd, shell=False):
    sys.stderr.write("+ " + " ".join(cmd) + "\n")
    return subprocess.run(cmd, check=True, shell=False)


def run_text(cmd, print_command=False):
    if print_command:
        sys.stderr.write("+ " + " ".join(cmd) + "\n")
    result = subprocess.run(cmd, stdout=subprocess.PIPE, encoding="utf-8", check=True)
    return result.stdout.strip()


def run_json(cmd):
    sys.stderr.write("+ " + " ".join(cmd) + "\n")
    result = subprocess.run(cmd, stdout=subprocess.PIPE, encoding="utf-8", check=True)

    try:
        return json.loads(result.stdout)
    except Exception:
        sys.stderr.write(f"Failed to parse JSON:\n{result.stdout}\n")
        raise


def current_branch():
    return run_text("git branch --show-current".split())


def get_run_id_from_items(items, field, prefix, data):
    found = set()

    for item in items or []:
        url = item.get(field, "")
        if url.startswith(prefix):
            run_id = url.removeprefix(prefix).split("/")[0]
            assert run_id.isdigit(), url
            found.add(int(run_id))

    found = sorted(found)

    if not found:
        print(pprint.pformat(data), flush=True, file=sys.stderr)
        sys.exit(f"run_id not found (search: {field=} {prefix=})")
    elif len(found) > 1:
        print(f"many run_ids (search: {field=} {prefix=}): {found}", file=sys.stderr, flush=True)

    return found[-1]


def get_pr_run_id_integration():
    data = run_json("gh pr status --json statusCheckRollup".split())
    items = data.get("currentBranch", {}).get("statusCheckRollup")
    return get_run_id_from_items(items, "targetUrl", DECO_TESTS_PREFIX, data)


def get_commit_run_id_integration(commit):
    data = run_json(["gh", "api", f"repos/databricks/cli/commits/{commit}/status"])
    items = data.get("statuses")
    return get_run_id_from_items(items, "target_url", DECO_TESTS_PREFIX, data)


def get_pr_run_id_unit():
    data = run_json("gh pr status --json statusCheckRollup".split())
    items = data.get("currentBranch", {}).get("statusCheckRollup")
    items = [x for x in items if x.get("workflowName") == "build"]
    return get_run_id_from_items(items, "detailsUrl", CLI_TESTS_PREFIX, data)


def get_commit_run_id_unit(commit):
    data = run_json(["gh", "run", "list", "-c", commit, "--json", "databaseId,workflowName"])
    results = []
    try:
        for item in data:
            if item["workflowName"] == "build":
                results.append(int(item["databaseId"]))
        results.sort()
        assert len(results) == 1, results
    except Exception:
        print(pprint.pformat(data), flush=True, file=sys.stderr)
        if not results:
            raise

    return results[-1]


def download_run_id(run_id, repo, rm):
    target_dir = f".gh-logs/{run_id}"
    if os.path.exists(target_dir):
        if rm:
            run(["rm", "-fr", target_dir])
        else:
            print(
                f"Already exists: {target_dir}. If that directory contains partial results, delete it to re-download: rm -fr .gh-logs/{run_id}",
                file=sys.stderr,
            )
            return target_dir
    cmd = ["gh", "run", "-R", repo, "download", str(run_id), "-D", target_dir]
    run(cmd)
    return target_dir


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--run", type=int, help="Github run_id to load")
    parser.add_argument("--commit", help="Commit to get run_id from. If not set, getting either PR status or most recent commit")
    parser.add_argument("--rm", help="Remove previously downloaded files first", action="store_true")
    parser.add_argument("--filter", help="Filter results by test name (substring match)")
    parser.add_argument("--filter-env", help="Filter results by env name (substring match)")
    parser.add_argument("--output", help="Show output for failing tests", action="store_true")
    parser.add_argument("--markdown", help="Output in GitHub-flavored markdown format", action="store_true")

    # This does not work because we don't store artifacts for unit tests. We could download logs instead but that requires different parsing method:
    # ~/work/cli % gh api -H "Accept: application/vnd.github+json" /repos/databricks/cli/actions/runs/15827411452/logs  > logs.zip
    parser.add_argument("--unit", action="store_true", help="Extract run_id for unit tests rather than integration tests (not working)")
    args = parser.parse_args()

    repo = CLI_REPO if args.unit else DECO_REPO
    assert repo

    if not args.run and not args.commit:
        if current_branch() == "main":
            args.commit = run_text("git rev-parse --short HEAD".split())
        else:
            if args.unit:
                args.run = get_pr_run_id_unit()
            else:
                args.run = get_pr_run_id_integration()

    if args.commit:
        assert not args.run
        if args.unit:
            args.run = get_commit_run_id_unit(args.commit)
        else:
            args.run = get_commit_run_id_integration(args.commit)

    target_dir = download_run_id(args.run, repo, rm=args.rm)
    print(flush=True)
    cmd = [sys.executable, str(PARSE_SCRIPT)]
    if args.filter:
        cmd.extend(["--filter", args.filter])
    if args.filter_env:
        cmd.extend(["--filter-env", args.filter_env])
    if args.output:
        cmd.append("--output")
    if args.markdown:
        cmd.append("--markdown")
    cmd.append(f"{target_dir}")
    run(cmd, shell=True)


if __name__ == "__main__":
    try:
        main()
    except subprocess.CalledProcessError as ex:
        sys.exit(ex)
