#!/usr/bin/env python3
"""Refresh the vendored mlops-stacks acceptance template to upstream HEAD.

The local test leg inits from the vendored copy; the cloud leg clones upstream.
Only the files the template renders for this test's config are vendored, and that
set is recomputed on every bump by rendering upstream, so an upstream rename is
picked up without a hand-maintained file list.
"""

import json
import os
import shutil
import subprocess
import tempfile
from pathlib import Path
from string import Template

REPO_URL = "https://github.com/databricks/mlops-stacks"

TEST_DIR = Path("acceptance/bundle/deploy/mlops-stacks")
TEMPLATE_DIR = TEST_DIR / "template"
REVISION_FILE = TEST_DIR / "template.REVISION"

# Upstream's project dir name is an illegal Go module path, so we vendor it renamed.
# The project name is lowercase-alphanumeric, so the transform is a no-op and the
# rendered output is unchanged.
UPSTREAM_PROJECT_DIR = "{{template `project_name_alphanumeric_underscore` .}}"
VENDORED_PROJECT_DIR = "{{.input_project_name}}"

# The test's config.json.tmpl is envsubst'd by the script; substitute the values the
# local leg gets. CLOUD_ENV is unset locally, which acceptance_test.go maps to "aws".
RENDER_ENV = {"UNIQUE_NAME": "x", "CLOUD_ENV_BASE": "aws"}

# Everything under template/ but outside the generated project is template machinery
# (run_validations, update_layout). It renders no output of its own, so it cannot be
# discovered from the rendered tree and is always kept.
PROJECT_ROOT_SEG = "{{.input_root_dir}}"


def render_path(rel_path, config):
    """Map a path under the upstream template/ dir to its path in the rendered project."""
    subs = {
        PROJECT_ROOT_SEG: config["input_root_dir"],
        VENDORED_PROJECT_DIR: config["input_project_name"],
        UPSTREAM_PROJECT_DIR: config["input_project_name"],
    }
    rendered = str(rel_path)
    for old, new in subs.items():
        rendered = rendered.replace(old, new)
    return rendered.removesuffix(".tmpl")


def render(cli, clone_dir, config_file, out_dir):
    empty_cfg = out_dir.parent / "empty.databrickscfg"
    empty_cfg.touch()
    # bundle init authenticates before rendering, so drop the caller's Databricks
    # environment and point it at a non-resolving host with an empty config file.
    env = {k: v for k, v in os.environ.items() if not k.startswith("DATABRICKS_")}
    env |= {
        "DATABRICKS_HOST": "https://bump-mlops-stacks.test",
        "DATABRICKS_TOKEN": "dummy",
        "DATABRICKS_CONFIG_FILE": str(empty_cfg),
    }
    subprocess.run(
        [str(cli), "bundle", "init", str(clone_dir), "--config-file", str(config_file)],
        check=True,
        cwd=out_dir,
        stdout=subprocess.DEVNULL,
        env=env,
    )


def keep_set(clone_dir, out_dir, config):
    """List the upstream files needed to render this test's project, relative to the repo root."""
    keep = [Path("databricks_template_schema.json")]
    keep += sorted(p.relative_to(clone_dir) for p in (clone_dir / "library").rglob("*") if p.is_file())
    for path in sorted((clone_dir / "template").rglob("*")):
        if not path.is_file():
            continue
        rel = path.relative_to(clone_dir)
        in_project = rel.parts[1:2] == (PROJECT_ROOT_SEG,)
        if not in_project or (out_dir / render_path(rel.relative_to("template"), config)).exists():
            keep.append(rel)
    return keep


def vendored_path(rel_path):
    return Path(*[VENDORED_PROJECT_DIR if p == UPSTREAM_PROJECT_DIR else p for p in rel_path.parts])


def main():
    root = Path(__file__).resolve().parent.parent
    config = Template((root / TEST_DIR / "config.json.tmpl").read_text()).substitute(RENDER_ENV)

    with tempfile.TemporaryDirectory(prefix="mlops-stacks-") as tmp:
        tmp = Path(tmp)
        clone_dir, out_dir, cli = tmp / "upstream", tmp / "rendered", tmp / "cli"
        out_dir.mkdir()
        subprocess.run(["go", "build", "-o", str(cli), "."], check=True, cwd=root)
        config_file = tmp / "config.json"
        config_file.write_text(config)

        subprocess.run(["git", "clone", "--depth", "1", "--quiet", REPO_URL, str(clone_dir)], check=True)
        sha = subprocess.run(
            ["git", "-C", str(clone_dir), "rev-parse", "HEAD"],
            check=True,
            capture_output=True,
            text=True,
        ).stdout.strip()

        render(cli, clone_dir, config_file, out_dir)
        keep = keep_set(clone_dir, out_dir, json.loads(config))

        template_dir = root / TEMPLATE_DIR
        shutil.rmtree(template_dir)
        for rel in keep:
            dst = template_dir / vendored_path(rel)
            dst.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(clone_dir / rel, dst)

    (root / REVISION_FILE).write_text(sha + "\n")

    print(f"Vendored {len(keep)} files from {REPO_URL} @ {sha}")


if __name__ == "__main__":
    main()
