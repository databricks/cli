# Device code login (experimental)

Device code login authenticates the Databricks CLI on machines that cannot open
a browser, such as headless servers, SSH sessions, and web terminals. You
approve the login in a browser on another device (for example your laptop or
phone), and type the confirmation code it shows you back into the CLI.

> [!WARNING]
> Device code login is experimental. It is disabled by default, may change
> without notice, and only supports workspace hosts.
> Account hosts and unified (SPOG) hosts are not supported yet.

## Prerequisites

- The workspace must have device code login enabled. If it is not, the CLI
  fails with `device code login is not enabled for this workspace`.
- Enable the feature in the CLI by setting `DATABRICKS_CLI_DEVICE_CODE=true`.

## Log in from a headless machine

```console
$ export DATABRICKS_CLI_DEVICE_CODE=true
$ databricks auth login --host https://<workspace-host> --profile my-profile --use-device-code
Using a browser on another device, visit:
https://<workspace-host>/device/verification

And enter the code:
WDJB-MJHT

Once finished, enter the confirmation code provided in your browser.
Confirmation code: FTFA-QQKY
Profile my-profile was successfully saved
```

1. Run `databricks auth login` with `--use-device-code`. The host can be passed as
   an argument, with `--host`, or come from an existing profile. Otherwise the
   CLI prompts for it. The profile picker leaves out profiles that device code
   login cannot use: profiles for an account host and profiles saved with
   `--resource`. Profiles for a unified host are still listed and are rejected
   after you pick one.
2. On another device, open the URL that the CLI shows, sign in to the
   workspace if needed, and enter the code. The code expires after about 10
   minutes.
3. Confirm that you started this login yourself. Only approve a login that you
   initiated: approving it grants the CLI access to the Databricks API as you.
4. The browser shows a confirmation code. Type it into the CLI. Codes are not
   case-sensitive, and the dash is optional. If the code is rejected, the CLI
   asks for it again and shows how many attempts are left, up to 3 attempts in
   total. Only give the confirmation code to the CLI that started the login,
   or to the AI agent that runs it (see
   [Log in without a terminal](#log-in-without-a-terminal)). Never share it
   with anyone else.

The CLI saves the profile and the token the same way as the browser-based
`databricks auth login`, so other commands use it via `--profile my-profile`.

## Log in without a terminal

When stdin is not a terminal, nobody can type the confirmation code into the
running CLI. This is the case when a script or an AI agent such as Claude Code
runs the login as a one-off command. The login then runs non-interactively, in
two steps:

```console
$ databricks auth login --host https://<workspace-host> --profile my-profile --use-device-code < /dev/null
Using a browser on another device, visit:
https://<workspace-host>/device/verification

And enter the code:
WDJB-MJHT

Once finished, complete the login by running the same command with
--user-code WDJB-MJHT and --confirmation-code set to the confirmation code
provided in your browser.
$ databricks auth login --host https://<workspace-host> --profile my-profile --use-device-code --user-code WDJB-MJHT --confirmation-code FTFA-QQKY
Profile my-profile was successfully saved
```

1. The first command prints the URL and the code, and keeps the pending login
   in the CLI's token storage. It does not change the profile, so commands
   that use it keep working until the login completes.
2. Approve the login in the browser as described above.
3. Run the same command with the hidden `--user-code` flag set to the code
   from step 1 and the hidden `--confirmation-code` flag set to the
   confirmation code. It saves the profile and the token. Without
   `--profile`, it only stores the token for the host.

When the CLI detects an AI agent, it also prints a note after the code, with
or without a terminal. The note tells the agent to pass the URL and the code
on to you, not to approve the login itself, and to ask you for the
confirmation code:

```console
Note for AI agents (claude-code): give the URL and the code above to the user.
Do not open the URL or approve the login yourself: the user must approve it.
Ask the user for the confirmation code their browser shows.
```

In a terminal, the note also tells the agent to enter the confirmation code
at the `Confirmation code` prompt of the running command.

The pending login stays on your machine, so a confirmation code that ends up
somewhere else, such as an agent's conversation, cannot complete the login on
another machine.

If the confirmation code is rejected, check it and run the command again. If
the login expired, start a new one.

Each profile, or host without a profile, has one pending login. Starting a
second login for it replaces the first, so complete one login before starting
another.

A successful login for the profile, in any way, and `databricks auth logout`
remove its pending login.

`--configure-cluster` is not supported by the non-interactive login. Set
`cluster_id` on the profile after the login.

## Session lifetime

The token is renewed automatically for up to 24 hours. After that, commands
fail with a token refresh error and you need to run the login again.

## Troubleshooting

| Error or symptom | Likely cause | What to do |
| --- | --- | --- |
| `--use-device-code is experimental. Set DATABRICKS_CLI_DEVICE_CODE=true to enable it` | The feature is not enabled in the CLI. | Set `DATABRICKS_CLI_DEVICE_CODE=true`. |
| `device code login is not enabled for this workspace` | The workspace does not advertise device code login. | Use a workspace where the feature is enabled. |
| `--use-device-code is only supported for workspace hosts` | The host is an account or unified host. | Log in to a workspace host instead. |
| The browser says the code is invalid or expired | The code was mistyped, or more than ~10 minutes passed. | Re-enter the code carefully, or run the login again for a new code. Codes never contain `0`, `1`, `2`, `O`, `Z`, or `I`. |
| `Invalid confirmation code` | The confirmation code was mistyped or left blank. | Re-enter the code. After 3 rejected codes, the login fails with `Invalid confirmation code (error code: invalid_request). Start login from scratch` and you need to run it again. |
| `token exchange: Invalid device code` | The login expired. A confirmation code entered after the login expired is first rejected as `Invalid confirmation code`, and the workspace then discards the login. | Run the login again to get a new code. |
| `token exchange: Device grant rate limit exceeded` | Too many confirmation codes were sent for this login. | Run the login again to get a new code. |
| `unauthorized_client` or `invalid_client` | The OAuth client is not allowed to use device code login in this workspace. | Use the default `databricks-cli` client, or ask your admin to enable the custom client. |
| `--configure-cluster is not supported by the non-interactive device code login` | The cluster picker needs a terminal. | Log in first, then set `cluster_id` on the profile. |
| `token exchange: Invalid confirmation code (error code: invalid_request). Check the confirmation code ...` | The confirmation code was mistyped, or the login was not approved yet. | Check the code and approve the login, then run the command again. If it still fails, start a new login. |
| `token exchange: Invalid device code (error code: invalid_request). Run the same command without --user-code and --confirmation-code ...` | The login expired. | Start a new login. |
| `a newer login for profile ... replaced this one` | A second login was started for the same profile or host. | Approve the newest code, and complete the newest login with its user code. |
| `the login for profile ... was started with a different host` (or client ID, or scopes) | The second command does not use the host, `--client-id` or `--scopes` of the first one. | Complete the login with the same command that started it, plus `--user-code` and `--confirmation-code`. |
| `no pending device code login for profile ...` | The login was already completed, or no login was started for this profile or host. | Start a new login. |
| `--confirmation-code requires --user-code` | `--user-code` is missing from the command. | Add `--user-code` with the code from the first step. |
| `--confirmation-code requires --use-device-code` | `--use-device-code` is missing from the command. | Run the same command as the first step, with `--user-code` and `--confirmation-code`. |
| `reading confirmation code: EOF` | The input was closed, e.g. with Ctrl+D, before a code was entered. | Run the login again. |
| `reading confirmation code: context deadline exceeded` | No confirmation code was entered before the login timed out (1 hour by default). | Run the login again. Use `--timeout` to change the limit. |
