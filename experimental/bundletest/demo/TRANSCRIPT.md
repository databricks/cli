# BundleTest demo transcript

Target length: about 90 seconds, including short pauses between scenes.

## 1. Project title — 0:00–0:10

> Meet BundleTest: a faster way to test Databricks bundles before deployment. It helps teams
> catch broken connections and unexpected data results while fixes are still fast and cheap.

## 2. What — 0:10–0:22

> A Databricks bundle packages jobs, pipelines, and their setup as code. Traditional tests can
> verify one function, but they cannot prove the deployed job points to the right file, reads
> the right table, or produces the expected result.

## 3. Who — 0:22–0:33

> That gap affects data engineers, analytics engineers, and platform teams: the people building
> and reviewing production data workflows. Today, a tiny configuration mistake can wait until
> a full deployment to appear.

## 4. Why — 0:33–0:44

> That means minutes of waiting, cloud compute, slower pull requests, and less confidence in
> every release. The earlier these bugs surface, the cheaper they are to understand and fix.

## 5. How: one command — 0:44–0:57

> BundleTest adds one familiar command: Databricks bundle test. Before tests begin, it explains
> what can run instantly on a laptop, what needs a real workspace, and which setup rules it can
> verify, so every result has a clear meaning.

## 6. How: test the real artifact — 0:57–1:09

> A test supplies realistic input, runs the workflow's actual SQL without rewriting it, and
> checks the real output. If a connection is wrong, the failure points to the exact resource,
> task, source file, test environment, and original error.

## 7. How: two confidence levels — 1:09–1:20

> Developers get feedback in seconds, can focus on tests affected by their change, and then
> reuse the same test in Databricks when they need real workspace behavior.

## 8. Expected impact — 1:20–1:30

> The expected impact: fewer failed deployments, faster reviews, lower compute waste, and
> stronger trust in bundle changes. BundleTest turns bundle validation from a late surprise
> into an everyday feedback loop.
