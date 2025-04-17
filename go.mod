module github.com/databricks/cli

go 1.24.0

toolchain go1.24.2

require (
	dario.cat/mergo v1.0.1 // BSD 3-Clause
	github.com/BurntSushi/toml v1.5.0 // MIT
	github.com/Masterminds/semver/v3 v3.3.1 // MIT
	github.com/briandowns/spinner v1.23.1 // Apache 2.0
	github.com/databricks/databricks-sdk-go v0.63.0 // Apache 2.0
	github.com/databricks/terraform-provider-databricks v1.73.0 // Apache 2.0
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
	github.com/wI2L/jsondiff v0.6.1 // MIT
	golang.org/x/exp v0.0.0-20240222234643-814bf88cf225
	golang.org/x/mod v0.24.0
	golang.org/x/oauth2 v0.29.0
	golang.org/x/sync v0.13.0
	golang.org/x/sys v0.32.0
	golang.org/x/term v0.31.0
	golang.org/x/text v0.24.0
	gopkg.in/ini.v1 v1.67.0 // Apache 2.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	cloud.google.com/go/auth v0.4.2 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.2 // indirect
	cloud.google.com/go/compute/metadata v0.5.2 // indirect
	github.com/ProtonMail/go-crypto v1.1.6 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/chzyer/readline v0.0.0-20180603132655-2972be24d48e // indirect
	github.com/cloudflare/circl v1.6.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.2 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-checkpoint v0.5.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-cty v1.5.0 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-plugin v1.6.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.7 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/hcl/v2 v2.23.0 // indirect
	github.com/hashicorp/logutils v1.0.0 // indirect
	github.com/hashicorp/terraform-plugin-framework v1.14.1 // indirect
	github.com/hashicorp/terraform-plugin-framework-validators v0.17.0 // indirect
	github.com/hashicorp/terraform-plugin-go v0.26.0 // indirect
	github.com/hashicorp/terraform-plugin-log v0.9.0 // indirect
	github.com/hashicorp/terraform-plugin-mux v0.18.0 // indirect
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.36.1 // indirect
	github.com/hashicorp/terraform-plugin-testing v1.12.0 // indirect
	github.com/hashicorp/terraform-registry-address v0.2.4 // indirect
	github.com/hashicorp/terraform-svchost v0.1.1 // indirect
	github.com/hashicorp/yamux v0.1.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/vmihailenco/msgpack v4.0.4+incompatible // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/zclconf/go-cty v1.16.2 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.49.0 // indirect
	go.opentelemetry.io/otel v1.31.0 // indirect
	go.opentelemetry.io/otel/metric v1.31.0 // indirect
	go.opentelemetry.io/otel/trace v1.31.0 // indirect
	golang.org/x/crypto v0.36.0 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.org/x/tools v0.31.0 // indirect
	google.golang.org/api v0.182.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241015192408-796eee8c2d53 // indirect
	google.golang.org/grpc v1.69.4 // indirect
	google.golang.org/protobuf v1.36.3 // indirect
)
