# Acceptance tests

Checklist:
- Keep unit tests passing for changed packages.
- Run `make test-update` when generated acceptance outputs change.
- Add resource acceptance coverage under `acceptance/bundle/resources/<resource>/...`.
- Add or update invariant coverage under `acceptance/bundle/invariant/` when applicable.
- Add interaction tests if the new resource has behavior coupled to other resources.

Canonical test framework guidance lives in:
- `acceptance/README.md`
- `acceptance/bundle/invariant/README.md`
- `bundle/tests/README.md`
