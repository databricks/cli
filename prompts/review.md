# PR Review Guidelines

## Overview
You are reviewing a Pull Request for the Databricks CLI. Please conduct a thorough code review following these guidelines.

## Review Checklist

### Code Quality
- [ ] Code follows the project's style guide and conventions
- [ ] Functions and variables have clear, descriptive names

### Testing
- [ ] Appropriate test coverage for new functionality
- [ ] Tests are clear and test the right behavior
- [ ] Golden files in acceptance tests are valid and make sense. Any error messages are readable and understandable as a new user.

### Edge Case Analysis
- [ ] Identify edge cases for the changed code (e.g. empty inputs, nil values, boundary conditions, error paths)
- [ ] Verify edge cases have acceptance test coverage

### Documentation
- [ ] Code includes necessary comments for complex logic
- [ ] Public APIs are documented
- [ ] README or other docs updated if needed

### Dependencies
- [ ] Any new dependencies introduced (e.g. Github Actions, go.mod libraries) are from a trusted source, i.e. either a popular public project or trusted entity like Databricks.
- [ ] All 3rd party Github actions should be pinned to a specific commit commit

### PR metadata
- [ ] The PR title should be clean and descriptive.
- [ ] The PR description should provide adequate details about the changes in the PR.


## Review Process
1. **Understand the Context**: Read the PR description, linked issues and the code / changes.
2. **Review code coverage**: For all changed files / code, ensure that the code being added is adequately tested. Strong preference should be given to end-to-end acceptance tests in acceptance/ over unit tests, unless the change being tested is a well defined and clean function / library.
3. **Identify gaps in testing / detect bugs (important)** Identify and construct edge cases that are not covered in existing tests. Contruct and run relevant acceptance tests (preferred) and unit tests to ensure these edge cases have a good behavior. Report when missing coverage is detected.
4. **Ensure adequate documentation**: Non trivial functions / code changes should have the appropriate amount of doc strings, so that's a new visitor to the code can quickly understand what's happening. This is even more important for abstract changes that use reflection for example.

## Output Format

Please provide your review in the following format:

### Summary
Brief overview of the changes and overall assessment.

### Positive Aspects
What was done well.

### Issues Found
List any bugs, errors, or serious concerns. Any edge cases that fail or should be covered.

### Suggestions
Recommendations for improvement (not blocking).

### Questions
Any clarifications needed.

## Command reference:
1. You can run and generate golden files for local acceptance tests by running: go test -timeout 300s -run ^TestAccept/$1$ github.com/databricks/cli/acceptance -update <path-to-test-from-acceptance/> (e.g. selftest)
2. You can run and generate golden files for cloud acceptance tests by running: deco env run -i -n azure-prod-ucws -- go test -timeout 600s -run ^TestAccept/$1$ github.com/databricks/cli/acceptance -update <path-to-test-from-acceptance/>
