#!/usr/bin/env python3
"""
Apply a single field mutation to a rendered databricks.yml.

Mutations are declared in the config as trailing comments, so they are inert YAML
and do not affect the invariant targets that read the same config without mutating
it:

    description: hello              # ACTION: REMOVE
    #user_api_scopes: [iam.access]  # ACTION: UNCOMMENT
    #compute_size: LARGE            # ACTION: SET XLARGE

Each mutation is named "<key>:<action>", e.g. "description:remove".

REMOVE and UNCOMMENT only work for a field the resource's update path can clear:
both targets return the remote to the base config, and for those two actions that
means clearing the field. A resource whose update request drops empty fields (an
omitempty payload without ForceSendFields, e.g. UpdateSchema) never converges, so
use SET there.

EXPECT names the plan action the mutation should produce, defaulting to
"non-skip". A field the engine deliberately ignores expects "skip" instead, and
which of the two targets sees that depends on why it is ignored -- an
ignore_local_changes field is skipped when the config changes, an
ignore_remote_changes field when the remote drifts. So the expectation can be set
for both targets at once or for one of them:

    EXPECT: skip                 both targets
    EXPECT_UPDATE: skip          apply_update only
    EXPECT_REMOTE_UPDATE: skip   apply_remote_update only

Usage:
    apply_mutation.py FILE --list [--expect-key KEY]   print "<name> <expect>" per mutation
    apply_mutation.py FILE NAME                        apply mutation NAME in place
"""

import argparse
import re
import sys

MARKER = "# ACTION:"
DEFAULT_EXPECT = "non-skip"
GENERIC_EXPECT_KEY = "EXPECT"
EXPECT_KEYS = (GENERIC_EXPECT_KEY, "EXPECT_UPDATE", "EXPECT_REMOTE_UPDATE")

# Splits "SET XLARGE EXPECT_UPDATE: skip" into the action part and the key/value pairs.
EXPECT_RE = re.compile(r"\b(" + "|".join(EXPECT_KEYS) + r"):\s*(\S+)")

REMOVE = "REMOVE"
UNCOMMENT = "UNCOMMENT"
SET = "SET"
ACTIONS = (REMOVE, UNCOMMENT, SET)


class Mutation:
    def __init__(self, lineno, indent, body, key, action, arg, expects):
        self.lineno = lineno
        self.indent = indent
        self.body = body
        self.key = key
        self.action = action
        self.arg = arg
        self.expects = expects

    @property
    def name(self):
        return f"{self.key}:{self.action.lower()}"

    def expect(self, expect_key):
        for candidate in (expect_key, GENERIC_EXPECT_KEY):
            if candidate in self.expects:
                return self.expects[candidate]
        return DEFAULT_EXPECT

    def render(self):
        """Return the replacement line, or None when the line is dropped."""
        if self.action == REMOVE:
            return None
        if self.action == UNCOMMENT:
            return self.indent + self.body
        return f"{self.indent}{self.key}: {self.arg}"


def parse_line(lineno, line):
    if MARKER not in line:
        return None

    head, _, spec = line.partition(MARKER)

    expects = dict(EXPECT_RE.findall(spec))
    spec = EXPECT_RE.sub("", spec)

    action, _, arg = spec.strip().partition(" ")
    action = action.upper()
    arg = arg.strip()

    indent = head[: len(head) - len(head.lstrip())]
    body = head.strip()
    commented = body.startswith("#")
    if commented:
        body = body[1:].lstrip()

    key = body.partition(":")[0].strip()

    def fail(message):
        sys.exit(f"{lineno}: {message}: {line.strip()!r}")

    if action not in ACTIONS:
        fail(f"unknown action {action!r}, expected one of {', '.join(ACTIONS)}")
    if not key:
        fail("cannot determine field name")
    if action == REMOVE and commented:
        fail("REMOVE on a commented-out line is a no-op; use SET or drop the annotation")
    if action == UNCOMMENT and not commented:
        fail("UNCOMMENT on an active line is a no-op; use REMOVE or comment the line out")
    if action == SET and not arg:
        fail("SET requires a value")

    return Mutation(lineno, indent, body, key, action, arg, expects)


def parse(filename):
    with open(filename) as fobj:
        lines = fobj.read().splitlines(keepends=True)

    mutations = []
    seen = {}
    for lineno, line in enumerate(lines, start=1):
        mutation = parse_line(lineno, line)
        if mutation is None:
            continue
        if mutation.name in seen:
            sys.exit(f"{lineno}: duplicate mutation {mutation.name!r}, first seen on line {seen[mutation.name]}")
        seen[mutation.name] = lineno
        mutations.append(mutation)

    return lines, mutations


def apply(filename, lines, mutation):
    replacement = mutation.render()
    index = mutation.lineno - 1
    if replacement is None:
        del lines[index]
    else:
        lines[index] = replacement + "\n"
    with open(filename, "w") as fobj:
        fobj.write("".join(lines))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("filename")
    parser.add_argument("name", nargs="?")
    parser.add_argument("--list", action="store_true")
    parser.add_argument("--expect-key", default=GENERIC_EXPECT_KEY, choices=EXPECT_KEYS)
    args = parser.parse_args()

    lines, mutations = parse(args.filename)

    if args.list:
        for mutation in mutations:
            print(mutation.name, mutation.expect(args.expect_key))
        return

    if not args.name:
        parser.error("either NAME or --list is required")

    for mutation in mutations:
        if mutation.name == args.name:
            apply(args.filename, lines, mutation)
            return

    available = ", ".join(m.name for m in mutations) or "(none)"
    sys.exit(f"mutation {args.name!r} not found in {args.filename}; available: {available}")


if __name__ == "__main__":
    main()
