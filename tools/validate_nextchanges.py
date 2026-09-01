#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# ///
"""Validate changelog fragment placement under ``.nextchanges/``.

Each PR adds its own file under ``.nextchanges/<section>/`` (see
``.nextchanges/README.md``). This fails CI when a fragment is misfiled or
empty, so it is caught up front rather than silently dropped when the release
renders ``.nextchanges/`` into ``CHANGELOG.md`` (see
``internal/genkit/tagging.py``).
"""

import argparse
import json
import os
import pathlib
import re
import subprocess
import sys

CHANGELOG_DIR = ".nextchanges"
CODEGEN_FILE = ".codegen.json"
NEXTCHANGES_SECTIONS_KEY = "nextchanges_sections"

# .nextchanges/version holds the next release version; the release reads it and
# bumps it. Accept a bare semver (optionally v-prefixed), e.g. 1.4.0 / v1.4.0.
VERSION_FILE = "version"
SEMVER_RE = re.compile(r"^v?\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$")

# README.md is allowed both at the .nextchanges root (the docs) and inside each
# section directory: the release renderer skips it, so a committed README.md
# keeps otherwise-empty section directories present in git without being
# mistaken for a fragment.
README = "README.md"

# nextversion.go embeds the version file above so the build can report the next
# release version. It lives here because go:embed cannot reach a parent
# directory, and keeping it here avoids a second copy of the version that could
# drift. The release renderer only reads *.md fragments, so it ignores this.
NEXTVERSION_GO = "nextversion.go"

# A fragment is a single changelog entry: one line that starts with "* " and
# ends with a period. An optional trailing PR link group "([#N](pull-url))" — or
# a comma-separated list "([#N](…), [#M](…))" for entries spanning several PRs —
# may follow the period. Every "#N" reference must be written as a full markdown
# link: a bare or paren-wrapped "#N" would render as an unintended auto-link in
# CHANGELOG.md, and nothing expands links anymore.
BULLET_PREFIX = "* "

# The trailing PR link group: a parenthesized, comma-separated list of markdown
# links at the very end of the entry, e.g. "([#12](…), [#34](…))". Matched
# loosely (any "[..](..)" link) so a malformed link inside still makes the group
# recognizable — it is then reported as a link error, rather than misfiring as
# "must end with a period" because a strict pattern failed to match.
_LINK = r"\[[^\]]*\]\([^)]*\)"
TRAILING_GROUP_RE = re.compile(rf" \((?P<links>{_LINK}(?:, {_LINK})*)\)$")
LINK_RE = re.compile(_LINK)

# A "#N" not preceded by "[" is a raw, unexpanded reference (bare, or wrapped in
# parens); the "[#N]" of a markdown link is preceded by "[" and so is excluded.
RAW_REF_RE = re.compile(r"(?<!\[)#\d+")
# A well-formed PR link "[#N](…/pull/N)", capturing both numbers so they can be
# compared and, via fullmatch, to tell a valid PR link from a malformed one.
PR_LINK_RE = re.compile(r"\[#(\d+)\]\(https://github\.com/databricks/cli/pull/(\d+)\)")


def fragment_format_problem(text):
    r"""Return why ``text`` is not a valid fragment, or ``None`` if it is.

    A fragment is exactly one changelog entry: a single non-empty line that
    starts with ``"* "`` and ends with ``"."``. An optional trailing PR link
    group (``([#N](pull-url))``, or a comma-separated list) may follow the period.

    >>> fragment_format_problem("* Added the `foo` command.")
    >>> fragment_format_problem("* Fixed a bug. ([#6208](https://github.com/databricks/cli/pull/6208))")
    >>> fragment_format_problem("Added the `foo` command.")
    'must start with a "* " bullet marker'
    >>> fragment_format_problem("* Added the `foo` command")
    'must end with a period'
    >>> fragment_format_problem("* First entry.\n* Second entry.")
    'must be a single entry on one line'
    >>> fragment_format_problem("   ")
    'empty fragment'
    """
    stripped = text.strip()
    if not stripped:
        return "empty fragment"
    if "\n" in stripped:
        return "must be a single entry on one line"
    if not stripped.startswith(BULLET_PREFIX):
        return 'must start with a "* " bullet marker'
    # The trailing PR link group follows the period; ignore it when checking
    # that the entry text itself ends with a period. Matched loosely so a
    # malformed link inside doesn't hide the group and misfire here — pr_link
    # _problem validates the links.
    if not TRAILING_GROUP_RE.sub("", stripped).endswith("."):
        return "must end with a period"
    return None


