#!/usr/bin/env python3
"""Exercise file-backed bundle commands while one DMS API persistently fails."""

import json
import os
import re
import shlex
import shutil
import subprocess
import sys
from pathlib import Path
from string import Template

sys.path.insert(0, str(Path(shutil.which("print_requests.py")).parent))
from add_repl import add_repl
from print_requests import read_json_many

API = sys.argv[1]
FIXTURES = Path(__file__).parent
CLI = os.environ["CLI"]
REQUESTS = Path(os.environ["OUT_REQUESTS"])
LOG = Path("LOG.commands").resolve()
PATTERNS = {
    "complete_version": "POST /api/2.0/bundle/deployments/{id}/versions/{version}/complete",
    "create_deployment": "POST /api/2.0/bundle/deployments",
    "create_version": "POST /api/2.0/bundle/deployments/{id}/versions",
    "delete_deployment": "DELETE /api/2.0/bundle/deployments/{id}",
    "get_deployment": "GET /api/2.0/bundle/deployments/{id}",
    "get_operation": "GET /api/2.0/bundle/deployments/{id}/versions/{version}/operations/*",
    "get_resource": "GET /api/2.0/bundle/deployments/{id}/resources/*",
    "get_version": "GET /api/2.0/bundle/deployments/{id}/versions/{version}",
    "heartbeat": "POST /api/2.0/bundle/deployments/{id}/versions/{version}/heartbeat",
    "list_deployments": "GET /api/2.0/bundle/deployments",
    "list_operations": "GET /api/2.0/bundle/deployments/{id}/versions/{version}/operations",
    "list_resources": "GET /api/2.0/bundle/deployments/{id}/resources",
    "list_versions": "GET /api/2.0/bundle/deployments/{id}/versions",
    "update_deployment": "PATCH /api/2.0/bundle/deployments/{id}",
    "update_operation": "PATCH /api/2.0/bundle/deployments/{id}/versions/*",
}
ACTIVE = None
REGISTERED = set()
KNOWN_IDS = set()


def run(cwd, *args, expected=0):
    visible = args[0] == "bundle"
    if visible:
        refresh_faults(cwd)
    result = subprocess.run(
        [CLI, *args],
        cwd=cwd,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT if visible else subprocess.PIPE,
    )
    output = result.stdout + (result.stderr or "")
    with LOG.open("a") as out:
        out.write(f"\n{cwd.name}: {args}\nexit={result.returncode}\n{output}")
    if visible:
        print("\n>>> " + shlex.join(args), flush=True)
        for line in output.splitlines():
            print("  " + line if line else "", flush=True)
    assert result.returncode == expected, output
    return result


def fault(pattern, enabled):
    subprocess.run(
        [sys.executable, shutil.which("fault.py"), pattern, "500", "0", "1000" if enabled else "0"], check=True
    )


def refresh_faults(cwd):
    if ACTIVE is None:
        return
    if "{version}" in ACTIVE:
        # Cover existing versions and the next version a command can create.
        serial = state(cwd).get("serial", 0)
        patterns = [ACTIVE.format(version=version) for version in range(1, serial + 2)]
    else:
        patterns = [ACTIVE]
    for pattern in patterns:
        if pattern not in REGISTERED:
            fault(pattern, True)
            REGISTERED.add(pattern)


def clear_faults():
    global ACTIVE
    ACTIVE = None
    for pattern in REGISTERED:
        fault(pattern, False)
    REGISTERED.clear()


def hits(pattern):
    if not REQUESTS.exists():
        return 0
    regex = re.escape(pattern).replace(r"\{version\}", r"[^/]+").replace(r"\*", ".*") + "$"
    return sum(bool(re.fullmatch(regex, f"{r['method']} {r['path']}")) for r in read_json_many(REQUESTS.read_text()))


def state(cwd):
    return json.loads((cwd / ".databricks/bundle/default/resources.json").read_text())


def ids(cwd):
    data = state(cwd)
    assert "deployment_history_shadow" in data["features"], data
    assert "deployment_history" not in data["features"], data
    current = {key: value["__id__"] for key, value in data["state"].items()}
    for key, value in current.items():
        if value not in KNOWN_IDS:
            add_repl(value, key.rsplit(".", 1)[1].upper() + "_ID")
            KNOWN_IDS.add(value)
    return current


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
    global ACTIVE
    print(f"\n=== {cwd.name}: setup", flush=True)
    deployment = None
    if API != "create_deployment":
        run(cwd, "bundle", "deploy", "--auto-approve")
        node = json.loads(
            run(cwd, "workspace", "get-status", root + "/state/resources.deployment.json", "-o", "json").stdout
        )
        deployment = str(node["object_id"])
    pattern = PATTERNS[API].format(id=deployment, version="{version}")
    ACTIVE = pattern
    refresh_faults(cwd)
    print(f"\n=== {cwd.name}: {API} fault enabled", flush=True)
    return pattern


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
    run(cwd, "bundle", "deploy", "--auto-approve")
    current = ids(cwd)
    assert_alive(cwd, current)
    summary = run(cwd, "bundle", "summary").stdout
    assert current["resources.jobs.foo"] in summary
    assert current["resources.pipelines.bar"] in summary
    for kind, key in [("job", "foo"), ("pipeline", "bar")]:
        group = "jobs" if kind == "job" else "pipelines"
        resource = f"resources.{group}.{key}"
        generate(cwd, kind, key, current[resource])
        assert (cwd / "exported" / f"{key}.{kind}.yml").is_file()
        if kind == "pipeline":
            assert (cwd / "exported-src/pipeline.sql").read_text() == (cwd / "pipeline.sql").read_text()
        run(cwd, "bundle", "deployment", "unbind", key)
        assert resource not in state(cwd)["state"]
        generate(cwd, kind, key, current[resource], bind=True)
        assert ids(cwd)[resource] == current[resource]
        if kind == "pipeline":
            assert (cwd / "src/pipeline.sql").read_text() == (cwd / "pipeline.sql").read_text()
    run(cwd, "bundle", "validate")
    run(cwd, "bundle", "deploy", "--auto-approve")
    assert ids(cwd) == current
    pipeline = json.loads(run(cwd, "pipelines", "get", current["resources.pipelines.bar"], "-o", "json").stdout)
    assert pipeline["spec"]["libraries"][0]["file"]["path"] == root + "/files/src/pipeline.sql"
    run(cwd, "bundle", "destroy", "--auto-approve")
    assert_destroyed(cwd, root, current)
    print("\nDMS API called: " + ("yes" if hits(pattern) > previous else "no"), flush=True)
finally:
    clear_faults()
