# PR #3908 Review Comments

**PR Title:** Improve apps logs streaming helpers
**PR URL:** https://github.com/databricks/cli/pull/3908
**Branch:** logs-subcommand

## Overview

This directory contains individual files for each review comment from November 14, 2025 onwards.
Each file can be assigned to different team members for parallel work.

## Priority Order

### HIGH Priority (Blockers)
1. **01-refactor-token-handling.md** - Major architectural change, affects multiple other items
2. **08-fix-tail-lines-with-follow.md** - Bug that needs fixing

### MEDIUM Priority
3. **02-move-logstream-package.md** - Organizational change

### LOW Priority (Do after high/medium)
4. **07-format-constants.md** - Code style (depends on #02)
5. **06-move-skill-file.md** - File organization
6. **10-address-comment-at-240.md** - Need to find original comment

### LOW Priority (May be obsolete)
These depend on whether token handling code is kept or removed:
- **03-convert-to-table-tests.md**
- **04-remove-persistent-auth-opts-check.md**
- **05-simplify-test-setup.md**

### Follow-up (Separate PR)
- **09-handle-output-flag.md** - Not a blocker, can be done later

## Suggested Workflow

1. Start with **#01** (token handling refactor) - This is the biggest change
2. Do **#02** (move logstream package) in parallel or after #01
3. Fix **#08** (tail-lines bug) - Can be done independently
4. After #01 is done, evaluate if #03, #04, #05 are still needed
5. Clean up with #06, #07, #10
6. Defer #09 to a follow-up PR

## Reviewers

- **@pietern** - Provided major architectural feedback on Nov 17
- **@pkosiec** - Detailed code review on Nov 14

## Notes

- Files in `scratch/` directory are NOT committed to git
- Each file is self-contained with context and action items
- Check dependencies between items before starting work
