---
name: pr-review
description: Review a branch against its merge base with main as a maintainer of this repo would, looking for correctness bugs, duplicated mechanisms, poor fit with existing extension points, behavior changes on empty input, stored state that goes stale, missed precedent, user-facing naming and text, imprecise doc comments, and weak or misplaced tests. Use when asked to review a branch, a commit, or a PR in this repository.
---

# PR review

Review the change as a maintainer who knows this codebase would. The question
is both "is this correct" and "does this fit what is already here". Produce a
small number of short, specific comments that make the author look at
something they missed.

Judge the change against what the codebase already does, not against the
author's own design. The PR description tells you what the author intended;
your job includes asking whether that intent is right. "This is how the new
code is designed" never justifies a behavior on its own.

## 1. Collect the change

The review target is `HEAD` unless you were given a commit. Set `TARGET` to it
and `BASE` to its merge base with main:

```bash
TARGET=HEAD
BASE=$(git merge-base "$TARGET" origin/main)
git diff --stat "$BASE" "$TARGET"
git diff "$BASE" "$TARGET"
```

If the change has a PR, read its title and description to learn the intent.
Do not read existing review comments before forming your own view. List the
hunks the description does not explain: a fix riding along with the main
change belongs in its own PR (`AGENTS.md`: one change per PR), so ask for
the split.

Check how far the branch is from main with
`git rev-list --count "$BASE"..origin/main` and
`git merge-tree --write-tree "$TARGET" origin/main`. When the base is far
behind, files the change touches may have moved, split or been deleted on
main, and conventions it follows (changelog location, test matrix variables)
may have changed. Those are judged at merge time, not at `TARGET`, so name
the new location in the review body.

Read every changed non-generated file in full at `TARGET`, not just the hunks
(`git show "$TARGET":path`). Skim generated files (schemas, codegen output,
acceptance test output) only to confirm they follow from the source change;
when the generator is cheap to run, regenerate into a temporary directory and
`diff -r` it against the committed tree, since stale generated files outside
the diff never show up in the hunks.
Read the repo's agent rules that apply to the changed files, under
`.agents/rules/` (testing, comments, language style guides).

## 2. Build situational awareness

Before judging the change, find what it touches in the rest of the codebase.
For every new type, function, flag, env var, config field, or code path:

- Search the codebase at `TARGET` (`git grep`) for code that already does the
  same job: the same noun combined with the same verb ("resolve", "match",
  "load", "parse", "retry"). Look in the same package first, then callers.
- Read the callers of the functions the change modifies, and the code that
  runs before and after the new code.
- Look for the helper the codebase normally uses for the small things the
  change does by hand: reading env vars, parsing booleans, normalizing paths
  or hosts, formatting errors, retrying. In this repo, `libs/env` has `Get`,
  `Lookup` and `GetBool`.
- When the change edits a constant, regex or literal, also grep a distinctive
  fragment of the old value: copies of a constant do not share its name.
  Under a "keep in sync with X" comment, check every copy, not only the ones
  the comment lists, and check that the parity test has an input that
  exercises the change.
- For every new option, flag or branch, find the concrete input that makes
  it do something (a file, a directory, an API response, a config value) and
  confirm that input can still be produced at `TARGET`. A mechanism whose
  trigger no longer exists is a finding.
- When the change configures a pinned third-party action or tool, read that
  tool's README or source at the pinned version
  (`gh api repos/<owner>/<repo>/contents/README.md?ref=<sha>`) and verify
  each option and default the change relies on.

Note, for yourself, how the new code differs from each existing piece you
find.

## 3. Required passes

Do every pass. Passes 1, 4, 5, 6, 7 and 8 each produce a table for the
worksheet, even when they find nothing; filling in the table is how you find
the comments. Each finding must be backed by code you read; drop anything you
can't point to, and re-check it against the code before keeping it. Comment
on code the change adds or modifies, not on pre-existing issues next to it.

1. **Correctness.** For each changed function, trace the main path and the
   edge cases: nil or empty values, errors that are dropped or wrapped
   inconsistently, loops and early returns, concurrency and shared state,
   resources that aren't closed, off-by-one and ordering assumptions, and
   behavior that differs between platforms. For a new goroutine fan-out,
   write down what bounds the number live at once and what each allocates;
   if the change ships a benchmark, run it with `-benchmem`. For bundle
   config changes, run the edge inputs through `bundle validate` in a
   scratch directory and quote the real output in the comment.
   Table: `line | input or state | what happens | is that right?`
