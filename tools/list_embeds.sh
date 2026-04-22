#!/bin/sh
# Print a single brace-expansion glob of every //go:embed target in the repo.
# Used as the EMBED_SOURCES dynamic var in Taskfile.yml so tasks can reference
# embed dependencies without hand-maintaining a list. Testdata directories are
# covered by a separate static `**/testdata/**` glob, not this script.
set -e
root="$(git rev-parse --show-toplevel)"
cd "$root"
go list -f '{{$d := .Dir}}{{range .EmbedPatterns}}{{$d}}/{{.}}
{{end}}{{range .TestEmbedPatterns}}{{$d}}/{{.}}
{{end}}{{range .XTestEmbedPatterns}}{{$d}}/{{.}}
{{end}}' ./... \
  | sed "s|^$root/||; s|/all:|/|" \
  | while IFS= read -r p; do
      [ -z "$p" ] && continue
      if [ -d "$p" ]; then
        printf '%s/**\n' "$p"
      elif [ -f "$p" ]; then
        printf '%s\n' "$p"
      fi
    done \
  | sort -u \
  | paste -sd, - \
  | sed 's|^|{|; s|$|}|'
