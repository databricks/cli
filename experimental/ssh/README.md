## SSH Tunnel for Databricks

The SSH tunnel lets customers connect any IDE to Databricks compute to run and debug all code - including non-Spark/ML - with environment parity, and simple setup.

## Compute Requirements
- Serverless compute, which is the default when `--cluster` is omitted, or
- A cluster in Dedicated access mode assigned to a single user, not to a group.

`ValidateClusterAccess` (`internal/client/client.go`) rejects every other access mode up front,
for terminal SSH sessions as well as for IDE Remote Development: the tunnel runs as a job that
attaches as a single user. Standard access mode (`USER_ISOLATION`), Dedicated-to-a-group, and
no-isolation clusters all fail with `cluster '<id>' must be a dedicated single-user cluster`.

## Usage
A. With local ssh config setup:
```shell
databricks ssh setup --name=hello --cluster=id # one time only
ssh hello # use system SSH client to create a session
```
B. Spawn an ssh session directly:
```shell
databricks ssh connect --cluster=id
```

### Connection names and host keys

`--name` is a stable handle for serverless compute, not a live session id: connecting with a
name reuses the SSH server still running under it, and starts a new one on the same name once
the previous server has shut down (`--shutdown-delay` after its last client disconnects).

Both cases verify the server's host key. The server generates the key on first use, keeps it in
the connection's secret scope in the workspace and reuses it for every `sshd` it launches, so
the workspace is the authority on the key. The client reads it from there and pins it in
`~/.databricks/ssh-tunnel-known-hosts/<name>` before each connection, and points ssh at that
file with `StrictHostKeyChecking yes`. Tunnel host keys therefore never land in
`~/.ssh/known_hosts`, where a name - unique only within one workspace - would collide with an
entry left by other compute.

### Filesystem access

On dedicated compute, the tunnel server registers its own process with the filesystem
daemons. SSH sessions that remain descendants of that server can keep accessing
`/Workspace` and `/Volumes` if the bootstrap notebook exits and the server survives.
This does not keep the compute or server alive, preserve detached processes that leave
the server's process tree, or restore them after a server restart.

The server checks its workspace-file registration and current credential every 60 seconds.
It updates the workspace-file registration when the credential changes or the registered
process is lost, and retries failed registrations. Unchanged workspace-file registrations
do not generate updates. Volume registration is refreshed on every tick so it recovers
after an independent UC-FUSE restart.
The bootstrap supplies a fixed credential; checking it again cannot renew an expired or
revoked credential. Registration entries may remain until compute shutdown. Matching
the process start time prevents a reused PID from inheriting an old registration.

