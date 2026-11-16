# Automated CLI PR reviews

`review-oss-bot` reviews same-repository PRs targeting `main` when they are
opened, reopened, marked ready, or updated with new commits. The author must
have write, maintain, or admin access. Drafts wait until they are marked ready.

To request another review, add a PR conversation comment containing exactly:

```text
/review
```

The requester must also have write, maintain, or admin access. This command
works on drafts too. Fork PRs and closed PRs are not eligible.

Find the **CLI PR review** run in the CLI repository's Actions tab. Its job
summary links to the exact backend run and waits for its outcome. A new request
cancels any unfinished backend review for that PR. The bot posts native inline
comments; it posts nothing when there are no findings. If the PR changes before
publication, the backend discards that outdated review.

The trigger runs trusted workflow code and never checks out the PR. The backend
in `databricks-eng/eng-dev-ecosystem` independently validates the author and PR,
reads source snapshots with restricted tools, and publishes through a separate
job. Reviews provide advisory feedback; the trigger does not approve or merge PRs.

The backend workflow also supports manual runs from its Actions tab. Use that
for diagnosis, including before the CLI trigger is merged.
