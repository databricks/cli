#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# dependencies = ["PyYAML>=6.0"]
# ///
"""
Sandboxed runner for Taskfile-defined commands.

Reads the task name from $TASK_NAME (set by Taskfile's top-level
`env: TASK_NAME: '{{.TASK}}'`) and behaves based on what the task declares:

  - no `sources:` and no `generates:` → passthrough (exec the command as-is)
  - `sources:` only                   → sandbox the command against those
                                        sources; any touched files are copied
                                        back without enforcement
  - `sources:` and `generates:`       → sandbox + enforce that touched files
                                        are a subset of generates

Hooked into Taskfile.yml via the SANDBOX_CMD task var, prepended to every
cmd as {{.SANDBOX_CMD}}. Default is empty (no-op). Two ways to enable:

    SANDBOX=1 ./task build               # uses this script (resolved via ROOT_DIR)
    SANDBOX=$PWD/my_wrapper.py ./task X  # uses a custom wrapper

Tasks with `dir:` work too, but SANDBOX must be an absolute path.
"""

import os
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

import yaml


def find_repo_root() -> Path:
    p = Path.cwd().resolve()
    while True:
        if (p / "Taskfile.yml").exists():
            return p
        if p.parent == p:
            sys.exit("sandboxed_run: Taskfile.yml not found walking up from cwd")
        p = p.parent


def glob_to_regex(pattern: str) -> re.Pattern:
    out = ""
    i = 0
    while i < len(pattern):
        if pattern[i : i + 3] == "**/":
            out += r"(?:.*/)?"
            i += 3
        elif pattern[i : i + 2] == "**":
            out += r".*"
            i += 2
        elif pattern[i] == "*":
            out += r"[^/]*"
            i += 1
        elif pattern[i] == "?":
            out += r"[^/]"
            i += 1
        else:
            out += re.escape(pattern[i])
            i += 1
    return re.compile(out + "$")


def normalize_patterns(patterns):
    # Return (op, pattern) pairs in declaration order. Later patterns can
    # re-include files that earlier patterns excluded, matching Task's own
    # source-list semantics.
    out = []
    for p in patterns or []:
        if isinstance(p, dict) and "exclude" in p:
            out.append(("exclude", p["exclude"]))
        elif isinstance(p, str):
            out.append(("include", p))
    return out


def expand_braces(pattern: str) -> list[str]:
    # Shell-style {a,b,c} expansion, recursive. No escaping, no ranges.
    m = re.search(r"\{([^{}]*)\}", pattern)
    if not m:
        return [pattern]
    prefix, suffix = pattern[: m.start()], pattern[m.end() :]
    out = []
    for opt in m.group(1).split(","):
        out.extend(expand_braces(prefix + opt + suffix))
    return out


def resolve_taskvars(pat: str, vars_dict: dict, root: Path) -> str:
    # Substitute Taskfile Go-template refs like {{.VAR}}. Static vars use their
    # string value; dynamic `sh:` vars are executed from the repo root.
    # Resolves a single {{.VAR}} inside {{.OTHER}} bodies too (one round).
    builtins = {"ROOT_DIR": str(root), "TASKFILE_DIR": str(root)}

    def lookup(name):
        if name in builtins:
            return builtins[name]
        v = vars_dict.get(name)
        if v is None:
            return None
        if isinstance(v, dict) and "sh" in v:
            sh_cmd = re.sub(
                r"\{\{\s*\.(\w+)\s*\}\}",
                lambda m: builtins.get(m.group(1), vars_dict.get(m.group(1), m.group(0))),
                v["sh"],
            )
            return subprocess.check_output(sh_cmd, shell=True, cwd=root, encoding="utf-8").strip()
        return str(v)

    def sub(m):
        out = lookup(m.group(1))
        return m.group(0) if out is None else out

    return re.sub(r"\{\{\s*\.(\w+)\s*\}\}", sub, pat)


def list_git_visible(root: Path) -> set[str]:
    # Tracked files + untracked-but-not-gitignored files, all repo-root-relative.
    # Used to skip gitignored paths (caches, venvs, build outputs) during source
    # expansion so tasks don't need to enumerate them as `exclude:` in Taskfile.yml.
    out = subprocess.check_output(
        ["git", "ls-files", "--cached", "--others", "--exclude-standard"],
        cwd=root,
        text=True,
    )
    return set(out.splitlines())


