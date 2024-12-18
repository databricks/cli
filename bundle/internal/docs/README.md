## docs-autogen

1. Install [Golang](https://go.dev/doc/install)
2. Run `go mod download` from the repo root
3. Run `make docs` from the repo
4. See generated document in `./bundle/internal/docs/docs.md`
5. To change descriptions update content in `./bundle/internal/schema/annotations.yml` and re-run `make docs`
