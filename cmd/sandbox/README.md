# SSH through sandbox port forwarding

Run `databricks sandbox ssh SANDBOX_ID` for an interactive shell, or pass a
remote command after `--`. This creates a reusable local SSH key automatically
and connects through workspace-authenticated WebSocket port forwarding.
It does not write a persistent SSH Host entry.

For native SSH tools, configure a local alias once:

```sh
databricks sandbox ssh-setup SANDBOX_ID --name my-sandbox --profile my-workspace
ssh my-sandbox
scp ./example.txt my-sandbox:/tmp/example.txt
sftp my-sandbox
```

Keep the Databricks CLI installed at the path recorded in the generated
`ProxyCommand`, with the saved Databricks profile and its credentials available.
Setup requires a named profile; environment-only credentials do not create a
persistent workspace selection. The proxy prepares the
sandbox and connects through port forwarding for each SSH connection. Run setup
again after relocating the executable or changing the selected workspace.

The local machine needs OpenSSH. The sandbox needs Python 3, `ssh-keygen`, and
`openssh-server`, with the unprivileged `sandbox-agent` account configured to
accept public-key authentication. The CLI uploads and invokes its server
automatically; no relay needs to be included in the image.
For development builds on macOS or Windows, provide `--releases-dir` containing
Linux release archives built from the same checkout.

The private client key remains on the local machine. Public-key entries expire
after 24 hours and are renewed during connection setup. The sandbox's persistent
HOME stores its server key and public-key entries, allowing reuse after restart.
Each proxy invocation starts and cleans up its own remote server command. A
server abandoned by its client exits after its idle timeout. A broken forwarding
connection ends the SSH session; reconnect to start another one.
The CLI allows up to ten minutes to prepare the remote server, including sandbox
startup and executable upload. The SSH handshake allows fifteen minutes overall,
including workspace authentication. Neither timeout limits an established session.

These commands require workspace support for sandbox HTTP/WebSocket port
forwarding. They do not connect to the legacy Gateway SSH endpoint or register
keys with its registry.