def filter_gitignored(paths: list[str], root: Path) -> list[str]:
    # Drop paths that git would ignore; used when copying touched files back
    # from the sandbox so transient outputs (e.g. uv-created .venv/) stay put.
    if not paths:
        return paths
    res = subprocess.run(
        ["git", "check-ignore", "--stdin"],
        input="\n".join(paths),
        cwd=root,
        capture_output=True,
        text=True,
    )
    ignored = set(res.stdout.splitlines())
    return [p for p in paths if p not in ignored]


def match_pattern(pat: str, task_dir: Path, root: Path, visible: set[str] | None):
    # Expand a single include pattern into a set of task_dir-relative paths.
    # Handles brace expansion, absolute paths (from {{.ROOT_DIR}}), and ../
    # parent climbs. Filters gitignored matches when the pattern is a glob.
    out = set()
    for sub_pat in expand_braces(pat):
        if sub_pat.startswith("/"):
            abs_path = Path(sub_pat)
            try:
                sub_pat = str(abs_path.relative_to(task_dir))
            except ValueError:
                sub_pat = os.path.relpath(abs_path, task_dir)
        base = task_dir
        rel_pat = sub_pat
        up = 0
        while rel_pat.startswith("../"):
            base = base.parent
            rel_pat = rel_pat[3:]
            up += 1
        # Skip the gitignore filter when the pattern names a specific file:
        # tasks sometimes depend on gitignored build outputs (e.g. the
        # downloaded .codegen/openapi.json) and should honor explicit names.
        is_glob = any(c in rel_pat for c in "*?[")
        for p in base.glob(rel_pat):
            if not p.is_file():
                continue
            rel = "../" * up + str(p.relative_to(base))
            if is_glob and visible is not None:
                try:
                    root_rel = str(p.resolve().relative_to(root))
                except ValueError:
                    root_rel = None
                if root_rel is not None and root_rel not in visible:
                    continue
            out.add(rel)
    return out


def expand_sources(
    patterns,
    task_dir: Path,
    root: Path,
    vars_dict: dict | None = None,
    visible: set[str] | None = None,
):
    # Process include/exclude ops in declaration order: a later `include` can
    # re-add files that an earlier `exclude` dropped. Matches Task's source
    # list semantics so the sandbox copies the same files Task tracks.
    ops = normalize_patterns(patterns)
    if vars_dict:
        ops = [(op, resolve_taskvars(p, vars_dict, task_dir)) for op, p in ops]
    files: set[str] = set()
    for op, pat in ops:
        if op == "include":
            files |= match_pattern(pat, task_dir, root, visible)
        else:
            rx = glob_to_regex(pat)
            files = {f for f in files if not rx.fullmatch(f)}
    return sorted(files)


def snapshot_mtimes(root: Path):
    return {p: p.stat().st_mtime_ns for p in root.rglob("*") if p.is_file()}


