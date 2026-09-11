# BundleTest demo transcript

Target length: about 80 seconds at a natural presentation pace.

## 1. Opening — 0:00

> Databricks bundles are easy to declare, but today we often discover wiring bugs only after
> a deploy and a real job run. BundleTest brings the pytest feedback loop to Databricks
> bundles.

## 2. The problem — 0:10

> A unit test can prove that a transform function works. It cannot prove that the deployed
> job points at the right SQL file, reads the right table, or writes the expected output.
> Those mistakes turn a tiny bug into a multi-minute feedback loop.

## 3. One local command — 0:22

> Now I can run Databricks bundle test locally. Before pytest starts, BundleTest shows exactly
> which tasks run in DuckDB, which need a workspace, and which resources support configuration
> assertions.

## 4. Test the real artifact — 0:33

> The test seeds the upstream boundary, runs the job's actual SQL artifact, and asserts on the
> resulting table. The query body is never mocked or rewritten, so a wrong table name stays a
> real failure.

## 5. Useful failures — 0:44

> When a run fails, the assertion includes the bundle resource, task, source file, backend,
> run identifier when available, and the original engine error. The next debugging step is
> visible immediately.

## 6. Run only affected tests — 0:55

> For fast pull request checks, changed mode maps edited bundle files back to their resources
> and selects tests through pytest markers. Changed YAML safely runs the complete suite.

## 7. Same test, real workspace — 1:05

> When local fidelity is not enough, the same test runs on Databricks with an explicit profile.
> Local runs provide speed, cloud runs provide full fidelity, and teams keep the pytest tools
> they already know.

## 8. Closing — 1:16

> BundleTest catches bundle wiring and data contract bugs before they become slow deployment
> failures. It is pytest for Databricks bundles.
