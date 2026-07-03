# /// script
# dependencies = ["PyGithub>=2,<3", "pyjwt<2.12.0", "charset-normalizer<3.4.6"]
# ///
"""databricks/cli release entrypoint.

Thin wrapper around the synced ``tagging.py`` — which is regenerated verbatim
from universe (``openapi/tagging/tagging.py``) by ``genkit update-sdk`` and must
stay pristine. The tagging workflow runs this file (``release_tagging.py``)
instead, so the CLI-specific behavior lives here rather than as edits to the
synced file:

* the changelog body is rendered from per-PR ``.nextchanges/<section>/*.md``
  fragments rather than read from a ``NEXT_CHANGELOG.md`` file, and
* the release version is read from ``.nextchanges/version`` (bumped to the next
  minor on each release) rather than a hand-maintained ``## Release vX.Y.Z``
  header.

It injects that behavior by rebinding two module-level seams in the upstream
module (``get_next_tag_info`` and ``clean_next_changelog``, both called by name
from ``process_package``/``preview_tag_infos``) and then delegates to the
untouched ``process()`` for all commit/tag/race/recovery logic.
"""

import os
import re
from typing import Optional

import tagging

NEXTCHANGES_DIR = ".nextchanges"

# Tracks the version of the next release. Read at release time and bumped to the
# next minor afterward — the role NEXT_CHANGELOG.md's "## Release vX.Y.Z" header
# played upstream.
VERSION_FILE = "version"

# Section subdirectory -> "### " header, in changelog order. Mirrors the
# section folders documented in .nextchanges/README.md and validated by
# tools/validate_nextchanges.py — keep the three in sync.
NEXTCHANGES_SECTIONS = (
    ("notable-changes", "Notable Changes"),
    ("cli", "CLI"),
    ("bundles", "Bundles"),
    ("dependency-updates", "Dependency updates"),
    ("api-changes", "API Changes"),
)


def _expand_pr_links(text: str) -> str:
    """
    Convert raw PR references to canonical markdown links, matching
    ``tools/update_github_links.py``: ``(#1234)`` and bare ``#1234`` become
    ``([#1234](https://github.com/<repo>/pull/1234))``. Existing links are
    left alone.
    """
    repo = os.environ.get("GITHUB_REPOSITORY", "databricks/cli")

    def link(n: str) -> str:
        return f"([#{n}](https://github.com/{repo}/pull/{n}))"

    text = re.sub(r"\(#(\d+)\)", lambda m: link(m.group(1)), text)
    # Bare #1234 not already inside a link ('[') or paren-wrapped ('(').
    return re.sub(r"(?<![\[(])#(\d+)\b", lambda m: link(m.group(1)), text)


def render_nextchanges(package_path: str) -> Optional[str]:
    """
    Render ``<package_path>/.nextchanges/<section>/*.md`` into the changelog
    body: one ``### <Section>`` block per non-empty section in
    NEXTCHANGES_SECTIONS order, fragments sorted by filename. Returns None when
    there are no fragments.

    The output matches the CHANGELOG.md convention: a blank line after each
    ``### <Section>`` heading, and every entry as a `` * `` bullet (a leading
    space before the ``*``). A leading ``* ``/``- `` marker is optional in a
    fragment and is normalized; continuation lines are left as authored.
    """
    base = os.path.join(os.getcwd(), package_path, NEXTCHANGES_DIR)
    if not os.path.isdir(base):
        return None

    blocks = []
    for slug, header in NEXTCHANGES_SECTIONS:
        section_dir = os.path.join(base, slug)
        if not os.path.isdir(section_dir):
            continue
        entries = []
        for name in sorted(os.listdir(section_dir)):
            if not name.endswith(".md") or name == "README.md":
                continue
            with open(os.path.join(section_dir, name)) as f:
                text = f.read().strip()
            if not text:
                continue
            first, _, rest = text.partition("\n")
            if first.startswith(("* ", "- ")):
                first = first[2:]
            # Leading-space bullet to match CHANGELOG.md (e.g. " * entry").
            entries.append(f" * {first}" + (("\n" + rest) if rest else ""))
        if entries:
            # Blank line after the heading, matching CHANGELOG.md.
            blocks.append(f"### {header}\n\n" + "\n".join(entries))

    if not blocks:
        return None
    return _expand_pr_links("\n\n".join(blocks))


def _version_path(package_path: str) -> str:
    return os.path.join(os.getcwd(), package_path, NEXTCHANGES_DIR, VERSION_FILE)


def next_version(package: tagging.Package) -> str:
    """
    Release version for this run, read from ``.nextchanges/version``. To cut a
    patch or major release, edit that file in the PR; otherwise the default
    (bumped after the previous release) is the next minor.
    """
    with open(_version_path(package.path)) as f:
        return str(tagging.Version.parse(f.read().strip().lstrip("v")))


def get_next_tag_info(package: tagging.Package) -> Optional[tagging.TagInfo]:
    """
    Replacement for ``tagging.get_next_tag_info``: build the release TagInfo
    from ``.nextchanges/`` fragments. Returns None when there are no entries
    (nothing to release), unless ``allow_empty_changelog`` is set in
    ``.codegen.json`` — matching the pristine skip behavior.
    """
    body = render_nextchanges(package.path)
    if body is None and not tagging._load_codegen_config().get("allow_empty_changelog", False):
        print("No .nextchanges/ entries. No changes will be made to the changelog.")
        return None

    version = next_version(package)
    # write_changelog() keys off the "## Release v…" header, so include it.
    content = f"## Release v{version}\n" + (f"\n{body}\n" if body else "")
    return tagging.TagInfo(package=package, version=version, content=content)


def clear_nextchanges(package_path: str) -> None:
    """
    Replacement for ``tagging.clean_next_changelog``: stage deletion of the
    ``.nextchanges/`` fragments consumed by this release and bump
    ``.nextchanges/version`` to the next minor (its post-release default; teams
    can still override it in a PR). Section directories and their README.md are
    left in place. ``process_package`` calls this as
    ``clean_next_changelog(package.path)``, so the signature matches.
    """
    base = os.path.join(os.getcwd(), package_path, NEXTCHANGES_DIR)
    for slug, _ in NEXTCHANGES_SECTIONS:
        section_dir = os.path.join(base, slug)
        if not os.path.isdir(section_dir):
            continue
        for name in sorted(os.listdir(section_dir)):
            if name.endswith(".md") and name != "README.md":
                tagging.gh.delete_file(os.path.join(section_dir, name))

    version_path = _version_path(package_path)
    with open(version_path) as f:
        released = tagging.Version.parse(f.read().strip().lstrip("v"))
    tagging.gh.add_file(version_path, f"{released.next_release_version()}\n")


def _delete_file(self, loc: str):
    """``git rm`` equivalent for GitHubRepo: stage a tree deletion (sha=None)."""
    local_path = os.path.relpath(loc, os.getcwd())
    print(f"Deleting file {local_path}")
    self.changed_files.append(tagging.InputGitTreeElement(path=local_path, mode="100644", type="blob", sha=None))


def install_nextchanges() -> None:
    """Rebind the tagging seams to the CLI's .nextchanges behavior."""
    tagging.GitHubRepo.delete_file = _delete_file
    tagging.get_next_tag_info = get_next_tag_info
    tagging.clean_next_changelog = clear_nextchanges


if __name__ == "__main__":
    install_nextchanges()
    tagging.validate_git_root()
    tagging.init_github()
    tagging.process()
