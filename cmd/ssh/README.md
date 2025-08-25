## SSH Tunnel for Databricks
SSH tunnel lets customers connect any IDE to Databricks clusters to run and debug all code - including non-Spark/ML- with environment parity, and simple setup.

## Cluster Requirements
- Dedicated (single user) access mode if you want to use Remote Development tools in IDEs
- Dedicated or Standard access mode for terminal ssh connections

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
make build snapshot-release
./cli ssh connect --cluster=<id> --releases-dir=./dist --debug # or modify ssh config accordingly
```

## Design

High level:
```mermaid
---
config:
  theme: redux
  layout: dagre
---
flowchart TD
 n1(["Client A"])
 subgraph s1["Control Plane"]
        n3["Jobs API"]
        n2["Driver Proxy API"]
        n11["Workspace API"]
  end
 subgraph s3["Spark User A (or root)"]
        n4["SSH Server A"]
  end
 subgraph s4["Spark User B"]
        n6["SSH Server B"]
  end
 subgraph s2["Cluster"]
        s3
        s4
        n12["WSFS/DBFS/Volumes"]
  end
    n1 -. "1 - start an ssh server job" .-> n3
    n3 -. "2 - start ssh server" .-> n4
    n4 <-. "3 - save the port number" .-> n12
    n1 <-. "4 - get ssh server port" .-> n11
    n1 <-. "6 - websocket connection" .-> n2
    n2 <-. "7 - websocket connection" .-> n4
    n6 <-.-> n12
    style s2 stroke:#757575
    style s1 stroke:#757575
```

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
  participant P4 as wsfs
  participant P6 as databricks ssh server
  participant P7 as sshd
  Note over P1,P6: Try to get a port and a remote user name of an existing server<br/> ($v is databricks CLI version, $cluster is supplied by the user)
  activate P1
  P1 ->> P4: GET ~/.ssh/$v/$cluster/metadata.json
  P4 -->> P1: {port: xxxx} or error
  P1 ->> P6: GET /dirver-proxy-api/$cluster/$port/metadata
  P6 -->> P1: {user: spark-xxxx} or {user: root} or error
  Note over P1,P6: Start the new server in the case of an error
  opt
    P1 -->> P1: generate<br/>key pair
    P1 -->> P4: PUT ~/.ssh/$v/bin/databricks, unless it's already there
    P1 ->> P4: PUT ~/.ssh/$v/$cluster/start-server-with-pub-key.ipynb
    P1 ->> P6: jobs/runs/submit start-server-with-pub-key.ipynb $cluster
    activate P6
    P6 ->> P6: start self-kill-timeout<br/>generate server key pair<br/>create custom sshd config<br/>listen for /ssh and /metadata on a free port
    P6 ->> P4: PUT ~/.ssh/$v/$cluster/metadata.json<br/>{port: xxxx}
    loop unil successful or timed out
      P1 -> P6: Get port and remote user name of the server (sequence 1 - 4 above)
    end
  end
  Note over P1,P7: We know the port and the user, spawn "ssh"
  P1 ->> P2: ssh -l $user -i $key<br/> -o ProxyCommand="databricks ssh connect --proxy $cluster $user $port"
  activate P2
  P2 ->> P3: exec ProxyCommand
  activate P3
  P3 ->> P6: wss:/dirver-proxy-api/$cluster/$port/ssh
  P6 ->> P6: stop self-kill-timeout
  P6 ->> P7: /usr/sbin/sshd -i -f config
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
    P6 ->> P6: start self-kill-timeout
    P6 ->> P4: DELETE  ~/.ssh/$v/$cluster/metadata.json
    deactivate P6
  end
```