def link_problem(text):
    r"""Return a problem with the ``#`` references in ``text``, or ``None``.

    Every reference must be a full markdown link; a bare or paren-wrapped ``#N``
    (which GitHub would auto-link in the rendered CHANGELOG.md) is rejected. A PR
    link's text number and URL number must also agree.

    >>> link_problem("* Fixed a bug. ([#5](https://github.com/databricks/cli/pull/5))")
    >>> link_problem("* Fixed a bug (#5).")
    'unexpanded reference #5: write it as a markdown link, e.g. [#5](https://github.com/databricks/cli/pull/5)'
    >>> link_problem("* Reverts #7 for now.")
    'unexpanded reference #7: write it as a markdown link, e.g. [#7](https://github.com/databricks/cli/pull/7)'
    >>> link_problem("* Oops. ([#5](https://github.com/databricks/cli/pull/9))")
    'PR link text #5 does not match its URL (pull/9)'
    """
    m = RAW_REF_RE.search(text)
    if m:
        ref = m.group(0)
        return f"unexpanded reference {ref}: write it as a markdown link, e.g. [{ref}](https://github.com/databricks/cli/pull/{ref[1:]})"
    for lm in PR_LINK_RE.finditer(text):
        if lm.group(1) != lm.group(2):
            return f"PR link text #{lm.group(1)} does not match its URL (pull/{lm.group(2)})"
    return None


def pr_link_problem(text, require_pr_link, expected_pr):
    r"""Return a problem with ``text``'s trailing PR link group, or ``None``.

    The group is recognized loosely, then each link must be a well-formed PR link
    (a malformed URL is reported as such). ``expected_pr`` is the PR that
    introduced the fragment (see ``infer_expected_pr``); it must appear among the
    linked PRs, so an entry may also list follow-up PRs. ``require_pr_link`` makes
    the group mandatory — set whenever the change is associated with a PR (see
    ``main``). Text/URL number agreement is checked by ``link_problem``.

    >>> pr_link_problem("* A change.", False, None)
    >>> pr_link_problem("* A change.", True, "5")
    'missing trailing PR link: end with ([#5](https://github.com/databricks/cli/pull/5))'
    >>> pr_link_problem("* A change.", True, None)
    'missing trailing PR link: end with ([#<PR>](https://github.com/databricks/cli/pull/<PR>))'
    >>> pr_link_problem("* A change. ([#6177](https://github.com/databricks/cli/6177))", True, "6177")
    'malformed trailing PR link "[#6177](https://github.com/databricks/cli/6177)": expected [#N](https://github.com/databricks/cli/pull/N)'
    >>> pr_link_problem("* A change. ([#5](https://github.com/databricks/cli/pull/5))", True, "5")
    >>> pr_link_problem("* A change. ([#5](https://github.com/databricks/cli/pull/5), [#9](https://github.com/databricks/cli/pull/9))", True, "9")
    >>> pr_link_problem("* A change. ([#5](https://github.com/databricks/cli/pull/5))", True, "9")
    'trailing PR link #5 must include the PR that added this fragment (#9)'
    >>> pr_link_problem("* A change. ([#5](https://github.com/databricks/cli/pull/5))", False, None)
    """
    m = TRAILING_GROUP_RE.search(text.strip())
    if m is None:
        if not require_pr_link:
            return None
        pr = expected_pr or "<PR>"
        return f"missing trailing PR link: end with ([#{pr}](https://github.com/databricks/cli/pull/{pr}))"
    numbers = []
    for link in LINK_RE.findall(m.group("links")):
        lm = PR_LINK_RE.fullmatch(link)
        if lm is None:
            return f'malformed trailing PR link "{link}": expected [#N](https://github.com/databricks/cli/pull/N)'
        numbers.append(lm.group(1))
    if expected_pr is not None and expected_pr not in numbers:
        shown = ", ".join("#" + n for n in numbers)
        return f"trailing PR link {shown} must include the PR that added this fragment (#{expected_pr})"
    return None


def infer_expected_pr(path, fallback_pr, root):
    """Return the PR number that introduced the fragment at ``path``.

    databricks/cli squash-merges end the commit subject with ``(#N)``, so the
    commit that most recently added the file names its PR. A fragment not yet on
    main (added on the current branch, or uncommitted) has no such commit, so
    ``fallback_pr`` — the current PR — is used. ``git`` runs in ``root`` so the
    repo being validated is queried even when ``--root`` differs from the process
    CWD. Requires full git history (the workflow checks out with
    ``fetch-depth: 0``); best-effort, so any git failure falls back rather than
    erroring."""
    try:
        result = subprocess.run(
            ["git", "log", "-1", "--diff-filter=A", "--format=%s", "--", str(path)],
            capture_output=True,
            text=True,
            timeout=10,
            cwd=root,
        )
    except (OSError, subprocess.SubprocessError):
        return fallback_pr
    if result.returncode == 0:
        m = re.search(r"\(#(\d+)\)\s*$", result.stdout.strip())
        if m:
            return m.group(1)
    return fallback_pr


