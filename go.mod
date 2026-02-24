module github.com/databricks/cli

go 1.25.0

toolchain go1.25.7

require (
	dario.cat/mergo v1.0.2 // BSD 3-Clause
	github.com/BurntSushi/toml v1.6.0 // MIT
	github.com/Masterminds/semver/v3 v3.4.0 // MIT
	github.com/charmbracelet/bubbles v1.0.0 // MIT
	github.com/charmbracelet/bubbletea v1.3.10 // MIT
	github.com/charmbracelet/huh v0.8.0
	github.com/charmbracelet/lipgloss v1.1.0 // MIT
	github.com/databricks/databricks-sdk-go v0.112.0 // Apache 2.0
	github.com/fatih/color v1.18.0 // MIT
	github.com/google/uuid v1.6.0 // BSD-3-Clause
	github.com/gorilla/mux v1.8.1 // BSD 3-Clause
	github.com/gorilla/websocket v1.5.3 // BSD 2-Clause
	github.com/hashicorp/go-version v1.8.0 // MPL 2.0
	github.com/hashicorp/hc-install v0.9.3 // MPL 2.0
	github.com/hashicorp/terraform-exec v0.25.0 // MPL 2.0
	github.com/hashicorp/terraform-json v0.27.2 // MPL 2.0
	github.com/hexops/gotextdiff v1.0.3 // BSD 3-Clause "New" or "Revised" License
	github.com/manifoldco/promptui v0.9.0 // BSD-3-Clause
	github.com/mattn/go-isatty v0.0.20 // MIT
	github.com/nwidger/jsoncolor v0.3.2 // MIT
	github.com/palantir/pkg/yamlpatch v1.5.0 // BSD-3-Clause
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // BSD-2-Clause
	github.com/quasilyte/go-ruleguard/dsl v0.3.22 // BSD 3-Clause
	github.com/sabhiram/go-gitignore v0.0.0-20210923224102-525f6e181f06 // MIT
	github.com/spf13/cobra v1.10.2 // Apache 2.0
	github.com/spf13/pflag v1.0.10 // BSD-3-Clause
	github.com/stretchr/testify v1.11.1 // MIT
	go.yaml.in/yaml/v3 v3.0.4 // MIT, Apache 2.0
	golang.org/x/crypto v0.48.0 // BSD-3-Clause
	golang.org/x/exp v0.0.0-20260112195511-716be5621a96
	golang.org/x/mod v0.33.0
	golang.org/x/oauth2 v0.35.0
	golang.org/x/sync v0.19.0
	golang.org/x/sys v0.41.0
	golang.org/x/text v0.34.0
	gopkg.in/ini.v1 v1.67.1 // Apache 2.0
)

// Dependencies for experimental MCP commands
require github.com/google/jsonschema-go v0.4.2 // MIT

require gopkg.in/yaml.v3 v3.0.1 // indirect

require (
	cloud.google.com/go/auth v0.18.1 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	github.com/ProtonMail/go-crypto v1.3.0 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/catppuccin/go v0.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/charmbracelet/colorprofile v0.4.1 // indirect
	github.com/charmbracelet/x/ansi v0.11.6 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.15 // indirect
	github.com/charmbracelet/x/exp/strings v0.0.0-20240722160745-212f7b056ed0 // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/chzyer/readline v0.0.0-20180603132655-2972be24d48e // indirect
	github.com/clipperhouse/displaywidth v0.9.0 // indirect
	github.com/clipperhouse/stringish v0.1.1 // indirect
	github.com/clipperhouse/uax29/v2 v2.5.0 // indirect
	github.com/cloudflare/circl v1.6.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/go-querystring v1.2.0 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.11 // indirect
	github.com/googleapis/gax-go/v2 v2.17.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.8 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.3.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.19 // indirect
	github.com/mitchellh/hashstructure/v2 v2.0.2 // indirect
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	github.com/zclconf/go-cty v1.17.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.65.0 // indirect
	go.opentelemetry.io/otel v1.40.0 // indirect
	go.opentelemetry.io/otel/metric v1.40.0 // indirect
	go.opentelemetry.io/otel/trace v1.40.0 // indirect
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/time v0.14.0 // indirect
	google.golang.org/api v0.265.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260203192932-546029d2fa20 // indirect
	google.golang.org/grpc v1.78.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
