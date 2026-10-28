#!/usr/bin/env python3
"""Exercise file-backed bundle commands while one DMS API persistently fails."""

import json
import os
import shutil
import subprocess
import sys
from pathlib import Path
from string import Template

sys.path.insert(0, str(Path(shutil.which("print_requests.py")).parent))
from print_requests import read_json_many

API = sys.argv[1]
FIXTURES = Path(__file__).parent
CLI = os.environ["CLI"]
REQUESTS = Path(os.environ["OUT_REQUESTS"])
LOG = Path("LOG.commands").resolve()
PATTERNS = {
    "create_deployment": "POST /api/2.0/bundle/deployments",
    "get_deployment": "GET /api/2.0/bundle/deployments/{id}",
    "update_deployment": "PATCH /api/2.0/bundle/deployments/{id}",
    "delete_deployment": "DELETE /api/2.0/bundle/deployments/{id}",
    "create_version": "POST /api/2.0/bundle/deployments/{id}/versions",
    "update_operation": "PATCH /api/2.0/bundle/deployments/{id}/versions/*",
    "complete_version": "POST /api/2.0/bundle/deployments/{id}/versions/*",
}


def run(cwd, *args, expected=0):
    result = subprocess.run([CLI, *args], cwd=cwd, text=True, capture_output=True)
    with LOG.open("a") as out:
        out.write(f"\n{cwd.name}: {args}\nexit={result.returncode}\n{result.stdout}{result.stderr}")
    assert result.returncode == expected, result.stdout + result.stderr
    return result


def fault(pattern, enabled):
    subprocess.run(
        [sys.executable, shutil.which("fault.py"), pattern, "500", "0", "1000" if enabled else "0"], check=True
    )


def hits(pattern):
    if not REQUESTS.exists():
        return 0
    requests = read_json_many(REQUESTS.read_text())
    return sum(
        (
            f"{r['method']} {r['path']}".startswith(pattern[:-1])
            if pattern.endswith("*")
            else f"{r['method']} {r['path']}" == pattern
        )
        for r in requests
    )


def state(cwd):
    return json.loads((cwd / ".databricks/bundle/default/resources.json").read_text())


def ids(cwd):
    data = state(cwd)
    assert "deployment_history_shadow" in data["features"], data
    assert "deployment_history" not in data["features"], data
    return {key: value["__id__"] for key, value in data["state"].items()}


def setup(phase):
    cwd = Path("work", phase).resolve()
    (cwd / "resources").mkdir(parents=True)
    name = f"dms-fault-{API}-{phase}-{os.environ['UNIQUE_NAME']}"
    root = f"/Workspace/Users/{os.environ['CURRENT_USER_NAME']}/.bundle/{name}"
    values = {"RESOURCE_NAME": name, "ROOT_PATH": root, "DATASET_NAME": name.replace("-", "_")}
    for source, target in [
        ("databricks.yml.tmpl", "databricks.yml"),
        ("job.yml.tmpl", "resources/foo.job.yml"),
        ("pipeline.yml.tmpl", "resources/bar.pipeline.yml"),
        ("pipeline.sql.tmpl", "pipeline.sql"),
    ]:
        (cwd / target).write_text(Template((FIXTURES / source).read_text()).substitute(values))
    return cwd, root


def activate(cwd, root):
    deployment = None
    if API != "create_deployment":
        run(cwd, "bundle", "deploy", "--auto-approve")
        node = json.loads(
            run(cwd, "workspace", "get-status", root + "/state/resources.deployment.json", "-o", "json").stdout
        )
        deployment = str(node["object_id"])
    pattern = PATTERNS[API].format(id=deployment)
    fault(pattern, True)
    return pattern


def assert_fault(result, pattern, before):
    assert hits(pattern) > before, f"{API} was never called"
    assert "Fault injected by test." in result.stdout + result.stderr, result.stdout + result.stderr


def assert_alive(cwd, current):
    job = json.loads(run(cwd, "jobs", "get", current["resources.jobs.foo"], "-o", "json").stdout)
    pipeline = json.loads(run(cwd, "pipelines", "get", current["resources.pipelines.bar"], "-o", "json").stdout)
    assert str(job["job_id"]) == current["resources.jobs.foo"]
    assert pipeline["pipeline_id"] == current["resources.pipelines.bar"]
    assert job["settings"]["name"].startswith("updated-dms-fault-")
    assert pipeline["spec"]["name"].startswith("updated-dms-fault-")


