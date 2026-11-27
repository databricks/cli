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
accept public-key authentication. The CLI caches a compatible Linux executable in the sandbox and invokes its server
automatically; no relay needs to be included in the image.
For development builds on macOS or Windows, provide `--releases-dir` containing
Linux release archives built from the same checkout.

The private client key remains on the local machine. Public-key entries expire
after 24 hours and are renewed during connection setup. The sandbox's persistent
HOME stores its server key and public-key entries, allowing reuse after restart.
Each proxy invocation starts and cleans up its own remote server command. The server waits up to two minutes for its first connection and exits when its
SSH session ends. The CLI renews the WebSocket and command-output stream every
25 minutes before their route lifetime limits. If a forwarding connection drops
unexpectedly, the CLI attempts to resume the same SSH session for up to 60 seconds.
If recovery fails or the remote session is gone, SSH ends and reconnecting starts
a new session. Setup prunes session
directories older than a day while preserving directories leased by active servers.
The CLI allows up to ten minutes to prepare the remote server, including sandbox
startup and executable upload. The SSH handshake allows fifteen minutes overall,
including workspace authentication. Neither timeout limits an established session.

These commands require workspace support for sandbox HTTP/WebSocket port
forwarding. They do not connect to the legacy Gateway SSH endpoint or register
keys with its registry.

## Migrating from gateway key registration

The legacy `databricks sandbox register`, `databricks sandbox ssh-key list`,
and `databricks sandbox ssh-key delete` commands have been removed. SSH now uses
workspace OAuth authentication and sandbox port forwarding; separate registration
with the gateway key registry is no longer needed.

Authenticate with `databricks auth login --host https://<workspace-host>`, then use
`databricks sandbox ssh SANDBOX_ID`. For editor Remote-SSH, `ssh`, `scp`, or `sftp`,
run `databricks sandbox ssh-setup SANDBOX_ID --name my-sandbox --profile PROFILE`
and use the new alias instead of the old gateway hostname.

Removing these commands does not delete existing local keys, change existing
`~/.ssh/config` entries or the legacy `~/.ssh/databricks-sandbox` include file,
or revoke any server-side key registrations. Existing gateway aliases remain
as they are; use the new OAuth-backed alias for supported sandbox connections.

### Revoking a previously registered gateway key

While the legacy gateway key registry remains available, you can revoke an old
registration through the generic API command. Use the workspace profile from
the original registration, list the keys, then delete only the selected key:

```sh
databricks api get /api/2.0/lakebox/ssh-keys --profile PROFILE
databricks api delete /api/2.0/lakebox/ssh-keys/KEY_HASH --profile PROFILE
```

Replace `KEY_HASH` with the `keyHash` from the selected entry in `sshKeys`.
Deletion revokes that key's legacy gateway access; it does not delete its local
key files or affect the workspace-authenticated port-forwarding SSH connection.
This cleanup is optional and is never performed automatically by this retirement.
