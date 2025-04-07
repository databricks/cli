"""
AI TODO: implement script. Use Python 3.9. Use pathlib. Do not add type annotations or redundant comments. Avoid wrapping exceptions in "nicer" error messages.

Find open PRs by non-team member (read first line in .github/CODEOWNERS) but approved by team member.
  Read most recent commit of such PR.

Find if there is cli-isolated-pr job running for this PR number & commit (Job URL: https://github.com/databricks-eng/eng-dev-ecosystem/actions/workflows/cli-isolated-pr.yml).
If not running, start one. Ask before starting,

"""
