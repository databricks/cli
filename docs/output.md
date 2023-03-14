# Priciples for CLI output

There are three types of output:
1. Command output
1. Command errors
1. Progress reporting
1. Logging

We try to adhere to the following rules:
* All commands may only write their primary output to standard out.
  This is required to make commands composable in scripts.
* Progress reporting may be provided to standard error,
  iff standard error is a TTY _and_ logging is disabled _or_ the logging
  output is different from standard error.
* Logging must be enabled explicitly. It outputs to standard error by
  default but can be configured to write to any file.
  Logging is **only** an aid to investigate issues and must not be relied
  on for command output, command errors, or progress reporting.