2. **One mechanism per job.** Does the change add a second way to do
   something the codebase already does, or copy logic that now has to be kept
   in sync ("same logic as X")? Point at the existing one and ask how they
   differ and whether they can be unified. If they legitimately differ, check
   that the new one's doc comment says how.
3. **Use the existing extension point.** If the codebase composes behavior
   through a chain or registry (loaders, mutators, middleware, hooks, resource
   adapters), new behavior should plug into it instead of being special-cased
   in a caller. Say where it would go and what that buys. In particular, a
   step that reads remote or stored state to decide what to do (recreate,
   skip, migrate) usually belongs where the framework already reads state,
   such as a resource's read or diff method, not in a later step.
4. **Every new error path.** List each error return the change adds. For
   each, write down the input that reaches it and ask whether that input
   should fail at all. First write `Feature acts on: ...`: the input the
   change was introduced to handle, not every input it reads. When that input
   is absent, the code should normally leave the existing behavior in place.
   Fill "without the change" by tracing the same input through the code at
   `BASE`, not by restating the new code. "The new code requires it" is not a
   reason to answer "yes".
   Table: `line | input that reaches it | what happens without the change |
   should it fail?`
5. **Sibling routes check the same things.** Find the routes in the change
   that accept, select or process the same kind of object differently (for
   example create vs update, matched by value vs named explicitly, deploy vs
   destroy). Write down what the change promises, from the description and
   doc comments, and list per route which promises it keeps. If one route
   checks or handles something its sibling doesn't, ask about it as a short
   question.
   Table: `route (line) | promised property | handled here? (line or "no")`
6. **Every new env var, flag, config field and output field.** Check the name
   first: it is user-facing API and should use words users already see in
   config, docs and output, not an internal term from the implementation.
   Then check how the value is used: a presence-only check on an on/off
   switch means `false` and `0` turn it on. Check that new config fields
   follow how similar existing fields are declared, validated and documented.
   A field that newly accepts another YAML kind (a sequence where only a
   string was allowed) is new config API too: check that
   `bundle/schema/jsonschema.json` was regenerated to match.
   Table: `line | name | user-facing word or internal term? | how the value is
   used | existing precedent`
7. **Stored values over time.** For every value the change writes to state,
   files or caches, write down when it is first written, when it is updated,
   and who reads it later. Ask whether it goes stale after the first write,
   whether comments and output describe that lifecycle correctly, and what
   should happen when the stored value disagrees with the source of truth
   (often: an error, not a silent fallback). A marker or feature flag the
   code reads to decide whether to act is a stored value too: list who
   writes it and whether the change's new write path sets it.
   Table: `value (line) | written when | updated when | read by | can it
   disagree with the source of truth, and what happens then?`
8. **Every new doc comment.** List them:

   ```bash
   git diff "$BASE" "$TARGET" -- '*.go' ':!*_test.go' | grep -E '^\+(//|func |type |var |const )'
   ```

   Each comment above a new top-level declaration goes in the table. It
   should say what is decided, from which input, and where the result comes
   from. Flag comments that are wrong, that blur similar nouns, or that
   describe the mechanism but not the intent. Offer the sharper sentence.
   Also look for code that works around surprising behavior of an API or
   tool (adds a suffix, strips a prefix, retries a specific error) without
   a comment saying why; ask why it happens.
   Table: `line | what the comment claims | what the code does | precise
   enough?`
