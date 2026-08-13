Released binaries are now built against the FIPS 140-3 validated Go Cryptographic
Module (v1.0.0, CMVP certificate #5247), with FIPS 140-3 mode enabled by default.
TLS connections negotiate only FIPS-approved cipher suites, which drops ChaCha20
and CBC suites from what the client offers. Start the CLI with
`GODEBUG=fips140=off` to restore the previous behaviour.
