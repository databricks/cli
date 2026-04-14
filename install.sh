#!/bin/sh
# Lakebox CLI installer — . <(curl -s devbox.dbrx.dev)

_lakebox_install() {
    INSTALL_DIR="$HOME/.lakebox/bin"
    REMOTE_NAME="databricks"
    LOCAL_NAME="lakebox"
    BASE_URL="https://devbox.dbrx.dev"

    case "$(uname -s)" in
        Linux*)  OS="linux" ;;
        Darwin*) OS="darwin" ;;
        *)  printf "error: unsupported OS: %s\n" "$(uname -s)" >&2; return 1 ;;
    esac

    case "$(uname -m)" in
        x86_64|amd64)  ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *)  printf "error: unsupported arch: %s\n" "$(uname -m)" >&2; return 1 ;;
    esac

    url="${BASE_URL}/${REMOTE_NAME}-${OS}-${ARCH}"

    printf "📦 Installing Lakebox CLI (%s/%s)...\n" "$OS" "$ARCH"

    mkdir -p "$INSTALL_DIR" || { printf "error: could not create %s\n" "$INSTALL_DIR" >&2; return 1; }

    if command -v curl >/dev/null 2>&1; then
        curl -fSL --progress-bar "$url" -o "$INSTALL_DIR/$LOCAL_NAME" || { printf "error: download failed\n" >&2; return 1; }
    elif command -v wget >/dev/null 2>&1; then
        wget -q --show-progress "$url" -O "$INSTALL_DIR/$LOCAL_NAME" || { printf "error: download failed\n" >&2; return 1; }
    else
        printf "error: curl or wget is required\n" >&2; return 1
    fi

    chmod +x "$INSTALL_DIR/$LOCAL_NAME"

    PATH_LINE="export PATH=\"\$HOME/.lakebox/bin:\$PATH\""
    case ":$PATH:" in
        *":$INSTALL_DIR:"*) ;;
        *)
            added=0
            for rc in "$HOME/.zshrc" "$HOME/.bashrc"; do
                [ -f "$rc" ] || continue
                if ! grep -qF '.lakebox/bin' "$rc" 2>/dev/null; then
                    printf '\n# Lakebox CLI\n%s\n' "$PATH_LINE" >> "$rc"
                    printf "📝 Updated %s\n" "$rc"
                    added=1
                fi
            done
            if [ "$added" = 0 ]; then
                if [ "$OS" = "darwin" ]; then
                    rc="$HOME/.zshrc"
                else
                    rc="$HOME/.bashrc"
                fi
                printf '\n# Lakebox CLI\n%s\n' "$PATH_LINE" >> "$rc"
                printf "📝 Updated %s\n" "$rc"
            fi
            export PATH="$INSTALL_DIR:$PATH"
            ;;
    esac

    printf "\n✅ Lakebox CLI installed to %s\n" "$INSTALL_DIR/$LOCAL_NAME"

    LAKEBOX_HOST="https://dbsql-dev-testing-default.dev.databricks.com"
    LAKEBOX_PROFILE="dbsql-dev-testing-default"
    if ! grep -qF "$LAKEBOX_PROFILE" "$HOME/.databrickscfg" 2>/dev/null; then
        printf "\n🔑 Logging in...\n"
        lakebox auth login --host "$LAKEBOX_HOST" --profile "$LAKEBOX_PROFILE"
    fi

    printf "\nCommon workflows:\n"
    printf "  lakebox ssh                             # SSH to your default lakebox\n"
    printf "  lakebox ssh my-project                  # SSH to a named lakebox\n"
    printf "  lakebox list                            # list your lakeboxes\n"
}

_lakebox_install
unset -f _lakebox_install