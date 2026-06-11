# Reproducing `databricks ssh connect` failure modes

This guide documents container/cluster misconfigurations that make `databricks ssh connect`
fail, how to reproduce each one, the symptom the user sees, and where the real error lives. It
is primarily a testing aid for the SSH feature's error-handling paths.

For the connection flow and architecture, see [README.md](./README.md).

## Background: where failures surface

The bootstrap is a **Python notebook job** that starts `databricks ssh server` on the cluster.
The server publishes its port to the workspace (`metadata.json`), the client reads it, prints
`Connected!`, and spawns `ssh`. The SSH daemon (`/usr/sbin/sshd`) is launched **lazily, per
client connection** (see `internal/server/sshd.go` and `internal/proxy/server.go`). Because of
this ordering, different misconfigurations fail at different stages:

| Stage | Needs | Failure mode if missing |
| --- | --- | --- |
| Bootstrap job runs | a working Databricks **Python** runtime in the image | [Mode 2](#mode-2-container-cant-run-the-python-bootstrap) |
| Per-connection SSH | **`/usr/sbin/sshd`** (OpenSSH server) in the image | [Mode 1](#mode-1-container-missing-the-openssh-server-sshd) |

## Prerequisites

- A workspace with **Databricks Container Services** (custom Docker images) enabled.
- Permission to create a **dedicated (single-user)** cluster.
- A dev build of the CLI. See the *Development* section of [README.md](./README.md):
  ```shell
  ./task build snapshot-release
  ./cli ssh connect --cluster=<cluster-id> --releases-dir=./dist --debug
  ```
- A container registry the workspace can pull from (e.g. a public Docker Hub repo) to host the
  test images below. Build them for the cluster's architecture (`linux/amd64` on most clouds):
  ```shell
  docker buildx build --platform linux/amd64 -t <namespace>/<image>:<tag> --push .
  ```

The cluster specs below use a single-node dedicated cluster. Adjust `node_type_id` and
`spark_version` for your cloud and DBR version:

```json
{
  "cluster_name": "ssh-failure-repro",
  "spark_version": "16.4.x-scala2.12",
  "node_type_id": "<your-cloud-node-type>",
  "num_workers": 0,
  "data_security_mode": "SINGLE_USER",
  "single_user_name": "<you@example.com>",
  "spark_conf": { "spark.databricks.cluster.profile": "singleNode", "spark.master": "local[*, 4]" },
  "custom_tags": { "ResourceClass": "SingleNode" },
  "autotermination_minutes": 60,
  "docker_image": { "url": "<namespace>/<image>:<tag>" }
}
```

Create it with `databricks clusters create --json @cluster.json --no-wait` and wait for the
`RUNNING` state (a custom-container pull can take several minutes).

## Mode 1: container missing the OpenSSH server (`sshd`)

A notebook-capable image that does **not** ship `openssh-server`. Build it by removing the SSH
server from an image that otherwise works:

```dockerfile
FROM databricksruntime/standard:16.4-LTS
RUN (apt-get remove -y openssh-server || true) \
    && rm -f /usr/sbin/sshd /usr/bin/sshd
```

Create a cluster on this image, then:

```shell
./cli ssh connect --cluster=<cluster-id> --releases-dir=./dist
```

**Symptom.** The bootstrap job succeeds and publishes metadata, so the client prints
`Connected!` — and then the connection drops. The server can't launch `/usr/sbin/sshd` for the
incoming connection and holds the websocket open, so historically the `ssh` client **hung**
until its `ConnectTimeout`. The real error,
`failed to start SSHD process: ... /usr/sbin/sshd: no such file or directory`, is only written
to the bootstrap job's **stdout logs** while the job is still `RUNNING` — it is never a failed
job state.

**With the error-handling improvements** the client aborts after a handshake timeout (no SSH
banner from the server) with an actionable hint to install `openssh-server`, and exits
promptly instead of hanging.

**Fix.** Install `openssh-server` in the image (`apt-get install -y openssh-server`).

## Mode 2: container can't run the Python bootstrap

A bare/minimal base that lacks a working Databricks Python runtime. The simplest example is
`databricksruntime/rbase:16.4-LTS` used directly as the cluster image (it is an R *base* layer;
notably it has no functioning `/databricks/python` notebook-execution environment).

Create a cluster on `databricksruntime/rbase:16.4-LTS`, then:

```shell
./cli ssh connect --cluster=<cluster-id> --releases-dir=./dist
```

**Symptom.** The bootstrap is a Python notebook, but the image can't execute notebook commands,
so the job fails with `Could not reach driver of cluster <id>`. The SSH server never starts and
never publishes metadata, so the client fails with
`server metadata error / ... metadata.json doesn't exist` — **before** the `sshd` step is ever
reached. (A trivial `print(...)` notebook job submitted to the same cluster fails the same way,
which is a quick way to confirm the image, not the SSH feature, is at fault.)

**With the error-handling improvements** the client fetches the failed run's state message,
notebook error/trace, and run-page URL and shows them instead of the generic metadata error.

**Fix.** Build on a notebook-capable base (e.g. `databricksruntime/standard:...`) or otherwise
provide a working Databricks Python environment, in addition to `openssh-server`.

## Working control

`databricksruntime/standard:16.4-LTS` ships **both** a working Python runtime **and** `sshd`,
so `ssh connect` to a cluster on it succeeds end to end. Use it as a baseline to confirm your
workspace, cluster spec, and dev build are healthy before reproducing a failure mode.

## Inspecting the bootstrap job logs

`ssh connect` prints `Job submitted successfully with run ID: <id>`. Inspect it with:

```shell
databricks jobs get-run <id>              # open run_page_url in the UI
databricks jobs get-run-output <task-run-id>   # task-run-id = .tasks[0].run_id of the run
```

Caveat: for a **running** server task, `get-run-output`'s `logs`/`error` are not populated —
the `sshd` error from [Mode 1](#mode-1-container-missing-the-openssh-server-sshd) lives in the
live notebook cell stdout / driver logs, not the Jobs run-output API. A failed run from
[Mode 2](#mode-2-container-cant-run-the-python-bootstrap) does populate the run's state message
and error.

## Reproducing locally, without a workspace

The proxy-layer behaviors have unit tests that don't need a cluster:

- `internal/proxy/client_server_test.go`
  - `TestClientExitsWhenServerCommandFails` — server can't launch its command and closes the
    connection; the client exits promptly.
  - `TestClientTimesOutWhenServerSendsNothing` — server holds the connection open and sends
    nothing (the Mode 1 shape); the client aborts on the handshake timeout.
- `internal/client/client_internal_test.go` — formatting of a failed bootstrap run's error
  (state message, error trace, run-page URL) using SDK mocks.

```shell
go test ./experimental/ssh/...
```