def generate(cwd, kind, key, identifier, bind=False):
    args = ["bundle", "generate", kind, f"--existing-{kind}-id", identifier, "--key", key]
    args += [
        "--config-dir",
        "resources" if bind else "exported",
        "--source-dir",
        "src" if bind else "exported-src",
        "--force",
    ]
    if bind:
        args.append("--bind")
    run(cwd, *args)


def assert_destroyed(cwd, root, current):
    job = run(cwd, "jobs", "get", current["resources.jobs.foo"], expected=1)
    assert "does not exist" in job.stderr.lower(), job.stderr
    pipeline = run(cwd, "pipelines", "get", current["resources.pipelines.bar"], expected=1)
    assert "not found" in pipeline.stderr.lower(), pipeline.stderr
    file = run(cwd, "workspace", "get-status", root + "/state/resources.json", expected=1)
    assert "doesn't exist" in file.stderr.lower() or "does not exist" in file.stderr.lower(), file.stderr


cwd, root = setup("commands")
pattern = activate(cwd, root)
try:
    # Pinning the root lets a bundle-name change exercise UpdateDeployment in place.
    config = cwd / "databricks.yml"
    config.write_text(config.read_text().replace("name: dms-fault-", "name: updated-dms-fault-"))
    for path in (cwd / "resources").glob("*.yml"):
        path.write_text(path.read_text().replace("name: dms-fault-", "name: updated-dms-fault-"))
    previous = hits(pattern)
    result = run(cwd, "bundle", "deploy", "--auto-approve")
    current = ids(cwd)
    if API != "delete_deployment":
        assert_fault(result, pattern, previous)
    print("deploy: ok", flush=True)
    run(cwd, "bundle", "validate")
    print("validate: ok", flush=True)
    summary = json.loads(run(cwd, "bundle", "summary", "-o", "json").stdout)
    assert str(summary["resources"]["jobs"]["foo"]["id"]) == current["resources.jobs.foo"]
    assert summary["resources"]["pipelines"]["bar"]["id"] == current["resources.pipelines.bar"]
    print("summary: ok", flush=True)
    for kind, key in [("job", "foo"), ("pipeline", "bar")]:
        group = "jobs" if kind == "job" else "pipelines"
        resource = f"resources.{group}.{key}"
        generate(cwd, kind, key, current[resource])
        assert (cwd / "exported" / f"{key}.{kind}.yml").is_file()
        if kind == "pipeline":
            assert (cwd / "exported-src/pipeline.sql").read_text() == (cwd / "pipeline.sql").read_text()
        run(cwd, "bundle", "deployment", "unbind", key)
        assert resource not in state(cwd)["state"]
        assert_alive(cwd, current)
        generate(cwd, kind, key, current[resource], bind=True)
        assert ids(cwd)[resource] == current[resource]
        if kind == "pipeline":
            assert (cwd / "src/pipeline.sql").read_text() == (cwd / "pipeline.sql").read_text()
        print(f"generate {kind}, unbind, generate --bind: ok", flush=True)
    run(cwd, "bundle", "validate")
    run(cwd, "bundle", "deploy", "--auto-approve")
    assert ids(cwd) == current
    pipeline = json.loads(run(cwd, "pipelines", "get", current["resources.pipelines.bar"], "-o", "json").stdout)
    assert pipeline["spec"]["libraries"][0]["file"]["path"] == root + "/files/src/pipeline.sql"
    print("redeploy preserves IDs: ok", flush=True)
    run(cwd, "bundle", "destroy", "--auto-approve")
    assert_destroyed(cwd, root, current)
    print("destroy: ok", flush=True)
finally:
    fault(pattern, False)

if API != "update_deployment":
    cwd, root = setup("destroy")
    pattern = activate(cwd, root)
    try:
        if API == "create_deployment":
            run(cwd, "bundle", "deploy", "--auto-approve")
        current = ids(cwd)
        previous = hits(pattern)
        result = run(cwd, "bundle", "destroy", "--auto-approve")
        assert_fault(result, pattern, previous)
        assert_destroyed(cwd, root, current)
        print("isolated destroy fault exercised: ok", flush=True)
    finally:
        fault(pattern, False)
