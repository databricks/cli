Harness over ../invariant: mutates curated configs and runs a real target script.
`run_fuzz.py` owns the seed loop and classifies deployed / rejected / gap / hang / bug.
`FUZZ_TARGET` in test.toml picks the target.

Each seed deletes or replaces fields on a MUTATE_BASES config, and may inject a curated optional
from INJECT. Free-form scalars sometimes get dangerous values (empty, whitespace, over-long,
control chars, int boundaries).

Helpers come from ../invariant/script.prepare (sourced explicitly; prepare/test.toml only merge
along the directory chain). Server stubs are copied into test.toml; script asserts they stay in
sync. Unmodeled routes return `TESTSERVER_GAP` and count as gaps.

A failure is a CLI bug. `LOG.repro` prints e.g.
`ENVFILTER=FUZZ_TARGET=no_drift FUZZ_SEED_START=<seed> FUZZ_SEED_COUNT=1 FUZZ_CHECK_DRIFT=0 task test-fuzz`
(`ENVFILTER` because `FUZZ_TARGET` is a matrix key).

`FUZZ_CHECK_DRIFT=0` (committed run) uses plan-determinism; `1` (`task test-fuzz` / nightly) uses
the exact no_drift check. Only the committed run is expected green; a red nightly is a finding to
triage.
