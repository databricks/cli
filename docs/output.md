# Principles for CLI output

There are four types of output:
1. Command output
2. Command error
3. Progress reporting
4. Logging

We try to adhere to the following rules:
* On success, the command's primary output is written to standard output.
  This is required to make commands composable in scripts.
* On error, the command's error message is written to standard error
  and the command exits with a non-zero exit code.
* Progress reporting may be provided to standard error,
  iff standard error is a TTY _and_ logging is disabled _or_ the logging
  output is different from standard error.
* Logging must be enabled explicitly.
  If enabled, it writes to standard error by default.
  Logging is **only** an aid to investigate issues and must not be relied
  on for command output, command errors, or progress reporting.