def load_sections(root):
    """Return the section slugs from .codegen.json, in changelog order.

    A missing or malformed .codegen.json raises: it is not something a PR author
    can be at fault for, so it crashes with the original traceback rather than
    being reported as a fragment problem."""
    codegen = json.loads((root / CODEGEN_FILE).read_text(encoding="utf-8"))

    sections = codegen.get(NEXTCHANGES_SECTIONS_KEY)
    if not isinstance(sections, dict) or not sections:
        raise ValueError(f"{CODEGEN_FILE} must define a non-empty {NEXTCHANGES_SECTIONS_KEY} object")

    return tuple(sections)


def find_problems(changelog_dir, sections, require_pr_link=False, fallback_pr=None, root=None):
    """Return a list of ``(path, message)`` for anything unexpected under
    ``.nextchanges/``: files that aren't a section fragment or known scaffolding,
    malformed fragments, a trailing PR link that is missing or names the wrong
    PR, and a missing/malformed version file. ``require_pr_link`` and
    ``fallback_pr`` drive the PR-link checks (set in CI / from the branch's PR,
    see ``main``); ``root`` is the repo the PR inference queries via git."""
    problems = []
    known_sections = set(sections)
    for path in sorted(changelog_dir.rglob("*")):
        if path.is_dir():
            continue
        rel = path.relative_to(changelog_dir)
        name = path.name

        # Root-level: only the version file and root documentation belong here. This prevents
        # someone accidentally putting a .md into .nextchanges thinking it would be picked up.
        if len(rel.parts) == 1:
            if name not in (VERSION_FILE, README, NEXTVERSION_GO):
                problems.append((path, "unexpected file at .nextchanges root"))
            continue

        # Section-level: .nextchanges/<section>/<file>.
        if len(rel.parts) == 2 and rel.parts[0] in known_sections:
            # README.md holds section docs and keeps the directory in git; the
            # renderer skips it, so it is not treated as a fragment.
            if name == README:
                continue
            if not name.endswith(".md"):
                problems.append((path, "unexpected file (fragments must be *.md)"))
            else:
                text = path.read_text(encoding="utf-8")
                problem = fragment_format_problem(text) or link_problem(text)
                if problem is None:
                    # Only infer the expected PR (a git call) once the fragment
                    # is structurally valid.
                    expected_pr = infer_expected_pr(path, fallback_pr, root)
                    problem = pr_link_problem(text, require_pr_link, expected_pr)
                if problem:
                    problems.append((path, problem))
            continue

        # Wrong depth or an unknown section directory.
        problems.append((path, "not in a known section directory"))

    version_path = changelog_dir / VERSION_FILE
    if not version_path.is_file():
        problems.append((version_path, "missing; expected the next release version (e.g. 1.4.0)"))
    elif not SEMVER_RE.match(version_path.read_text(encoding="utf-8").strip()):
        problems.append((version_path, "not a valid semver version (e.g. 1.4.0)"))
    return problems


def in_ci():
    """Whether we're running in GitHub Actions (where the link is required).

    GitHub Actions sets ``GITHUB_ACTIONS`` for every step; see
    https://docs.github.com/en/actions/learn-github-actions/variables#default-environment-variables"""
    return os.environ.get("GITHUB_ACTIONS") == "true"


def current_branch_pr(root):
    """Best-effort PR number for the current branch (via ``gh``), or ``None``.

    A local convenience only; the branch is ambiguous when its name maps to
    several PRs, so CI uses the authoritative event number instead (see
    ``detect_current_pr``). ``gh`` runs in ``root`` so it resolves the repo being
    validated. Any failure — ``gh`` missing, offline, unauthenticated, or no PR
    for the branch — returns ``None`` so local runs never hard-fail on tooling."""
    try:
        result = subprocess.run(
            ["gh", "pr", "view", "--json", "number", "-q", ".number"],
            capture_output=True,
            text=True,
            timeout=10,
            cwd=root,
        )
    except (OSError, subprocess.SubprocessError) as e:
        print(f"gh pr view failed: {e}", file=sys.stderr)
        return None
    out = result.stdout.strip()
    return out if result.returncode == 0 and out.isdigit() else None


