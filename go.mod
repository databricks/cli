module github.com/databricks/bricks

go 1.18

require (
	github.com/atotto/clipboard v0.1.4
	github.com/databricks/databricks-sdk-go v0.2.0
	github.com/ghodss/yaml v1.0.0 // MIT + NOTICE
	github.com/manifoldco/promptui v0.9.0 // BSD-3-Clause license
	github.com/mitchellh/go-homedir v1.1.0 // MIT
	github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8 // BSD-2-Clause
	github.com/sabhiram/go-gitignore v0.0.0-20210923224102-525f6e181f06 // MIT
	github.com/spf13/cobra v1.6.1 // Apache 2.0
	github.com/stretchr/testify v1.8.1 // MIT
	github.com/whilp/git-urls v1.0.0 // MIT
	golang.org/x/mod v0.7.0 // BSD-3-Clause
	gopkg.in/ini.v1 v1.67.0 // Apache 2.0
)

replace github.com/databricks/databricks-sdk-go v0.2.0 => ../databricks-sdk-go

require (
	github.com/fatih/color v1.13.0
	github.com/mattn/go-isatty v0.0.14
	github.com/nwidger/jsoncolor v0.3.1
	github.com/google/uuid v1.3.0
	github.com/hashicorp/go-version v1.6.0
	github.com/hashicorp/hc-install v0.4.0
	github.com/hashicorp/terraform-exec v0.17.3
	github.com/hashicorp/terraform-json v0.14.0
	golang.org/x/exp v0.0.0-20221031165847-c99f073a8326
	golang.org/x/sync v0.1.0
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211
)

require github.com/mattn/go-colorable v0.1.9 // indirect

require (
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/zclconf/go-cty v1.11.0 // indirect
	golang.org/x/crypto v0.1.0 // indirect
)

require (
	cloud.google.com/go/compute v1.13.0 // indirect
	cloud.google.com/go/compute/metadata v0.2.2 // indirect
	github.com/chzyer/readline v0.0.0-20180603132655-2972be24d48e // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.2.0 // indirect
	github.com/imdario/mergo v0.3.13
	github.com/inconshreveable/mousetrap v1.0.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.5
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/net v0.1.0 // indirect
	golang.org/x/oauth2 v0.0.0-20221014153046-6fdb5e3db783
	golang.org/x/sys v0.1.0 // indirect
	golang.org/x/text v0.4.0 // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	google.golang.org/api v0.103.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20221206210731-b1a01be3a5f6 // indirect
	google.golang.org/grpc v1.51.0 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