Registration failures are warnings. On compute where the daemon APIs are unavailable,
including serverless, the tunnel starts and file access continues to depend on the
bootstrap notebook as before. When the current-user lookup succeeds, registration also
includes the workspace home directory so workspace files can prepopulate its directory cache.
This does not change the session's working directory or filesystem permissions.
See [filesystem troubleshooting](./FAILURE_MODES.md#filesystem-access-after-the-bootstrap-notebook-exits).

## Development

```shell
./task build snapshot-release
./cli ssh connect --cluster=<id> --releases-dir=./dist --debug # or modify ssh config accordingly
```

To reproduce and test the known `ssh connect` failure modes (container missing `sshd`, or a
container that can't run the Python bootstrap), see [FAILURE_MODES.md](./FAILURE_MODES.md).

## Keeping detached processes alive

By default nothing outlives the session: when the last client disconnects, the server shuts
down after `--shutdown-delay` and the bootstrap notebook sweeps every process it parents,
including work that was deliberately detached with `tmux`, `setsid` or `nohup`.

`databricks ssh connect --cluster=<id> --keep-detached-for=<duration>` changes that. On
teardown the tunnel terminates only its own process group - the server and its `sshd`
children - and then holds the job run open for up to `<duration>` while any detached process
is still running. Two things to know before using it:

- **It holds the cluster up.** A `RUNNING` job run suppresses autotermination, so the
  cluster keeps accruing DBUs until the work finishes or the duration runs out. That is why
  the flag takes a duration rather than a boolean, and why it is off by default: the unit of
  the knob is the thing being spent. `--keep-detached-for` cannot exceed the job's own 24h
  timeout, and multi-day work still belongs in Jobs/DABs. Note also that reconnecting starts
  a new run rather than rejoining the lingering one, so each session with live detached work
  leaves its own run behind.
- **The notebook has to stay alive, not just the process.** Workspace filesystem access is
  authorized by walking the live process tree for a registered ancestor, and the bootstrap
  notebook is that ancestor. A detached process that outlives it keeps `/dbfs` and REST API
  access but loses `/Workspace` and `/Volumes` with `EPERM` - which is why the group-scoped
  teardown is tied to the linger and not enabled on its own.

Dedicated clusters only. On serverless the container is torn down with the run, so survivors
die regardless and the flag is rejected.

When the flag is *not* set and the server does find detached processes at teardown, it logs a
warning naming them, so work that is about to be swept is no longer lost silently.

## Design

High level:
```mermaid
---
config:
  theme: redux
  layout: dagre
---
flowchart TD
 n1(["Client"])
 subgraph s1["Control Plane"]
        n3["Jobs API"]
        n2["Driver Proxy API"]
        n11["Workspace API"]
        n13["Secrets API"]
  end
 subgraph s2["Compute - dedicated single user, or root"]
        n4["SSH Server"]
        n7["sshd, one process per connection"]
  end
    n1 -. "1 - store the client key pair" .-> n13
    n1 -. "2 - upload the CLI binary" .-> n11
    n1 -. "3 - start an ssh server job" .-> n3
    n3 -. "4 - start ssh server" .-> n4
    n4 -. "5 - read the client public key and store the host key" .-> n13
    n4 -. "6 - publish the ssh server port number" .-> n11
    n1 -. "7 - get the ssh server port number" .-> n11
    n1 <-. "8 - websocket connection" .-> n2
    n2 <-. "9 - websocket connection" .-> n4
    n4 <-. "10 - stdio" .-> n7
    style s2 stroke:#757575
    style s1 stroke:#757575
```

The client public key reaches the server through a secret scope rather than through the
bootstrap notebook, and the server host key is persisted in the same scope so its fingerprint
survives a restart - which is what makes `StrictHostKeyChecking accept-new` safe here.

Connection flow:
```mermaid
---
config:
  theme: base
---
sequenceDiagram
  autonumber
  participant P1 as databricks ssh connect
  participant P2 as ssh client
  participant P3 as databricks ssh connect --proxy
  participant P4 as workspace api
  participant P5 as secrets api
  participant P6 as databricks ssh server
  participant P7 as sshd
  Note over P1,P6: Try to get a port and a remote user name of an existing server.<br/>$v is the databricks CLI version. $s is the session id:<br/>the cluster id, or the --name value for serverless.
  activate P1
  P1 ->> P4: GET /Workspace/Users/$me/.databricks/ssh-tunnel/$v/$s/metadata.json
  P4 -->> P1: {port, cluster_id, usage_policy_id} or error
  P1 ->> P6: GET /driver-proxy-api/o/$workspaceId/$cluster/$port/metadata
  P6 -->> P1: the OS user the server runs as - root - or error
  Note over P1,P6: Start a new server if either step failed, or if the<br/>running one was started under a different usage policy.
  opt
    P1 ->> P5: create scope $me-$s-ssh-tunnel-keys,<br/>generate and store the client key pair unless already there
    P1 -->> P4: PUT ssh-tunnel/$v/$cliName/databricks, unless it's already there
    P1 ->> P4: PUT ssh-tunnel/$v/$s/ssh-server-bootstrap
    P1 ->> P6: jobs/runs/submit ssh-server-bootstrap $cluster
    activate P6
    P6 ->> P6: pick a free port, starting at 7772
    P6 ->> P4: PUT ssh-tunnel/$v/$s/metadata.json<br/>{port, cluster_id, usage_policy_id}
    P6 ->> P5: read the client public key,<br/>generate and store the server host key unless already there
    P6 ->> P6: write authorized_keys and a custom sshd config<br/>start self-kill-timeout<br/>listen for /ssh, /metadata and /logs
    loop until successful or timed out - 30 attempts, 2s apart
      P1 -> P6: Get port and remote user name of the server (sequence 1 - 4 above)
    end
  end
  Note over P1,P7: We know the port and the user, spawn "ssh"
  P1 ->> P2: ssh -l $user -i $key -o ServerAliveInterval=30<br/> -o ProxyCommand="databricks ssh connect --proxy<br/> --cluster=$cluster --metadata=$user,$port,$cluster"
  activate P2
  P2 ->> P3: exec ProxyCommand
  activate P3
  P3 ->> P6: wss:/driver-proxy-api/o/$workspaceId/$cluster/$port/ssh?id=$connId
  P6 ->> P6: stop self-kill-timeout
  P6 ->> P7: /usr/sbin/sshd -f config -i
  activate P7
  P2 -> P7: pubkey auth
  loop until the connection is closed<br/>by ssh client, sshd, or driver-proxy
    P2 -> P7: stdin and stdout
    P1 -> P2: stdin, stdout, and stderr
    deactivate P7
    deactivate P3
    deactivate P2
    deactivate P1
  end
  break when the last ws connection drops
    P6 ->> P6: start self-kill-timeout, then exit when it fires
    deactivate P6
  end
```

Note that `metadata.json` is published before the server starts accepting connections, and is
left behind when the server exits. Neither its presence nor its contents prove that a server is
running, which is why the client always re-checks `/metadata` through the driver proxy before
reusing a port.

### Session resume protocol

Resume protocol v2 lets an SSH session survive a dropped websocket without losing or duplicating
bytes. The protocol is symmetric: the client and server keep separate offsets and replay buffers
for their outgoing streams.

Before opening the websocket, the client requests `/capabilities`. It enables resume only when the
server returns `{"resume_version":2}`. A missing route, another version, an invalid response, or a
probe that takes more than ten seconds leaves resume disabled. The SSH connection still proceeds.
When resume is enabled, the initial websocket URL includes these query parameters:

| Parameter | Initial value | Meaning |
| --- | --- | --- |
| `id` | A new session UUID | Identifies the SSH session across websocket connections. |
| `resume_version` | `2` | Selects this protocol version. |
| `delivered` | `0` | Reports how many peer payload bytes this side has written to its local stream. |

Binary websocket frames carry SSH payload. Text frames contain `{"delivered":N}` and acknowledge
that the receiver wrote the first `N` bytes to sshd's stdin on the server or stdout on the client.
Each side appends outgoing payload to a one MiB replay buffer before writing it to the websocket.
Acknowledgments release buffer space. The receiver sends them after 64 KiB or 100 milliseconds.
When the buffer is full, the sender stops reading its source until an acknowledgment frees space.

After an unexpected disconnect, the client retries for 60 seconds. The server retains the session
for 90 seconds so it remains available throughout that retry period.

```mermaid
sequenceDiagram
  participant C as Client proxy
  participant S as Server proxy
  C->>S: GET /ssh?id=ID&resume_version=2&delivered=C&reattach=1
  S->>S: Close the old socket and stop delivery at offset S
  S-->>C: Websocket upgrade
  S->>C: Text frame {"delivered":S}
  par Replay server output after C
    S->>C: Binary frames [C, server sent offset)
  and Replay client input after S
    C->>S: Binary frames [S, client sent offset)
  end
  C->>S: Continue payload and acknowledgments
  S->>C: Continue payload and acknowledgments
```

The `delivered` value on the reattach URL tells the server where to replay its output. The first
text frame on the new websocket gives the client the corresponding offset for its input. Each side
replays the buffered range from the peer's offset before sending new payload. The server returns
HTTP 409 when the existing session did not negotiate resume and HTTP 410 when the session no longer
exists. Those responses stop retries; other connection failures retry within the 60-second budget.

A normal websocket close with reason `finished` ends the session instead of starting a reattach.
This explicit reason distinguishes a clean EOF from a dropped connection. Scheduled authentication
handovers reuse the same session ID without `reattach=1`; if a handover itself drops, protocol v2
uses the same reattach exchange to recover it.