def detect_current_pr(root):
    """PR number to expect for not-yet-merged fragments, or ``None``.

    In CI, use the authoritative event PR number (``PR_NUMBER``, set from
    ``github.event.number``) — never the branch, which is ambiguous when a name
    maps to several PRs. It is empty on a CI push (e.g. to main), where each
    fragment's PR is inferred from its own squash-merge commit instead. Locally,
    fall back to a best-effort ``gh`` lookup of the branch's PR."""
    pr = os.environ.get("PR_NUMBER", "").strip()
    if pr:
        return pr
    if in_ci():
        return None
    return current_branch_pr(root)


def has_fragments(changelog_dir):
    """Whether any *.md fragment (excluding README.md) exists under a section."""
    return any(p.name != README for p in changelog_dir.glob("*/*.md"))


def fragment_paths(changelog_dir):
    """The section fragments under ``changelog_dir`` (README.md excluded), sorted."""
    return sorted(p for p in changelog_dir.glob("*/*.md") if p.name != README)


def fixed_fragment(text, pr):
    r"""Return ``text`` with a trailing PR link for ``pr`` appended, or unchanged.

    Only a well-formed fragment (single line, ``* `` bullet, trailing period) with
    no trailing link group is changed; anything else is returned unchanged for the
    linter to report, so the fix never guesses at a malformed entry.

    >>> fixed_fragment("* Added a thing.\n", "42")
    '* Added a thing. ([#42](https://github.com/databricks/cli/pull/42))\n'
    >>> fixed_fragment("* Already linked. ([#7](https://github.com/databricks/cli/pull/7))\n", "42")
    '* Already linked. ([#7](https://github.com/databricks/cli/pull/7))\n'
    >>> fixed_fragment("No bullet.\n", "42")
    'No bullet.\n'
    """
    stripped = text.strip()
    if fragment_format_problem(text) is not None or TRAILING_GROUP_RE.search(stripped):
        return text
    return f"{stripped} ([#{pr}](https://github.com/databricks/cli/pull/{pr}))\n"


def autofix(changelog_dir, root):
    """Append the branch's PR link to fragments missing one (a lint autofix).

    The PR is inferred from the current branch (local only; see
    ``current_branch_pr``). Prints each file changed. Silently does nothing when
    the branch has no PR yet — the number can't be inferred, so there's nothing
    to add — since this runs by default on every validation."""
    pr = current_branch_pr(root)
    if pr is None:
        return
    for path in fragment_paths(changelog_dir):
        text = path.read_text(encoding="utf-8")
        fixed = fixed_fragment(text, pr)
        if fixed != text:
            path.write_text(fixed, encoding="utf-8")
            print(f"Fixed {path.relative_to(root)}: added ([#{pr}](.../pull/{pr}))")


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--root", type=pathlib.Path, default=pathlib.Path.cwd(), help="repository root")
    args = parser.parse_args(argv)

    changelog_dir = args.root / CHANGELOG_DIR
    if not changelog_dir.is_dir():
        return

    sections = load_sections(args.root)

    # Auto-fix by default: add the branch's PR link to fragments missing one
    # before validating. A no-op in CI and anywhere the branch has no PR.
    if has_fragments(changelog_dir):
        autofix(changelog_dir, args.root)

    # A trailing PR link is required whenever the change is associated with a PR:
    # always in CI (fail closed), and locally when the branch has an open PR.
    # ``fallback_pr`` is the PR to expect for not-yet-merged fragments — the
    # authoritative event number in CI, else the branch's PR locally. Detected
    # only when there are fragments, to avoid a `gh` call on unrelated runs.
    require_pr_link = False
    fallback_pr = None
    if has_fragments(changelog_dir):
        fallback_pr = detect_current_pr(args.root)
        require_pr_link = in_ci() or fallback_pr is not None

    problems = find_problems(changelog_dir, sections, require_pr_link, fallback_pr, args.root)
    if problems:
        for path, msg in problems:
            print(f"{path}: {msg}", file=sys.stderr)
        print(f"\nFragments must live at {CHANGELOG_DIR}/<section>/<name>.md", file=sys.stderr)
        print("and be a single line with a `* ` bullet marker and a trailing period, e.g.", file=sys.stderr)
        print("  * Added the `databricks quickstart` command.", file=sys.stderr)
        print(f"Valid sections: {', '.join(sections)}", file=sys.stderr)
        print(f"{CHANGELOG_DIR}/{VERSION_FILE} must hold the next release version.", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
