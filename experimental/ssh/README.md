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

## Development
```shell
./task build snapshot-release
./cli ssh connect --cluster=<id> --releases-dir=./dist --debug # or modify ssh config accordingly
```

To reproduce and test the known `ssh connect` failure modes (container missing `sshd`, or a
container that can't run the Python bootstrap), see [FAILURE_MODES.md](./FAILURE_MODES.md).

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
