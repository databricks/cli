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
   CLI prompts for it.
2. On another device, open the URL that the CLI shows, sign in to the
   workspace if needed, and enter the code. The code expires after about 10
   minutes.
3. Confirm that you started this login yourself. Only approve a login that you
   initiated: approving it grants the CLI access to the Databricks API as you.
4. The browser shows a confirmation code. Type it into the CLI. Codes are not
   case-sensitive, and the dash is optional. If the code is rejected, the CLI
   asks for it again and shows how many attempts are left, up to 3 attempts in
   total. Never share the confirmation code with anyone.

The CLI saves the profile and the token the same way as the browser-based
`databricks auth login`, so other commands use it via `--profile my-profile`.

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
| `reading confirmation code: EOF` | stdin was closed before a code was entered. | Keep stdin open until you write the confirmation code. |
| `reading confirmation code: context deadline exceeded` | No confirmation code was entered before the login timed out (1 hour by default). | Run the login again. Use `--timeout` to change the limit. |
