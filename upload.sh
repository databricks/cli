#!/bin/sh
set -eu

HOST="arca.ssh"
FILES="install.sh databricks-darwin-amd64 databricks-darwin-arm64 databricks-linux-amd64 databricks-linux-arm64"

for f in $FILES; do
    printf "Uploading %s...\n" "$f"
    scp "$f" "$HOST:~/"
    ssh "$HOST" "~/unp-upload.sh ~/$f"
done

printf "\nDone.\n"