def main():
    argv = sys.argv[1:]
    if not argv:
        sys.exit("usage: sandboxed_run.py CMD [ARG ...]")

    name = os.environ.get("TASK_NAME")
    if not name:
        sys.exit("sandboxed_run: $TASK_NAME is unset; add `env: TASK_NAME: '{{.TASK}}'` at Taskfile top level")

    root = find_repo_root()
    data = yaml.safe_load((root / "Taskfile.yml").read_text())
    task = (data.get("tasks") or {}).get(name)
    if not isinstance(task, dict):
        sys.exit(f"sandboxed_run: task {name!r} not found in Taskfile.yml")
    vars_dict = data.get("vars") or {}

    source_pats = task.get("sources")
    generates_pats = task.get("generates")

    # Passthrough: task declares neither inputs nor outputs.
    if not source_pats and not generates_pats:
        sys.exit(subprocess.call(argv))

    if not source_pats:
        sys.exit(f"sandboxed_run: task {name!r} has `generates:` but no `sources:`")

    # `sources:` / `generates:` are relative to the task's `dir:`, which itself
    # is relative to Taskfile.yml (repo root). Paths in the sandbox mirror the
    # real tree so the command sees the same layout.
    task_dir = root / (task.get("dir") or ".")
    visible = list_git_visible(root)
    sources = expand_sources(source_pats, task_dir, root, vars_dict, visible)
    if not sources:
        sys.exit(f"sandboxed_run: task {name!r} sources matched no files")
    # `generates:` patterns are task_dir-relative; rebase them to repo-root
    # so they match touched paths (which are rel to tmp, mirroring the repo).
    task_rel = str(task_dir.relative_to(root))
    prefix = "" if task_rel == "." else task_rel + "/"
    generates_res = [glob_to_regex(prefix + p) for p in generates_pats or []]
    enforce = bool(generates_pats)

    # Create tmp manually so we can manage cleanup around `git worktree remove`,
    # which deletes the directory itself.
    tmp = Path(tempfile.mkdtemp(prefix=f"taskbox-{name}-"))
    # Register a throwaway git worktree at tmp so tests that look for
    # `.git` or rely on git commands (git status, config, ignore rules
    # resolution) find a real git repo. --no-checkout avoids duplicating
    # the tree; we layer the declared sources on top ourselves.
    worktree_ok = (
        subprocess.call(
            ["git", "worktree", "add", "--no-checkout", "--detach", str(tmp), "HEAD"],
            cwd=root,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        == 0
    )
    try:
        sandbox_dir = tmp / task_dir.relative_to(root)
        sandbox_dir.mkdir(parents=True, exist_ok=True)
        for rel in sources:
            # normpath collapses any ../ segments in rel (e.g. when task_dir
            # is `tools` and rel is `../Taskfile.yml`, dst lands at tmp/Taskfile.yml).
            dst = Path(os.path.normpath(sandbox_dir / rel))
            dst.parent.mkdir(parents=True, exist_ok=True)
            # Preserve symlinks so tests that rely on them (e.g. libs/fileset,
            # cmd/labs/project) observe the same topology as the real tree.
            shutil.copy2(task_dir / rel, dst, follow_symlinks=False)
        for gen in generates_pats or []:
            # Pre-create parent dirs for generates targets so tasks can write
            # outputs even when their dir has no source files.
            if any(c in gen for c in "*?["):
                continue
            (sandbox_dir / gen).parent.mkdir(parents=True, exist_ok=True)

        before = snapshot_mtimes(tmp)
        sandbox_cwd = tmp / Path.cwd().resolve().relative_to(root)
        sandbox_cwd.mkdir(parents=True, exist_ok=True)
        mode = "enforced" if enforce else "sources-only"
        print(
            f"[sandbox {name}] mode={mode} sources={len(sources)} cwd={sandbox_cwd}",
            file=sys.stderr,
        )
        rc = subprocess.call(argv, cwd=sandbox_cwd)

        after = snapshot_mtimes(tmp)
        touched = sorted(str(p.relative_to(tmp)) for p, mt in after.items() if before.get(p) != mt)
        # Filter out git worktree metadata (`.git` file, index, logs, etc.).
        touched = [t for t in touched if t != ".git" and not t.startswith(".git/")]
        # Drop gitignored touched files (caches, .venv, build outputs) so tasks
        # don't need to enumerate them as excludes. Declared `generates:` outputs
        # are preserved even if gitignored (e.g. python/docs/_output).
        declared = [t for t in touched if any(r.fullmatch(t) for r in generates_res)]
        touched = declared + filter_gitignored([t for t in touched if t not in set(declared)], root)

        if enforce:
            unexpected = [t for t in touched if not any(r.fullmatch(t) for r in generates_res)]
            if unexpected:
                print(
                    f"[sandbox {name}] ERROR: touched files not declared in `generates:`:",
                    file=sys.stderr,
                )
                for u in unexpected:
                    print(f"  {u}", file=sys.stderr)
                sys.exit(rc or 2)

        for t in touched:
            src = tmp / t
            dst = root / t
            dst.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(src, dst)
        # `touched` is reported relative to tmp, so paths are already
        # repo-root-relative (the sandbox tree mirrors the real tree).
    finally:
        if worktree_ok:
            subprocess.call(
                ["git", "worktree", "remove", "--force", str(tmp)],
                cwd=root,
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
            )
        shutil.rmtree(tmp, ignore_errors=True)

    sys.exit(rc)


if __name__ == "__main__":
    main()
