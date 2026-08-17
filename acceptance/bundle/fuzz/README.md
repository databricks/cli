Harness over ../invariant: mutates curated configs and runs a real target script.
`run_fuzz.py` owns the seed loop (`seed.sh` per seed) and classifies deployed /
rejected / gap / hang / bug. `FUZZ_TARGET` in test.toml picks the target.

Each seed mutates a JSON snapshot in `bases/` of a deploy-verified config from
`../invariant/configs/` and may inject a curated optional from INJECT. Regen with
`./task generate-fuzz-bases`; `validate-generated` fails on drift.

Helpers come from ../invariant/script.prepare (sourced explicitly; prepare/test.toml
only merge along the directory chain). Server stubs are copied into test.toml;
script asserts stub parity. Unmodeled routes return `TESTSERVER_GAP` (gaps).

A failure is a CLI bug. `LOG.repro` prints e.g.
`ENVFILTER=FUZZ_TARGET=no_drift FUZZ_SEED_START=<seed> FUZZ_SEED_COUNT=1 FUZZ_CHECK_DRIFT=0 ./task test-fuzz`
(`ENVFILTER` because `FUZZ_TARGET` is a matrix key).

`FUZZ_CHECK_DRIFT=0` (committed run) uses plan-determinism; `1` (`task test-fuzz` /
nightly) uses exact no_drift. Only the committed run is expected green; a red
nightly is a finding to triage (gates `test-result`; failure summary has the repro).
