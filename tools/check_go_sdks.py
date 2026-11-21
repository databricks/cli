#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# ///
"""Check that the CLI's two Go SDKs and its generated files describe the same release.

The CLI uses both the classic Go SDK (github.com/databricks/databricks-sdk-go) and modules
of the modular Go SDK (github.com/databricks/sdk-go/...). Each SDK release records the
Universe commit it was cut from in .codegen/_last_sha. The daily Go SDK update keeps the
CLI consistent with them, and this check fails when it isn't:

  - .codegen/_openapi_sha, which .codegen/cli.json was generated from, must be the classic
    Go SDK release's commit.
  - climod/.codegen/_last_sha names the sdk-go release that climod/ was generated from. That
    must be a published (tagged) sdk-go release, and every sdk-go module in go.mod must be
    at its version in that release: the top release in the package's CHANGELOG.md at the
    release commit, whose tag must exist, as genkit apply-tag tags it.

The update picks the newest sdk-go release cut from the classic release's commit or an
earlier one. This check can't verify that order, because it needs Universe's history.
Before climod/ is first generated, it has no _last_sha, and only the first check runs.
"""

import json
import os
import subprocess
import sys
import tempfile

CLASSIC_SDK = "github.com/databricks/databricks-sdk-go"
SDK_GO_PREFIX = "github.com/databricks/sdk-go/"
SDK_GO_REPO = "https://github.com/databricks/sdk-go"
RELEASE_MARKER = ".codegen/_last_sha"
CLIMOD_MARKER = "climod/.codegen/_last_sha"
RELEASE_HEADING = "## Release "


def run(*args, cwd=None):
    return subprocess.run(args, cwd=cwd, check=True, capture_output=True, text=True).stdout


def release_tag(clone, release, module_path):
    """Return the tag and version of an sdk-go module in a release, as genkit apply-tag does."""
    directory = module_path.removeprefix(SDK_GO_PREFIX)
    package = json.loads(run("git", "show", f"{release}:{directory}/.package.json", cwd=clone)).get("package", "")
    changelog = run("git", "show", f"{release}:{directory}/CHANGELOG.md", cwd=clone)
    _, found, heading = changelog.partition(RELEASE_HEADING)
    fields = heading.split("\n", 1)[0].split()
    if not found or not fields:
        raise ValueError(f"{directory}/CHANGELOG.md has no release")
    version = fields[0]
    return (f"{package}/{version}" if package else version), version


def find_release(clone, universe_sha):
    """Return the newest tagged sdk-go commit whose release marker is universe_sha, or None."""
    # A release merges its mirror PR, which updates the marker, before it tags the merged
    # commit, so only a tagged commit is a published release.
    tagged = set(run("git", "rev-list", "--no-walk", "--tags", cwd=clone).split())
    # sdk-go releases from its main branch, whose first-parent history lists the newest
    # release first.
    for commit in run("git", "rev-list", "--first-parent", "HEAD", cwd=clone).split():
        if commit not in tagged:
            continue
        # Releases from before the marker existed don't have it.
        if not run("git", "ls-tree", "--name-only", commit, RELEASE_MARKER, cwd=clone).strip():
            continue
        if run("git", "show", f"{commit}:{RELEASE_MARKER}", cwd=clone).strip() == universe_sha:
            return commit
    return None


def check_sdk_go(clone, climod_sha, requires):
    """Return the errors in go.mod's sdk-go modules against the release that climod/ describes."""
    release = find_release(clone, climod_sha)
    if release is None:
        return [f"climod/ was generated from Universe commit {climod_sha}, but no sdk-go release was cut from it."]
    errors = []
    tags = set(run("git", "tag", cwd=clone).split())
    for path, version in sorted(requires.items()):
        if not path.startswith(SDK_GO_PREFIX):
            continue
        tag, want = release_tag(clone, release, path)
        if tag not in tags:
            errors.append(f"The sdk-go release that climod/ describes hasn't published {path} {want} yet.")
        elif want != version:
            errors.append(
                f"go.mod requires {path} {version}, but the sdk-go release that climod/ describes has {want}. "
                f"Pin {want}, or let the Go SDK update move climod/ to a newer sdk-go release."
            )
    return errors


def main():
    errors = []
    requires = {r["Path"]: r["Version"] for r in json.loads(run("go", "mod", "edit", "-json")).get("Require", [])}

    classic = json.loads(run("go", "mod", "download", "-json", CLASSIC_SDK))
    with open(os.path.join(classic["Dir"], RELEASE_MARKER)) as f:
        classic_sha = f.read().strip()
    with open(".codegen/_openapi_sha") as f:
        openapi_sha = f.read().strip()
    if openapi_sha != classic_sha:
        errors.append(
            f".codegen/_openapi_sha is {openapi_sha}, but {CLASSIC_SDK} {classic['Version']} was released "
            f"from Universe commit {classic_sha}. Run the Go SDK update or ./task generate-clijson."
        )

    if not os.path.exists(CLIMOD_MARKER):
        print(f"{CLIMOD_MARKER} doesn't exist yet, so the sdk-go modules aren't checked.")
    else:
        with open(CLIMOD_MARKER) as f:
            climod_sha = f.read().strip()
        with tempfile.TemporaryDirectory() as clone:
            run("git", "clone", "--quiet", "--filter=blob:none", "--no-checkout", SDK_GO_REPO, clone)
            errors += check_sdk_go(clone, climod_sha, requires)

    for error in errors:
        print(error, file=sys.stderr)
    return 1 if errors else 0


if __name__ == "__main__":
    sys.exit(main())