9. **Tests.** Review test code as carefully as the code under test.
   - Level: follow `.agents/rules/testing.md`. User-visible CLI behavior and
     the bundle mutator pipeline are tested with acceptance tests first; pure
     functions and helpers with unit tests next to the existing ones. Flag a
     unit test that should be an acceptance test, and logic with many input
     combinations that has no unit tests (name a case they would catch).
   - Does each new or changed test show the effect it claims? The output
     should include the state after the action (list or print it again after
     a sync, a deploy or a migration) and the exact value the change is
     about, not just a successful exit.
   - The fake server in `libs/testserver` only returns what its handlers were
     taught, so a test that records the request, or a handler that echoes a
     field back, shows what the CLI sends, not what the API accepts. When the
     change sends a new field, a new value format (paths, identifiers,
     durations) or a new call to a real API, and no test covering it has
     `Cloud = true` (`Cloud` is inherited, so check `out.test.toml`) and the
     description does not mention a run against a real workspace, ask whether
     it was verified against one.
   - Are fixtures readable? Prefer generating archives and other binary
     inputs in the test script over committing opaque files.
   - Does the test reuse the repo's existing test helpers and scripts, and
     keep helpers that a planned follow-up will need again?
   - One long script that walks many failure modes is hard to review:
     suggest splitting it by category.
10. **User-facing text.** Read every new or changed changelog fragment
    (`.nextchanges/`), error message, warning and help text. It should be
    short, accurate and written for users, follow `.nextchanges/README.md`
    for changelog entries, and reuse an existing constant instead of
    repeating the same text in several places. When the change lifts or
    alters a restriction, grep for the existing messages that describe the
    old behavior (the refusal, its hint): no hunk touches them, yet the
    change makes them wrong. When the change is user-visible and adds no
    `.nextchanges/` fragment, ask for one.

Skip formatting, naming of internal identifiers, anything a linter catches,
and questions about defensive checks.

## 4. Write the comments

**Coverage.** Every problem you confirmed gets a comment. There is no target
count: a small change may need one comment and a large one fifteen or more.
Never drop a real finding to keep the review short, and never pad a review
with weak ones to look thorough. A problem row in the worksheet is dropped
only if re-checking the code shows it is wrong, already handled, or
documented as intended (in a comment, a test, or the description); say which
in the Evidence section.

**Each comment must:**
- Point at the single line where a reader would trip (the error return, the
  vague comment, the presence check, the copied block).
- Say what is wrong in terms of behavior, with a concrete input and outcome:
  "With a plan saved while the snapshot was clean, `deploy --plan --force`
  stays on the modified snapshot, while `bundle deploy --force` moves to a
  fresh one". Name the other code involved by file and symbol when the
  comment depends on it.
- End with what you want: the change you'd make, or the question the author
  needs to answer. "Can we reuse `X` from `path`?", "Should this fail
  instead?". A comment that only describes is not finished.

**Voice:**
- Short and plain: usually one to three sentences, more only when the
  consequence needs a second example. No headings, bullets or severity labels
  inside a comment. Don't restate the code or list test cases.
- Prefer a question when the author may know something you don't: "We don't
  check X here?", "Can you look at how this differs from Y and whether they
  can be unified?"
- State what you expected when the code contradicts it: "I thought this
  would fall back to Y."
- Give an out on genuine judgment calls ("or is this not worth it?"), not on
  bugs.
- No praise, no verdict, no summary of the change.

**Order.** List inline comments by severity: wrong behavior users can hit
first, then design and duplication, then tests, then docs and wording. Put
nits (wording, small naming) together at the end, or in the review body as a
short list if there are several.

**Before printing:**
1. Check coverage: every problem row in the worksheet appears either as a
   comment or under "Not raised" in the Evidence section with the reason.
2. Re-check each comment against the code once more; fix or drop any that
   misread it.
3. Edit each comment: cut what doesn't help the author act, remove praise
   and verdicts, and drop any comment that answers its own question.
4. Raise each issue once. When the same problem appears in several places,
   comment on the first and list the other locations in that comment. The
   review body doesn't repeat inline comments; use it for cross-cutting asks
   (missing tests, a design question that spans files).
5. If the PR already has review threads, yours or another reviewer's, read
   them now. Drop a comment that repeats a point already raised there or
   that the author already answered (verification evidence often lives in a
   thread, not in the description), and add a short one where an earlier
   ask was not addressed.

## 5. Output

Print, do not post:

```
## Inline comments

### path/to/file.go:LINE
<comment>

## Review body
<cross-cutting asks and grouped nits, or nothing>

## Evidence (not for posting)
- path/to/file.go:LINE: <what in the code backs each comment>

Not raised:
- <worksheet row>: <why: wrong on re-check, already handled, or documented as intended>

## Worksheet (not for posting)
<the tables from the passes>
```

Use line numbers from `TARGET`. Only post to GitHub if asked.
