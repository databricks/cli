# testcli

This package provides a way to run the CLI from tests as if it were a separate process.
By running the CLI inline we can still set breakpoints and step through execution.

It transitively imports pretty much this entire repository, which is why we
intentionally keep this package _separate_ from `testutil`.
