#!/usr/bin/env python3
"""
Contract check for edit_fuzz_config's field selection (the harness diffs stdout; a
non-zero exit marks a violation on stderr):

- A comment/description that recreates on change for its resource type (per
  resources.yml, e.g. model_serving_endpoints.description) is never chosen, so the
  update invariant does not assert an in-place update the backend cannot perform.
- A mutable comment/description is still chosen, even when an immutable one appears
  first.
- The immutable map actually loads and reflects resources.yml.
"""

import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from edit_fuzz_config import find_line, immutable_fields

IMMUTABLE_ONLY = """\
resources:
  model_serving_endpoints:
    foo:
      name: "test-endpoint"
      description: "old"
"""

IMMUTABLE_THEN_MUTABLE = """\
resources:
  model_serving_endpoints:
    foo:
      description: "immutable"
  jobs:
    bar:
      description: "mutable"
"""

MUTABLE_ONLY = """\
resources:
  jobs:
    bar:
      description: "mutable"
"""


def choose(text, immutable):
    i, m = find_line(text.splitlines(keepends=True), immutable)
    return None if m is None else i


def main():
    immutable = immutable_fields()
    failed = False

    # Guards the loader and the classification the update invariant relies on.
    serving_immutable = "description" in immutable.get("model_serving_endpoints", set())
    if not serving_immutable:
        sys.stderr.write("expected model_serving_endpoints.description to be immutable in resources.yml\n")
        failed = True

    if choose(IMMUTABLE_ONLY, immutable) is not None:
        sys.stderr.write("picked an immutable description\n")
        failed = True

    if choose(IMMUTABLE_THEN_MUTABLE, immutable) != 6:
        sys.stderr.write("expected to skip the immutable description and pick the mutable one\n")
        failed = True

    if choose(MUTABLE_ONLY, immutable) != 3:
        sys.stderr.write("expected to pick the mutable description\n")
        failed = True

    print(f"model_serving_endpoints.description immutable: {serving_immutable}")
    print(f"IMMUTABLE_ONLY: {choose(IMMUTABLE_ONLY, immutable)}")
    print(f"IMMUTABLE_THEN_MUTABLE: {choose(IMMUTABLE_THEN_MUTABLE, immutable)}")
    print(f"MUTABLE_ONLY: {choose(MUTABLE_ONLY, immutable)}")

    if failed:
        sys.exit(1)


if __name__ == "__main__":
    main()
