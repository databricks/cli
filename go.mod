module github.com/databricks/cli

go 1.24.0

toolchain go1.24.2

require (
	dario.cat/mergo v1.0.2 // BSD 3-Clause
	github.com/BurntSushi/toml v1.5.0 // MIT
	github.com/Masterminds/semver/v3 v3.4.0 // MIT
	github.com/briandowns/spinner v1.23.1 // Apache 2.0
	github.com/databricks/databricks-sdk-go v0.75.0 // Apache 2.0
	github.com/fatih/color v1.18.0 // MIT
	github.com/google/uuid v1.6.0 // BSD-3-Clause
	github.com/gorilla/mux v1.8.1 // BSD 3-Clause
	github.com/gorilla/websocket v1.5.3 // BSD 2-Clause
	github.com/hashicorp/go-version v1.7.0 // MPL 2.0
	github.com/hashicorp/hc-install v0.9.2 // MPL 2.0
	github.com/hashicorp/terraform-exec v0.23.0 // MPL 2.0
	github.com/hashicorp/terraform-json v0.24.0 // MPL 2.0
	github.com/hexops/gotextdiff v1.0.3 // BSD 3-Clause "New" or "Revised" License
	github.com/manifoldco/promptui v0.9.0 // BSD-3-Clause
	github.com/mattn/go-isatty v0.0.20 // MIT
	github.com/nwidger/jsoncolor v0.3.2 // MIT
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // BSD-2-Clause
	github.com/quasilyte/go-ruleguard/dsl v0.3.22 // BSD 3-Clause
	github.com/sabhiram/go-gitignore v0.0.0-20210923224102-525f6e181f06 // MIT
	github.com/spf13/cobra v1.9.1 // Apache 2.0
	github.com/spf13/pflag v1.0.6 // BSD-3-Clause
	github.com/stretchr/testify v1.10.0 // MIT
	golang.org/x/exp v0.0.0-20250620022241-b7579e27df2b
	golang.org/x/mod v0.26.0
	golang.org/x/oauth2 v0.30.0
	golang.org/x/sync v0.16.0
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/term v0.32.0
	golang.org/x/text v0.27.0
	gopkg.in/ini.v1 v1.67.0 // Apache 2.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	cloud.google.com/go/auth v0.16.2 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.7.0 // indirect
	github.com/ProtonMail/go-crypto v1.1.6 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/bitfield/gotestdox v0.2.2 // indirect
	github.com/bmatcuk/doublestar/v4 v4.7.1 // indirect
	github.com/chzyer/readline v0.0.0-20180603132655-2972be24d48e // indirect
	github.com/cloudflare/circl v1.6.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dnephin/pflag v1.0.7 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/google/yamlfmt v0.17.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.6 // indirect
	github.com/googleapis/gax-go/v2 v2.14.2 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.7 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/zclconf/go-cty v1.16.2 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.61.0 // indirect
	go.opentelemetry.io/otel v1.36.0 // indirect
	go.opentelemetry.io/otel/metric v1.36.0 // indirect
	go.opentelemetry.io/otel/trace v1.36.0 // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/time v0.12.0 // indirect
	golang.org/x/tools v0.34.0 // indirect
	google.golang.org/api v0.238.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250603155806-513f23925822 // indirect
	google.golang.org/grpc v1.73.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gotest.tools/gotestsum v1.12.1 // indirect
)

tool (
	github.com/google/yamlfmt/cmd/yamlfmt
	gotest.tools/gotestsum
)
