# Version changelog

## 0.215.0

CLI:
* The SDK update fixes `fs cp` calls timing out when copying large files.

Bundles:
* Fix summary command when internal Terraform config doesn't exist ([#1242](https://github.com/databricks/cli/pull/1242)).
* Configure root.NoArgs for bundle commands where applicable ([#1250](https://github.com/databricks/cli/pull/1250)).
* Fixed building Python artifacts on Windows with WSL ([#1249](https://github.com/databricks/cli/pull/1249)).
* Add `--validate-only` flag to run validate-only pipeline update ([#1251](https://github.com/databricks/cli/pull/1251)).
* Only transform wheel libraries when using trampoline ([#1248](https://github.com/databricks/cli/pull/1248)).
* Return `application_id` for service principal lookups ([#1245](https://github.com/databricks/cli/pull/1245)).
* Support relative paths in artifact files source section and always upload all artifact files ([#1247](https://github.com/databricks/cli/pull/1247)).
* Fix DBConnect support in VS Code ([#1253](https://github.com/databricks/cli/pull/1253)).

Internal:
* Added test to verify scripts.Execute mutator works correctly ([#1237](https://github.com/databricks/cli/pull/1237)).

API Changes:
* Added `databricks permission-migration` command group.
* Updated nesting of the `databricks settings` and `databricks account settings commands`
* Changed `databricks vector-search-endpoints delete-endpoint` command with new required argument order.
* Changed `databricks vector-search-indexes create-index` command with new required argument order.
* Changed `databricks vector-search-indexes delete-data-vector-index` command with new required argument order.
* Changed `databricks vector-search-indexes upsert-data-vector-index` command with new required argument order.

OpenAPI commit d855b30f25a06fe84f25214efa20e7f1fffcdf9e (2024-03-04)

Dependency updates:
* Bump github.com/stretchr/testify from 1.8.4 to 1.9.0 ([#1252](https://github.com/databricks/cli/pull/1252)).
* Update Go SDK to v0.34.0 ([#1256](https://github.com/databricks/cli/pull/1256)).
## 0.214.1

CLI:
 * Improved error message when no .databrickscfg ([#1223](https://github.com/databricks/cli/pull/1223)).
 * Use Go SDK Iterators when listing resources with the CLI ([#1202](https://github.com/databricks/cli/pull/1202)).

Bundles:
 * Only set ComputeID value when `--compute-id` flag provided ([#1229](https://github.com/databricks/cli/pull/1229)).
 * Add correct tag value for models in dev mode ([#1230](https://github.com/databricks/cli/pull/1230)).
 * Upgrade Terraform provider to 1.37.0 ([#1235](https://github.com/databricks/cli/pull/1235)).

Internal:
 * Fix CLI nightlies on our UC workspaces ([#1225](https://github.com/databricks/cli/pull/1225)).
 * Handle alias types for map keys in toTyped conversion ([#1232](https://github.com/databricks/cli/pull/1232)).



## 0.214.0

CLI:
 * Add support for UC Volumes to the `databricks fs` commands ([#1209](https://github.com/databricks/cli/pull/1209)).

Bundles:
 * Use dynamic configuration model in bundles ([#1098](https://github.com/databricks/cli/pull/1098)).
 * Allow use of variables references in primitive non-string fields ([#1219](https://github.com/databricks/cli/pull/1219)).
 * Add an experimental default-sql template ([#1051](https://github.com/databricks/cli/pull/1051)).
 * Add an experimental dbt-sql template ([#1059](https://github.com/databricks/cli/pull/1059)).

Internal:
 * Add fork-user to winget release workflow ([#1214](https://github.com/databricks/cli/pull/1214)).
 * Use `any` as type for data sources and resources in `tf/schema` ([#1216](https://github.com/databricks/cli/pull/1216)).
 * Avoid infinite recursion when normalizing a recursive type ([#1213](https://github.com/databricks/cli/pull/1213)).
 * Fix issue where interpolating a new ref would rewrite unrelated fields ([#1217](https://github.com/databricks/cli/pull/1217)).
 * Use `dyn.Value` as input to generating Terraform JSON ([#1218](https://github.com/databricks/cli/pull/1218)).

API Changes:
 * Changed `databricks lakehouse-monitors update` command with new required argument order.
 * Added `databricks online-tables` command group.

OpenAPI commit cdd76a98a4fca7008572b3a94427566dd286c63b (2024-02-19)
Dependency updates:
 * Bump Terraform provider to v1.36.2 ([#1215](https://github.com/databricks/cli/pull/1215)).
 * Bump github.com/databricks/databricks-sdk-go from 0.32.0 to 0.33.0 ([#1222](https://github.com/databricks/cli/pull/1222)).

## 0.213.0

CLI:
 * Ignore environment variables for `auth profiles` ([#1189](https://github.com/databricks/cli/pull/1189)).
 * Update LICENSE file to match Databricks license language ([#1013](https://github.com/databricks/cli/pull/1013)).

Bundles:
 * Added `bundle deployment bind` and `unbind` command ([#1131](https://github.com/databricks/cli/pull/1131)).
 * Use allowlist for Git-related fields to include in metadata ([#1187](https://github.com/databricks/cli/pull/1187)).
 * Added `--restart` flag for `bundle run` command ([#1191](https://github.com/databricks/cli/pull/1191)).
 * Generate correct YAML if `custom_tags` or `spark_conf` is used for pipeline or job cluster configuration ([#1210](https://github.com/databricks/cli/pull/1210)).

Internal:
 * Move folders package into libs ([#1184](https://github.com/databricks/cli/pull/1184)).
 * Log time it takes for profile to load ([#1186](https://github.com/databricks/cli/pull/1186)).
 * Use mockery to generate mocks compatible with testify/mock ([#1190](https://github.com/databricks/cli/pull/1190)).
 * Retain partially valid structs in `convert.Normalize` ([#1203](https://github.com/databricks/cli/pull/1203)).
 * Skip `for_each_task` when generating the bundle schema ([#1204](https://github.com/databricks/cli/pull/1204)).
 * Regenerate the CLI using the same OpenAPI spec as the SDK ([#1205](https://github.com/databricks/cli/pull/1205)).
 * Avoid race-conditions while executing sub-commands ([#1201](https://github.com/databricks/cli/pull/1201)).

API Changes:
 * Added `databricks tables exists` command.
 * Added `databricks lakehouse-monitors` command group.
 * Removed `databricks files get-status` command.
 * Added `databricks files create-directory` command.
 * Added `databricks files delete-directory` command.
 * Added `databricks files get-directory-metadata` command.
 * Added `databricks files get-metadata` command.
 * Added `databricks files list-directory-contents` command.
 * Removed `databricks pipelines reset` command.
 * Changed `databricks account settings delete-personal-compute-setting` command with new required argument order.
 * Removed `databricks account settings read-personal-compute-setting` command.
 * Changed `databricks account settings update-personal-compute-setting` command with new required argument order.
 * Added `databricks account settings get-personal-compute-setting` command.
 * Removed `databricks settings delete-default-workspace-namespace` command.
 * Removed `databricks settings read-default-workspace-namespace` command.
 * Removed `databricks settings update-default-workspace-namespace` command.
 * Added `databricks settings delete-default-namespace-setting` command.
 * Added `databricks settings delete-restrict-workspace-admins-setting` command.
 * Added `databricks settings get-default-namespace-setting` command.
 * Added `databricks settings get-restrict-workspace-admins-setting` command.
 * Added `databricks settings update-default-namespace-setting` command.
 * Added `databricks settings update-restrict-workspace-admins-setting` command.
 * Changed `databricks token-management create-obo-token` command with new required argument order.
 * Changed `databricks token-management get` command to return .
 * Changed `databricks dashboards create` command . New request type is .
 * Added `databricks dashboards update` command.

OpenAPI commit c40670f5a2055c92cf0a6aac92a5bccebfb80866 (2024-02-14)
Dependency updates:
 * Bump github.com/hashicorp/hc-install from 0.6.2 to 0.6.3 ([#1200](https://github.com/databricks/cli/pull/1200)).
 * Bump golang.org/x/term from 0.16.0 to 0.17.0 ([#1197](https://github.com/databricks/cli/pull/1197)).
 * Bump golang.org/x/oauth2 from 0.16.0 to 0.17.0 ([#1198](https://github.com/databricks/cli/pull/1198)).
 * Bump github.com/databricks/databricks-sdk-go from 0.30.1 to 0.32.0 ([#1199](https://github.com/databricks/cli/pull/1199)).

## 0.212.4

Bundles:
 * Allow specifying executable in artifact section and skip bash from WSL ([#1169](https://github.com/databricks/cli/pull/1169)).
 * Added warning when trying to deploy bundle with `--fail-on-active-runs` and running resources ([#1163](https://github.com/databricks/cli/pull/1163)).
 * Group bundle run flags by job and pipeline types ([#1174](https://github.com/databricks/cli/pull/1174)).
 * Make sure grouped flags are added to the command flag set ([#1180](https://github.com/databricks/cli/pull/1180)).
 * Add short_name helper function to bundle init templates ([#1167](https://github.com/databricks/cli/pull/1167)).

Internal:
 * Fix dynamic representation of zero values in maps and slices ([#1154](https://github.com/databricks/cli/pull/1154)).
 * Refactor library to artifact matching to not use pointers ([#1172](https://github.com/databricks/cli/pull/1172)).
 * Harden `dyn.Value` equality check ([#1173](https://github.com/databricks/cli/pull/1173)).
 * Ensure every variable reference is passed to lookup function ([#1176](https://github.com/databricks/cli/pull/1176)).
 * Empty struct should yield empty map in `convert.FromTyped` ([#1177](https://github.com/databricks/cli/pull/1177)).
 * Zero destination struct in `convert.ToTyped` ([#1178](https://github.com/databricks/cli/pull/1178)).
 * Fix integration test with invalid configuration ([#1182](https://github.com/databricks/cli/pull/1182)).
 * Use `acc.WorkspaceTest` helper from bundle integration tests ([#1181](https://github.com/databricks/cli/pull/1181)).

## 0.212.3

CLI:
 * Release Windows packages to winget-pkgs ([#1144](https://github.com/databricks/cli/pull/1144)).

Bundles:
 * Add `--key` flag for generate commands to specify resource key ([#1165](https://github.com/databricks/cli/pull/1165)).


Dependency updates:
 * Bump github.com/google/uuid from 1.5.0 to 1.6.0 ([#1160](https://github.com/databricks/cli/pull/1160)).
 * Update Go SDK to v0.30.1 ([#1162](https://github.com/databricks/cli/pull/1162)).

## 0.212.2

CLI:
 * Prompt for account profile only for account-level command execution instead of during `databricks labs install` flow ([#1128](https://github.com/databricks/cli/pull/1128)).
 * Bring back `--json` flag for workspace-conf set-status command ([#1151](https://github.com/databricks/cli/pull/1151)).

Bundles:
 * Set `run_as` permissions after variable interpolation ([#1141](https://github.com/databricks/cli/pull/1141)).
 * Add functionality to visit values in `dyn.Value` tree ([#1142](https://github.com/databricks/cli/pull/1142)).
 * Add `dynvar` package for variable resolution with a `dyn.Value` tree ([#1143](https://github.com/databricks/cli/pull/1143)).
 * Add support for `anyOf` to `skip_prompt_if` ([#1133](https://github.com/databricks/cli/pull/1133)).
 * Added `bundle generate pipeline` command ([#1139](https://github.com/databricks/cli/pull/1139)).

Internal:
 * Use MockWorkspaceClient from SDK instead of WithImpl mocking ([#1134](https://github.com/databricks/cli/pull/1134)).

Dependency updates:
 * Bump github.com/databricks/databricks-sdk-go from 0.29.0 to 0.29.1 ([#1137](https://github.com/databricks/cli/pull/1137)).
 * Bump github.com/hashicorp/terraform-json from 0.20.0 to 0.21.0 ([#1138](https://github.com/databricks/cli/pull/1138)).
 * Update actions/setup-go to v5 ([#1148](https://github.com/databricks/cli/pull/1148)).
 * Update codecov/codecov-action to v3 ([#1149](https://github.com/databricks/cli/pull/1149)).
 * Use latest patch release of Go toolchain ([#1152](https://github.com/databricks/cli/pull/1152)).

## 0.212.1

CLI:
 * Fix windows style file paths in fs cp command ([#1118](https://github.com/databricks/cli/pull/1118)).
 * Do not require positional arguments if they should be provided in JSON ([#1125](https://github.com/databricks/cli/pull/1125)).
 * Always require path parameters as positional arguments ([#1129](https://github.com/databricks/cli/pull/1129)).

Bundles:
 * Add debug log line for when bundle init is run from non-TTY interface ([#1117](https://github.com/databricks/cli/pull/1117)).
 * Added `databricks bundle generate job` command ([#1043](https://github.com/databricks/cli/pull/1043)).
 * Support passing job parameters to bundle run ([#1115](https://github.com/databricks/cli/pull/1115)).

Dependency updates:
 * Bump golang.org/x/oauth2 from 0.15.0 to 0.16.0 ([#1124](https://github.com/databricks/cli/pull/1124)).

## 0.212.0

CLI:
 * Tweak command groups in CLI help ([#1092](https://github.com/databricks/cli/pull/1092)).
 * Fixed storage-credentials list command in text output ([#1094](https://github.com/databricks/cli/pull/1094)).
 * Allow account client auth with environment variables when no `.databrickscfg` file present ([#1097](https://github.com/databricks/cli/pull/1097)).
 * Always log with text format by default ([#1096](https://github.com/databricks/cli/pull/1096)).

Bundles:
 * Added exec.NewCommandExecutor to execute commands with correct interpreter ([#1075](https://github.com/databricks/cli/pull/1075)).
 * Use resource key as name in permissions code ([#1087](https://github.com/databricks/cli/pull/1087)).
 * Add support for reprompts if user input does not match template schema ([#946](https://github.com/databricks/cli/pull/946)).
 * Improve experience for multiple builtin templates ([#1052](https://github.com/databricks/cli/pull/1052)).
 * Improve error when bundle root is not writable ([#1093](https://github.com/databricks/cli/pull/1093)).
 * Change recommended production deployment path from `/Shared` to `/Users` ([#1091](https://github.com/databricks/cli/pull/1091)).
 * Allow referencing bundle resources by name ([#872](https://github.com/databricks/cli/pull/872)).
 * Make libs/exec fallback to `sh` if `bash` cannot be found ([#1114](https://github.com/databricks/cli/pull/1114)).

Internal:
 * Functionality to walk a `config.Value` tree ([#1081](https://github.com/databricks/cli/pull/1081)).
 * Rename libs/config -> libs/dyn ([#1086](https://github.com/databricks/cli/pull/1086)).
 * Added codecov integration on code push ([#1095](https://github.com/databricks/cli/pull/1095)).
 * Consolidate functions to convert `dyn.Value` to native types ([#1100](https://github.com/databricks/cli/pull/1100)).
 * Define constant for the invalid `dyn.Value` ([#1101](https://github.com/databricks/cli/pull/1101)).

API Changes:
 * Changed `databricks serving-endpoints update-config` command with new required argument order.
 * Changed `databricks account ip-access-lists create` command with new required argument order.
 * Changed `databricks account ip-access-lists replace` command with new required argument order.
 * Changed `databricks account ip-access-lists update` command with new required argument order.
 * Changed `databricks ip-access-lists create` command with new required argument order.
 * Changed `databricks ip-access-lists replace` command with new required argument order.
 * Changed `databricks ip-access-lists update` command with new required argument order.

OpenAPI commit a7a9dc025bb80303e676bf3708942c6aa06689f1 (2024-01-04)

Dependency updates:
 * Bump github.com/hashicorp/terraform-exec from 0.19.0 to 0.20.0 ([#1088](https://github.com/databricks/cli/pull/1088)).
 * Bump go.uber.org/mock from 0.3.0 to 0.4.0 ([#1089](https://github.com/databricks/cli/pull/1089)).
 * Bump github.com/hashicorp/terraform-json from 0.18.0 to 0.20.0 ([#1090](https://github.com/databricks/cli/pull/1090)).
 * Bump golang.org/x/term from 0.15.0 to 0.16.0 ([#1103](https://github.com/databricks/cli/pull/1103)).
 * Bump github.com/cloudflare/circl from 1.3.3 to 1.3.7 ([#1105](https://github.com/databricks/cli/pull/1105)).
 * Bump golang.org/x/sync from 0.5.0 to 0.6.0 ([#1104](https://github.com/databricks/cli/pull/1104)).
 * Upgrade golang.org/x/crypto to v0.17.0 in internal module ([#1110](https://github.com/databricks/cli/pull/1110)).
 * Upgrade Go SDK to v0.29.0 ([#1111](https://github.com/databricks/cli/pull/1111)).
## 0.211.0

CLI:
 * Upgrade Go SDK to 0.27.0 ([#1064](https://github.com/databricks/cli/pull/1064)).
 * Skip profile resolution if `DATABRICKS_AUTH_TYPE` is set ([#1068](https://github.com/databricks/cli/pull/1068)).
 * Do not allow input prompts in Git Bash terminal ([#1069](https://github.com/databricks/cli/pull/1069)).
 * Added output template for list-secrets command ([#1074](https://github.com/databricks/cli/pull/1074)).

Bundles:
* Set metadata fields required to enable break-glass UI for jobs ([#880](https://github.com/databricks/cli/pull/880)).
* Do not prompt for template values in Git Bash ([#1082](https://github.com/databricks/cli/pull/1082)).
* Tune output of bundle deploy command ([#1047](https://github.com/databricks/cli/pull/1047)).

API Changes:
 * Changed `databricks connections update` command with new required argument order.
 * Changed `databricks serving-endpoints update-config` command with new required argument order.
 * Added `databricks serving-endpoints put` command.
 * Removed `databricks account network-policy` command group.

OpenAPI commit 63caa3cb0c05045e81d3dcf2451fa990d8670f36 (2023-12-12)

Dependency updates:
 * Bump github.com/google/uuid from 1.4.0 to 1.5.0 ([#1073](https://github.com/databricks/cli/pull/1073)).
 * Bump golang.org/x/crypto from 0.16.0 to 0.17.0 ([#1076](https://github.com/databricks/cli/pull/1076)).

## 0.210.3

Bundles:
 * Improve default template ([#1046](https://github.com/databricks/cli/pull/1046)).
 * Fix passthrough of pipeline notifications ([#1058](https://github.com/databricks/cli/pull/1058)).

Internal:
 * Stub out Python virtual environment installation for `labs` commands ([#1057](https://github.com/databricks/cli/pull/1057)).
 * Upgrade Terraform schema version to v1.31.1 ([#1055](https://github.com/databricks/cli/pull/1055)).


Dependency updates:
 * Bump github.com/hashicorp/hc-install from 0.6.1 to 0.6.2 ([#1054](https://github.com/databricks/cli/pull/1054)).
 * Bump github.com/databricks/databricks-sdk-go from 0.26.1 to 0.26.2 ([#1053](https://github.com/databricks/cli/pull/1053)).

## 0.210.2

CLI:
 * Add documentation for positional args in commands generated from the Databricks OpenAPI specification ([#1033](https://github.com/databricks/cli/pull/1033)).
 * Ask for host when .databrickscfg doesn't exist ([#1041](https://github.com/databricks/cli/pull/1041)).
 * Add list of supported values for flags that represent an enum field ([#1036](https://github.com/databricks/cli/pull/1036)).

Bundles:
 * Fix panic when bundle auth resolution fails ([#1002](https://github.com/databricks/cli/pull/1002)).
 * Add versioning for bundle templates ([#972](https://github.com/databricks/cli/pull/972)).
 * Add support for conditional prompting in bundle init ([#971](https://github.com/databricks/cli/pull/971)).
 * Pass parameters to task when run with `--python-params` and `python_wheel_wrapper` is true ([#1037](https://github.com/databricks/cli/pull/1037)).
 * Change default_python template to auto-update version on each wheel build ([#1034](https://github.com/databricks/cli/pull/1034)).

Internal:
 * Rewrite the friendly log handler ([#1038](https://github.com/databricks/cli/pull/1038)).
 * Move bundle schema update to an internal module ([#1012](https://github.com/databricks/cli/pull/1012)).


Dependency updates:
 * Bump github.com/databricks/databricks-sdk-go from 0.26.0 to 0.26.1 ([#1040](https://github.com/databricks/cli/pull/1040)).

## 0.210.1

This is a bugfix release to address issues with v0.210.0.

CLI:
 * Fix `panic:  is not set` ([#1027](https://github.com/databricks/cli/pull/1027)).
 * Fix `databricks configure` if new profile is specified ([#1030](https://github.com/databricks/cli/pull/1030)).
 * Filter out system clusters for `--configure-cluster` ([#1031](https://github.com/databricks/cli/pull/1031)).

Bundles:
 * Fixed panic when job has trigger and in development mode ([#1026](https://github.com/databricks/cli/pull/1026)).

Internal:
 * Use `fetch-tags` option in release workflows ([#1025](https://github.com/databricks/cli/pull/1025)).



## 0.210.0

This release includes the new `databricks labs` command to install, manage, and run Databricks Labs projects.

CLI:
 * Add `--debug` as shortcut for `--log-level debug` ([#964](https://github.com/databricks/cli/pull/964)).
 * Improved usability of `databricks auth login ... --configure-cluster` ([#956](https://github.com/databricks/cli/pull/956)).
 * Make `databricks configure` save only explicit fields ([#973](https://github.com/databricks/cli/pull/973)).
 * Add `databricks labs` command group ([#914](https://github.com/databricks/cli/pull/914)).
 * Tolerate missing .databrickscfg file during `databricks auth login` ([#1003](https://github.com/databricks/cli/pull/1003)).
 * Add `--configure-cluster` flag to configure command ([#1005](https://github.com/databricks/cli/pull/1005)).
 * Fix bug where the account or workspace client could be `nil` ([#1020](https://github.com/databricks/cli/pull/1020)).

Bundles:
 * Do not allow empty descriptions for bundle template inputs ([#967](https://github.com/databricks/cli/pull/967)).
 * Added support for top-level permissions ([#928](https://github.com/databricks/cli/pull/928)).
 * Allow jobs to be manually unpaused in development mode ([#885](https://github.com/databricks/cli/pull/885)).
 * Fix template initialization from current working directory ([#976](https://github.com/databricks/cli/pull/976)).
 * Add `--tag` and `--branch` options to bundle init command ([#975](https://github.com/databricks/cli/pull/975)).
 * Work around DLT issue with `` not being set correctly ([#999](https://github.com/databricks/cli/pull/999)).
 * Enable `spark_jar_task` with local JAR libraries ([#993](https://github.com/databricks/cli/pull/993)).
 * Pass `USERPROFILE` environment variable to Terraform ([#1001](https://github.com/databricks/cli/pull/1001)).
 * Improve error message when path is not a bundle template ([#985](https://github.com/databricks/cli/pull/985)).
 * Correctly overwrite local state if remote state is newer ([#1008](https://github.com/databricks/cli/pull/1008)).
 * Add mlops-stacks to the default `databricks bundle init` prompt ([#988](https://github.com/databricks/cli/pull/988)).
 * Do not add wheel content hash in uploaded Python wheel path ([#1015](https://github.com/databricks/cli/pull/1015)).
 * Do not replace pipeline libraries if there are no matches for pattern ([#1021](https://github.com/databricks/cli/pull/1021)).

Internal:
 * Update CLI version in the VS Code extension during release ([#1014](https://github.com/databricks/cli/pull/1014)).

API Changes:
 * Changed `databricks functions create` command.
 * Changed `databricks metastores create` command with new required argument order.
 * Removed `databricks metastores enable-optimization` command.
 * Removed `databricks account o-auth-enrollment` command group.
 * Removed `databricks apps delete` command.
 * Removed `databricks apps get` command.
 * Added `databricks apps delete-app` command.
 * Added `databricks apps get-app` command.
 * Added `databricks apps get-app-deployment-status` command.
 * Added `databricks apps get-apps` command.
 * Added `databricks apps get-events` command.
 * Added `databricks account network-connectivity` command group.

OpenAPI commit 22f09783eb8a84d52026f856be3b2068f9498db3 (2023-11-23)

Dependency updates:
 * Bump golang.org/x/term from 0.13.0 to 0.14.0 ([#981](https://github.com/databricks/cli/pull/981)).
 * Bump github.com/hashicorp/terraform-json from 0.17.1 to 0.18.0 ([#979](https://github.com/databricks/cli/pull/979)).
 * Bump golang.org/x/oauth2 from 0.13.0 to 0.14.0 ([#982](https://github.com/databricks/cli/pull/982)).
 * Bump github.com/databricks/databricks-sdk-go from 0.24.0 to 0.25.0 ([#980](https://github.com/databricks/cli/pull/980)).
 * Bump github.com/databricks/databricks-sdk-go from 0.25.0 to 0.26.0 ([#1019](https://github.com/databricks/cli/pull/1019)).

## 0.209.1

CLI:
 * Hide `--progress-format` global flag ([#965](https://github.com/databricks/cli/pull/965)).
 * Make configure command visible + fix bundle command description ([#961](https://github.com/databricks/cli/pull/961)).
 * Log process ID in each log entry ([#949](https://github.com/databricks/cli/pull/949)).
 * Improve error message when `--json` flag is specified ([#933](https://github.com/databricks/cli/pull/933)).

Bundles:
 * Remove validation for default value against pattern ([#959](https://github.com/databricks/cli/pull/959)).
 * Bundle path rewrites for dbt and SQL file tasks ([#962](https://github.com/databricks/cli/pull/962)).
 * Initialize variable definitions that are defined without properties ([#966](https://github.com/databricks/cli/pull/966)).

Internal:
 * Function to merge two instances of `config.Value` ([#938](https://github.com/databricks/cli/pull/938)).
 * Make to/from string methods private to the jsonschema package ([#942](https://github.com/databricks/cli/pull/942)).
 * Make Cobra runner compatible with testing interactive flows ([#957](https://github.com/databricks/cli/pull/957)).
 * Added `env.UserHomeDir(ctx)` for parallel-friendly tests ([#955](https://github.com/databricks/cli/pull/955)).


Dependency updates:
 * Bump golang.org/x/mod from 0.13.0 to 0.14.0 ([#954](https://github.com/databricks/cli/pull/954)).
 * Bump golang.org/x/text from 0.13.0 to 0.14.0 ([#953](https://github.com/databricks/cli/pull/953)).
 * Bump golang.org/x/sync from 0.4.0 to 0.5.0 ([#951](https://github.com/databricks/cli/pull/951)).
 * Bump github.com/spf13/cobra from 1.7.0 to 1.8.0 ([#950](https://github.com/databricks/cli/pull/950)).
 * Bump github.com/fatih/color from 1.15.0 to 1.16.0 ([#952](https://github.com/databricks/cli/pull/952)).

## 0.209.0

CLI:
 * Added GitHub issue templates for CLI and DABs issues ([#925](https://github.com/databricks/cli/pull/925)).
 * Simplified code generation logic for handling path and request body parameters and JSON input ([#905](https://github.com/databricks/cli/pull/905)).


Bundles:
 * Fixed URL for bundle template documentation in init command help docs ([#903](https://github.com/databricks/cli/pull/903)).
 * Fixed pattern validation for input parameters in a bundle template ([#912](https://github.com/databricks/cli/pull/912)).
 * Fixed multiline description rendering for enum input parameters in bundle templates ([#916](https://github.com/databricks/cli/pull/916)).
 * Changed production mode check for whether identity used is a service principal to use UserName  ([#924](https://github.com/databricks/cli/pull/924)).
 * Changed bundle deploy to upload partial terraform state even if deployment fails ([#923](https://github.com/databricks/cli/pull/923)).
 * Added support for welcome messages to bundle templates ([#907](https://github.com/databricks/cli/pull/907)).
 * Added support for uploading bundle deployment metadata to WSFS ([#845](https://github.com/databricks/cli/pull/845)).


Internal:
 * Loading an empty yaml file yields a nil ([#906](https://github.com/databricks/cli/pull/906)).
 * Library to convert config.Value to Go struct ([#904](https://github.com/databricks/cli/pull/904)).
 * Remove default resolution of repo names against the Databricks Github account([#940](https://github.com/databricks/cli/pull/940)).
 * Run make fmt from fmt job ([#929](https://github.com/databricks/cli/pull/929)).
 * `make snapshot` to build file in `.databricks/databricks` ([#927](https://github.com/databricks/cli/pull/927)).
 * Add configuration normalization code ([#915](https://github.com/databricks/cli/pull/915)).

API Changes:
 * Added `databricks account network-policy` command group.

Dependency updates:
 * Bump Terraform provider from v1.28.0 to v1.29.0 ([#926](https://github.com/databricks/cli/pull/926)).
 * Bump the Go SDK in the CLI from v0.23 to v0.24 ([#919](https://github.com/databricks/cli/pull/919)).
 * Bump google.golang.org/grpc from 1.58.2 to 1.58.3 ([#920](https://github.com/databricks/cli/pull/920)).
 * Bump github.com/google/uuid from 1.3.1 to 1.4.0 ([#932](https://github.com/databricks/cli/pull/932)).

OpenAPI commit 5903bb39137fd76ac384b2044e425f9c56840e00 (2023-10-23)

## 0.208.2

CLI:
 * Never load authentication configuration from bundle for sync command ([#889](https://github.com/databricks/cli/pull/889)).
 * Fixed requiring positional arguments for API URL parameters ([#878](https://github.com/databricks/cli/pull/878)).

Bundles:
 * Add support for validating CLI version when loading a jsonschema object ([#883](https://github.com/databricks/cli/pull/883)).
 * Do not emit wheel wrapper error when python_wheel_wrapper setting is true ([#894](https://github.com/databricks/cli/pull/894)).
 * Resolve configuration before performing verification ([#890](https://github.com/databricks/cli/pull/890)).
 * Fix wheel task not working with with 13.x clusters ([#898](https://github.com/databricks/cli/pull/898)).

Internal:
 * Skip prompt on completion hook ([#888](https://github.com/databricks/cli/pull/888)).
 * New YAML loader to support configuration location ([#828](https://github.com/databricks/cli/pull/828)).

Dependency updates:
 * Bump github.com/mattn/go-isatty from 0.0.19 to 0.0.20 ([#896](https://github.com/databricks/cli/pull/896)).

## 0.208.1

CLI:
 * Fix rendering of streaming response ([#876](https://github.com/databricks/cli/pull/876)).

Bundles:
 * Rename MLOps Stack to MLOps Stacks ([#881](https://github.com/databricks/cli/pull/881)).
 * Support Python wheels larger than 10MB ([#879](https://github.com/databricks/cli/pull/879)).
 * Improve the output of the `databricks bundle init` command ([#795](https://github.com/databricks/cli/pull/795)).



## 0.208.0

Note: this release includes a fix for the issue where zero values (for example
`num_workers: 0`) were not included in the request body.

CLI:
 * Use already instantiated WorkspaceClient in sync command ([#867](https://github.com/databricks/cli/pull/867)).

Bundles:
 * Support Unity Catalog Registered Models in bundles ([#846](https://github.com/databricks/cli/pull/846)).
 * Fixed merging task libraries from targets ([#868](https://github.com/databricks/cli/pull/868)).
 * Add alias for mlops-stack template URL ([#869](https://github.com/databricks/cli/pull/869)).

API Changes:
 * Changed `databricks account billable-usage download` command to start returning output.
 * Changed `databricks account storage-credentials delete` command with new required argument order.
 * Changed `databricks account storage-credentials get` command with new required argument order.
 * Changed `databricks account storage-credentials update` command with new required argument order.
 * Added `databricks workspace-bindings get-bindings` command.
 * Added `databricks workspace-bindings update-bindings` command.
 * Removed `databricks account network-policy` command group.
 * Changed `databricks ip-access-lists list` command to return output.

OpenAPI commit 493a76554afd3afdd15dc858773d01643f80352a (2023-10-12)

Dependency updates:
 * Update Go SDK to 0.23.0 and use custom marshaller ([#772](https://github.com/databricks/cli/pull/772)).
 * Bump Terraform provider to v1.28.0 ([#871](https://github.com/databricks/cli/pull/871)).
 * Bump golang.org/x/net from 0.16.0 to 0.17.0 ([#863](https://github.com/databricks/cli/pull/863)).
 * Bump github.com/hashicorp/hc-install from 0.6.0 to 0.6.1 ([#870](https://github.com/databricks/cli/pull/870)).

## 0.207.1

CLI:
 * Improve `workspace import` command by allowing references to local files for content ([#793](https://github.com/databricks/cli/pull/793)).
 * Add `--file` flag to workspace export command ([#794](https://github.com/databricks/cli/pull/794)).
 * Ensure profile flag is respected for sync command ([#837](https://github.com/databricks/cli/pull/837)).
 * Add hint to delete sync snapshot if parsing fails ([#853](https://github.com/databricks/cli/pull/853)).
 * Use profile information when getting a token using the CLI ([#855](https://github.com/databricks/cli/pull/855)).

Bundles:
 * Minor template tweaks ([#832](https://github.com/databricks/cli/pull/832)).
 * Fixed using repo files as pipeline libraries ([#847](https://github.com/databricks/cli/pull/847)).
 * Support .gitignore syntax in sync section and make sure it works recursively ([#854](https://github.com/databricks/cli/pull/854)).
 * Allow target overrides for sync section ([#856](https://github.com/databricks/cli/pull/856)).

Internal:
 * Fix import export integration tests on windows ([#842](https://github.com/databricks/cli/pull/842)).
 * Fix workspace import test ([#844](https://github.com/databricks/cli/pull/844)).
 * Automatically create a release PR in homebrew-tap repo ([#841](https://github.com/databricks/cli/pull/841)).


Dependency updates:
 * Bump golang.org/x/term from 0.12.0 to 0.13.0 ([#852](https://github.com/databricks/cli/pull/852)).
 * Bump golang.org/x/mod from 0.12.0 to 0.13.0 ([#851](https://github.com/databricks/cli/pull/851)).
 * Bump golang.org/x/sync from 0.3.0 to 0.4.0 ([#849](https://github.com/databricks/cli/pull/849)).
 * Bump golang.org/x/oauth2 from 0.12.0 to 0.13.0 ([#850](https://github.com/databricks/cli/pull/850)).

## 0.207.0

CLI:
 * Refactor change computation for sync ([#785](https://github.com/databricks/cli/pull/785)).

Bundles:
 * Allow digits in the generated short name ([#820](https://github.com/databricks/cli/pull/820)).
 * Emit an error when incompatible all purpose cluster used with Python wheel tasks ([#823](https://github.com/databricks/cli/pull/823)).
 * Use normalized short name for tag value in development mode ([#821](https://github.com/databricks/cli/pull/821)).
 * Added `python.DetectInterpreters` and other utils ([#805](https://github.com/databricks/cli/pull/805)).
 * Mark artifacts properties as optional ([#834](https://github.com/databricks/cli/pull/834)).
 * Added support for glob patterns in pipeline libraries section ([#833](https://github.com/databricks/cli/pull/833)).

Internal:
 * Run tests to verify backend tag validation behavior ([#814](https://github.com/databricks/cli/pull/814)).
 * Library to validate and normalize cloud specific tags ([#819](https://github.com/databricks/cli/pull/819)).
 * Added test to submit and run various Python tasks on multiple DBR versions ([#806](https://github.com/databricks/cli/pull/806)).
 * Create a release PR in setup-cli repo on tag push ([#827](https://github.com/databricks/cli/pull/827)).

API Changes:
 * Changed `databricks account metastore-assignments list` command to return .
 * Changed `databricks jobs cancel-all-runs` command with new required argument order.
 * Added `databricks account o-auth-published-apps` command group.
 * Changed `databricks serving-endpoints query` command . New request type is .
 * Added `databricks serving-endpoints patch` command.
 * Added `databricks credentials-manager` command group.
 * Added `databricks settings` command group.
 * Changed `databricks clean-rooms list` command to require request of .
 * Changed `databricks statement-execution execute-statement` command with new required argument order.

OpenAPI commit bcbf6e851e3d82fd910940910dd31c10c059746c (2023-10-02)
Dependency updates:
 * Bump github.com/google/uuid from 1.3.0 to 1.3.1 ([#825](https://github.com/databricks/cli/pull/825)).
 * Updated Go SDK to 0.22.0 ([#831](https://github.com/databricks/cli/pull/831)).

## 0.206.0

Bundles:
 * Enable target overrides for pipeline clusters ([#792](https://github.com/databricks/cli/pull/792)).
 * Add support for regex patterns in template schema ([#768](https://github.com/databricks/cli/pull/768)).
 * Make the default `databricks bundle init` template more self-explanatory ([#796](https://github.com/databricks/cli/pull/796)).
 * Make a notebook wrapper for Python wheel tasks optional ([#797](https://github.com/databricks/cli/pull/797)).
 * Added a warning when Python wheel wrapper needs to be used ([#807](https://github.com/databricks/cli/pull/807)).

Internal:
 * Added `process.Background()` and `process.Forwarded()` ([#804](https://github.com/databricks/cli/pull/804)).

Dependency updates:
 * Bump golang.org/x/term from 0.11.0 to 0.12.0 ([#798](https://github.com/databricks/cli/pull/798)).
 * Bump github.com/hashicorp/terraform-exec from 0.18.1 to 0.19.0 ([#801](https://github.com/databricks/cli/pull/801)).
 * Bump golang.org/x/oauth2 from 0.11.0 to 0.12.0 ([#802](https://github.com/databricks/cli/pull/802)).

## 0.205.2

CLI:
 * Prompt for profile only in interactive mode ([#788](https://github.com/databricks/cli/pull/788)).

Internal:
 * Added setup Python action ([#789](https://github.com/databricks/cli/pull/789)).


## 0.205.1

Bundles:
 * Use enums for default python template ([#765](https://github.com/databricks/cli/pull/765)).
 * Make bundle deploy work if no resources are defined ([#767](https://github.com/databricks/cli/pull/767)).
 * Added support for experimental scripts section ([#632](https://github.com/databricks/cli/pull/632)).
 * Error when unknown keys are encounters during template execution ([#766](https://github.com/databricks/cli/pull/766)).
 * Fall back to full Git clone if shallow clone is not supported ([#775](https://github.com/databricks/cli/pull/775)).
 * Enable environment overrides for job tasks ([#779](https://github.com/databricks/cli/pull/779)).
 * Increase timeout waiting for job run to 1 day ([#786](https://github.com/databricks/cli/pull/786)).

Internal:
 * Update Go SDK to v0.19.3 (unreleased) ([#778](https://github.com/databricks/cli/pull/778)).



## 0.205.0

This release marks the public preview phase of Databricks Asset Bundles.

For more information, please refer to our online documentation at
https://docs.databricks.com/en/dev-tools/bundles/.

CLI:
 * Prompt once for a client profile ([#727](https://github.com/databricks/cli/pull/727)).

Bundles:
 * Use clearer error message when no interpolation value is found. ([#764](https://github.com/databricks/cli/pull/764)).
 * Use interactive prompt to select resource to run if not specified ([#762](https://github.com/databricks/cli/pull/762)).
 * Add documentation link bundle command group description ([#770](https://github.com/databricks/cli/pull/770)).


## 0.204.1

Bundles:
 * Fix conversion of job parameters ([#744](https://github.com/databricks/cli/pull/744)).
 * Add schema and config validation to jsonschema package ([#740](https://github.com/databricks/cli/pull/740)).
 * Support Model Serving Endpoints in bundles ([#682](https://github.com/databricks/cli/pull/682)).
 * Do not include empty output in job run output ([#749](https://github.com/databricks/cli/pull/749)).
 * Fixed marking libraries from DBFS as remote ([#750](https://github.com/databricks/cli/pull/750)).
 * Process only Python wheel tasks which have local libraries used ([#751](https://github.com/databricks/cli/pull/751)).
 * Add enum support for bundle templates ([#668](https://github.com/databricks/cli/pull/668)).
 * Apply Python wheel trampoline if workspace library is used ([#755](https://github.com/databricks/cli/pull/755)).
 * List available targets when incorrect target passed ([#756](https://github.com/databricks/cli/pull/756)).
 * Make bundle and sync fields optional ([#757](https://github.com/databricks/cli/pull/757)).
 * Consolidate environment variable interaction ([#747](https://github.com/databricks/cli/pull/747)).

Internal:
 * Update Go SDK to v0.19.1 ([#759](https://github.com/databricks/cli/pull/759)).



## 0.204.0

This release includes permission related commands for a subset of workspace
services where they apply. These complement the `permissions` command and
do not require specification of the object type to work with, as that is
implied by the command they are nested under.

CLI:
 * Group permission related commands ([#730](https://github.com/databricks/cli/pull/730)).

Bundles:
 * Fixed artifact file uploading on Windows and wheel execution on DBR 13.3 ([#722](https://github.com/databricks/cli/pull/722)).
 * Make resource and artifact paths in bundle config relative to config folder ([#708](https://github.com/databricks/cli/pull/708)).
 * Add support for ordering of input prompts ([#662](https://github.com/databricks/cli/pull/662)).
 * Fix IsServicePrincipal() only working for workspace admins ([#732](https://github.com/databricks/cli/pull/732)).
 * databricks bundle init template v1 ([#686](https://github.com/databricks/cli/pull/686)).
 * databricks bundle init template v2: optional stubs, DLT support ([#700](https://github.com/databricks/cli/pull/700)).
 * Show 'databricks bundle init' template in CLI prompt ([#725](https://github.com/databricks/cli/pull/725)).
 * Include  in set of environment variables to pass along. ([#736](https://github.com/databricks/cli/pull/736)).

Internal:
 * Update Go SDK to v0.19.0 ([#729](https://github.com/databricks/cli/pull/729)).
 * Replace API call to test configuration with dummy authenticate call ([#728](https://github.com/databricks/cli/pull/728)).

API Changes:
 * Changed `databricks account storage-credentials create` command to return .
 * Changed `databricks account storage-credentials get` command to return .
 * Changed `databricks account storage-credentials list` command to return .
 * Changed `databricks account storage-credentials update` command to return .
 * Changed `databricks connections create` command with new required argument order.
 * Changed `databricks connections update` command with new required argument order.
 * Changed `databricks volumes create` command with new required argument order.
 * Added `databricks artifact-allowlists` command group.
 * Added `databricks model-versions` command group.
 * Added `databricks registered-models` command group.
 * Added `databricks cluster-policies get-permission-levels` command.
 * Added `databricks cluster-policies get-permissions` command.
 * Added `databricks cluster-policies set-permissions` command.
 * Added `databricks cluster-policies update-permissions` command.
 * Added `databricks clusters get-permission-levels` command.
 * Added `databricks clusters get-permissions` command.
 * Added `databricks clusters set-permissions` command.
 * Added `databricks clusters update-permissions` command.
 * Added `databricks instance-pools get-permission-levels` command.
 * Added `databricks instance-pools get-permissions` command.
 * Added `databricks instance-pools set-permissions` command.
 * Added `databricks instance-pools update-permissions` command.
 * Added `databricks files` command group.
 * Changed `databricks permissions set` command to start returning .
 * Changed `databricks permissions update` command to start returning .
 * Added `databricks users get-permission-levels` command.
 * Added `databricks users get-permissions` command.
 * Added `databricks users set-permissions` command.
 * Added `databricks users update-permissions` command.
 * Added `databricks jobs get-permission-levels` command.
 * Added `databricks jobs get-permissions` command.
 * Added `databricks jobs set-permissions` command.
 * Added `databricks jobs update-permissions` command.
 * Changed `databricks experiments get-by-name` command to return .
 * Changed `databricks experiments get-experiment` command to return .
 * Added `databricks experiments delete-runs` command.
 * Added `databricks experiments get-permission-levels` command.
 * Added `databricks experiments get-permissions` command.
 * Added `databricks experiments restore-runs` command.
 * Added `databricks experiments set-permissions` command.
 * Added `databricks experiments update-permissions` command.
 * Added `databricks model-registry get-permission-levels` command.
 * Added `databricks model-registry get-permissions` command.
 * Added `databricks model-registry set-permissions` command.
 * Added `databricks model-registry update-permissions` command.
 * Added `databricks pipelines get-permission-levels` command.
 * Added `databricks pipelines get-permissions` command.
 * Added `databricks pipelines set-permissions` command.
 * Added `databricks pipelines update-permissions` command.
 * Added `databricks serving-endpoints get-permission-levels` command.
 * Added `databricks serving-endpoints get-permissions` command.
 * Added `databricks serving-endpoints set-permissions` command.
 * Added `databricks serving-endpoints update-permissions` command.
 * Added `databricks token-management get-permission-levels` command.
 * Added `databricks token-management get-permissions` command.
 * Added `databricks token-management set-permissions` command.
 * Added `databricks token-management update-permissions` command.
 * Changed `databricks dashboards create` command with new required argument order.
 * Added `databricks warehouses get-permission-levels` command.
 * Added `databricks warehouses get-permissions` command.
 * Added `databricks warehouses set-permissions` command.
 * Added `databricks warehouses update-permissions` command.
 * Added `databricks dashboard-widgets` command group.
 * Added `databricks query-visualizations` command group.
 * Added `databricks repos get-permission-levels` command.
 * Added `databricks repos get-permissions` command.
 * Added `databricks repos set-permissions` command.
 * Added `databricks repos update-permissions` command.
 * Added `databricks secrets get-secret` command.
 * Added `databricks workspace get-permission-levels` command.
 * Added `databricks workspace get-permissions` command.
 * Added `databricks workspace set-permissions` command.
 * Added `databricks workspace update-permissions` command.

OpenAPI commit 09a7fa63d9ae243e5407941f200960ca14d48b07 (2023-09-04)

## 0.203.3

Bundles:
 * Support cluster overrides with cluster_key and compute_key ([#696](https://github.com/databricks/cli/pull/696)).
 * Allow referencing local Python wheels without artifacts section defined ([#703](https://github.com/databricks/cli/pull/703)).
 * Fixed --environment flag ([#705](https://github.com/databricks/cli/pull/705)).
 * Correctly identify local paths in libraries section ([#702](https://github.com/databricks/cli/pull/702)).
 * Fixed path joining in FindFilesWithSuffixInPath ([#704](https://github.com/databricks/cli/pull/704)).
 *  Added transformation mutator for Python wheel task for them to work on DBR <13.1 ([#635](https://github.com/databricks/cli/pull/635)).

Internal:
 * Add a foundation for built-in templates ([#685](https://github.com/databricks/cli/pull/685)).
 * Test transform when no Python wheel tasks defined ([#714](https://github.com/databricks/cli/pull/714)).
 * Pin Terraform binary version to 1.5.5 ([#715](https://github.com/databricks/cli/pull/715)).
 * Cleanup after "Add a foundation for built-in templates" ([#707](https://github.com/databricks/cli/pull/707)).
 * Filter down to Python wheel tasks only for trampoline ([#712](https://github.com/databricks/cli/pull/712)).
 * Update Terraform provider schema structs from 1.23.0 ([#713](https://github.com/databricks/cli/pull/713)).

## 0.203.2

CLI:
 * Added `databricks account o-auth-enrollment enable` command ([#687](https://github.com/databricks/cli/pull/687)).

Bundles:
 * Do not try auto detect Python package if no Python wheel tasks defined ([#674](https://github.com/databricks/cli/pull/674)).
 * Renamed `environments` to `targets` in bundle configuration ([#670](https://github.com/databricks/cli/pull/670)).
 * Rename init project-dir flag to output-dir ([#676](https://github.com/databricks/cli/pull/676)).
 * Added support for sync.include and sync.exclude sections ([#671](https://github.com/databricks/cli/pull/671)).
 * Add template directory flag for bundle templates ([#675](https://github.com/databricks/cli/pull/675)).
 * Never ignore root directory when enumerating files in a repository ([#683](https://github.com/databricks/cli/pull/683)).
 * Improve 'mode' error message ([#681](https://github.com/databricks/cli/pull/681)).
 * Added run_as section for bundle configuration ([#692](https://github.com/databricks/cli/pull/692)).

## 0.203.1

CLI:
 * Always resolve .databrickscfg file ([#659](https://github.com/databricks/cli/pull/659)).

Bundles:
 * Add internal tag for bundle fields to be skipped from schema ([#636](https://github.com/databricks/cli/pull/636)).
 * Log the bundle root configuration file if applicable ([#657](https://github.com/databricks/cli/pull/657)).
 * Execute paths without the .tmpl extension as templates ([#654](https://github.com/databricks/cli/pull/654)).
 * Enable environment overrides for job clusters ([#658](https://github.com/databricks/cli/pull/658)).
 * Merge artifacts and resources block with overrides enabled ([#660](https://github.com/databricks/cli/pull/660)).
 * Locked terraform binary version to <= 1.5.5 ([#666](https://github.com/databricks/cli/pull/666)).
 * Return better error messages for invalid JSON schema types in templates ([#661](https://github.com/databricks/cli/pull/661)).
 * Use custom prompter for bundle template inputs ([#663](https://github.com/databricks/cli/pull/663)).
 * Add map and pair helper functions for bundle templates ([#665](https://github.com/databricks/cli/pull/665)).
 * Correct name for force acquire deploy flag ([#656](https://github.com/databricks/cli/pull/656)).
 * Confirm that override with a zero value doesn't work ([#669](https://github.com/databricks/cli/pull/669)).

Internal:
 * Consolidate functions in libs/git ([#652](https://github.com/databricks/cli/pull/652)).
 * Upgraded Go version to 1.21 ([#664](https://github.com/databricks/cli/pull/664)).

## 0.203.0

CLI:
 * Infer host from profile during `auth login` ([#629](https://github.com/databricks/cli/pull/629)).

Bundles:
 * Extend deployment mode support ([#577](https://github.com/databricks/cli/pull/577)).
 * Add validation for Git settings in bundles ([#578](https://github.com/databricks/cli/pull/578)).
 * Only treat files with .tmpl extension as templates ([#594](https://github.com/databricks/cli/pull/594)).
 * Add JSON schema validation for input template parameters ([#598](https://github.com/databricks/cli/pull/598)).
 * Add DATABRICKS_BUNDLE_INCLUDE_PATHS to specify include paths through env vars ([#591](https://github.com/databricks/cli/pull/591)).
 * Initialise a empty default bundle if BUNDLE_ROOT and DATABRICKS_BUNDLE_INCLUDES env vars are present ([#604](https://github.com/databricks/cli/pull/604)).
 * Regenerate bundle resource structs from latest Terraform provider ([#633](https://github.com/databricks/cli/pull/633)).
 * Fixed processing jobs libraries with remote path ([#638](https://github.com/databricks/cli/pull/638)).
 * Add unit test for file name execution during rendering ([#640](https://github.com/databricks/cli/pull/640)).
 * Add bundle init command and support for prompting user for input values ([#631](https://github.com/databricks/cli/pull/631)).
 * Fix bundle git branch validation ([#645](https://github.com/databricks/cli/pull/645)).

Internal:
 * Fix mkdir integration test on GCP ([#620](https://github.com/databricks/cli/pull/620)).
 * Fix git clone integration test for non-existing repo ([#610](https://github.com/databricks/cli/pull/610)).
 * Remove push to main trigger for build workflow ([#621](https://github.com/databricks/cli/pull/621)).
 * Remove workflow to publish binaries to S3 ([#622](https://github.com/databricks/cli/pull/622)).
 * Fix failing fs mkdir test on azure ([#627](https://github.com/databricks/cli/pull/627)).
 * Print y/n options when displaying prompts using cmdio.Ask ([#650](https://github.com/databricks/cli/pull/650)).

API Changes:
 * Changed `databricks account metastore-assignments create` command to not return anything.
 * Added `databricks account network-policy` command group.

OpenAPI commit 7b57ba3a53f4de3d049b6a24391fe5474212daf8 (2023-07-28)

Dependency updates:
 * Bump OpenAPI specification & Go SDK Version ([#624](https://github.com/databricks/cli/pull/624)).
 * Bump golang.org/x/term from 0.10.0 to 0.11.0 ([#643](https://github.com/databricks/cli/pull/643)).
 * Bump golang.org/x/text from 0.11.0 to 0.12.0 ([#642](https://github.com/databricks/cli/pull/642)).
 * Bump golang.org/x/oauth2 from 0.10.0 to 0.11.0 ([#641](https://github.com/databricks/cli/pull/641)).

## 0.202.0

Breaking Change:
 * Require include glob patterns to be explicitly defined ([#602](https://github.com/databricks/cli/pull/602)).

Bundles:
 * Add support for more SDK config options ([#587](https://github.com/databricks/cli/pull/587)).
 * Add template renderer for Databricks templates ([#589](https://github.com/databricks/cli/pull/589)).
 * Fix formatting in renderer.go ([#593](https://github.com/databricks/cli/pull/593)).
 * Fixed python wheel test ([#608](https://github.com/databricks/cli/pull/608)).
 * Auto detect Python wheel packages and infer build command ([#603](https://github.com/databricks/cli/pull/603)).
 * Added support for artifacts building for bundles ([#583](https://github.com/databricks/cli/pull/583)).
 * Add support for cloning repositories ([#544](https://github.com/databricks/cli/pull/544)).
 * Add regexp compile helper function for templates ([#601](https://github.com/databricks/cli/pull/601)).
 * Add unit test that raw strings are printed as is ([#599](https://github.com/databricks/cli/pull/599)).

Internal:
 * Fix tests under ./cmd/configure if DATABRICKS_TOKEN is set ([#605](https://github.com/databricks/cli/pull/605)).
 * Remove dependency on global state in generated commands ([#595](https://github.com/databricks/cli/pull/595)).
 * Remove dependency on global state for the root command ([#606](https://github.com/databricks/cli/pull/606)).
 * Add merge_group trigger for build ([#612](https://github.com/databricks/cli/pull/612)).
 * Added support for build command chaining and error on missing wheel ([#607](https://github.com/databricks/cli/pull/607)).
 * Add TestAcc prefix to filer test and fix any failing tests ([#611](https://github.com/databricks/cli/pull/611)).
 * Add url parse helper function for templates ([#600](https://github.com/databricks/cli/pull/600)).
 * Remove dependency on global state for remaining commands ([#613](https://github.com/databricks/cli/pull/613)).
 * Update CHANGELOG template ([#588](https://github.com/databricks/cli/pull/588)).



## 0.201.0

CLI:
 * Support tab completion for profiles ([#572](https://github.com/databricks/cli/pull/572)).
 * Improve auth login experience ([#570](https://github.com/databricks/cli/pull/570)).
 * Integrate with auto-release infra ([#581](https://github.com/databricks/cli/pull/581)).

Bundles:
 * Add development runs ([#522](https://github.com/databricks/cli/pull/522)).
 * Correctly use --profile flag passed for all bundle commands ([#571](https://github.com/databricks/cli/pull/571)).
 * Disallow notebooks in paths where files are expected ([#573](https://github.com/databricks/cli/pull/573)).
 * Remove base path checks during sync ([#576](https://github.com/databricks/cli/pull/576)).
 * First look for databricks.yml before falling back to bundle.yml ([#580](https://github.com/databricks/cli/pull/580)).

API Changes:
 * Removed `databricks metastores maintenance` command.
 * Added `databricks metastores enable-optimization` command.
 * Added `databricks tables update` command.
 * Changed `databricks account settings delete-personal-compute-setting` command with new required argument order.
 * Changed `databricks account settings read-personal-compute-setting` command with new required argument order.
 * Added `databricks clean-rooms` command group.

OpenAPI commit 850a075ed9758d21a6bc4409506b48c8b9f93ab4 (2023-07-18)

Dependency updates:
 * Bump golang.org/x/term from 0.9.0 to 0.10.0 ([#567](https://github.com/databricks/cli/pull/567)).
 * Bump golang.org/x/oauth2 from 0.9.0 to 0.10.0 ([#566](https://github.com/databricks/cli/pull/566)).
 * Bump golang.org/x/mod from 0.11.0 to 0.12.0 ([#568](https://github.com/databricks/cli/pull/568)).
 * Bump github.com/databricks/databricks-sdk-go from 0.12.0 to 0.13.0 ([#585](https://github.com/databricks/cli/pull/585)).

## 0.200.2

CLI:
* Fix secrets put-secret command ([#545](https://github.com/databricks/cli/pull/545)).
* Fixed ignoring required positional parameters when --json flag is provided ([#535](https://github.com/databricks/cli/pull/535)).
* Update cp help message to not require file scheme ([#554](https://github.com/databricks/cli/pull/554)).

Bundles:
* Fix: bundle destroy fails when bundle.tf.json file is deleted ([#519](https://github.com/databricks/cli/pull/519)).
* Fixed error reporting when included invalid files in include section ([#543](https://github.com/databricks/cli/pull/543)).
* Make top level workspace optional in JSON schema ([#562](https://github.com/databricks/cli/pull/562)).
* Propagate TF_CLI_CONFIG_FILE env variable ([#555](https://github.com/databricks/cli/pull/555)).
* Update Terraform provider schema structs ([#563](https://github.com/databricks/cli/pull/563)).
* Update inline JSON schema documentation ([#557](https://github.com/databricks/cli/pull/557)).

Dependencies:
* Bump Go SDK to v0.12.0 ([#540](https://github.com/databricks/cli/pull/540)).
* Bump github.com/hashicorp/terraform-json from 0.17.0 to 0.17.1 ([#541](https://github.com/databricks/cli/pull/541)).

## 0.200.1

CLI:
* Add --absolute flag for ls command ([#508](https://github.com/databricks/cli/pull/508)).
* Add dbfs scheme prefix to paths in cp command output ([#516](https://github.com/databricks/cli/pull/516)).
* Add provider detection to the repos create command ([#528](https://github.com/databricks/cli/pull/528)).
* Added configure-cluster flag for auth login ([#500](https://github.com/databricks/cli/pull/500)).
* Added prompts for Databricks profile for auth login command ([#502](https://github.com/databricks/cli/pull/502)).
* Allow specifying repo by path for repos commands ([#526](https://github.com/databricks/cli/pull/526)).
* Decode contents by default in workspace export command ([#531](https://github.com/databricks/cli/pull/531)).
* Fixed jobs create command to only accept JSON payload ([#498](https://github.com/databricks/cli/pull/498)).
* Make local files default for fs commands ([#506](https://github.com/databricks/cli/pull/506)).
* Remove \r from new line print statments ([#509](https://github.com/databricks/cli/pull/509)).
* Remove extra call to filer.Stat in dbfs filer.Read ([#515](https://github.com/databricks/cli/pull/515)).
* Update alerts command integration test ([#512](https://github.com/databricks/cli/pull/512)).
* Update variable regex to support hyphens ([#503](https://github.com/databricks/cli/pull/503)).

Bundles:
* Add DATABRICKS_BUNDLE_TMP env variable ([#462](https://github.com/databricks/cli/pull/462)).
* Update Terraform provider schema structs ([#504](https://github.com/databricks/cli/pull/504)).

Dependencies:
* Bump github.com/databricks/databricks-sdk-go from 0.9.1-0.20230614092458-b5bbc1c8dabb to 0.10.0 ([#497](https://github.com/databricks/cli/pull/497)).

Internal:
* Use direct download for workspace filer read ([#514](https://github.com/databricks/cli/pull/514)).

## 0.200.0

This version marks the first version available as public preview.

The minor bump to 200 better disambiguates between Databricks CLI "v1" (the Python version)
and this version, Databricks CLI "v2". The minor version of 0.100 may look lower than 0.17
to some, whereas 200 does not. This bump has no other significance.

CLI:
 * Add filer.Filer implementation backed by the Files API ([#474](https://github.com/databricks/cli/pull/474)).
 * Add fs cp command ([#463](https://github.com/databricks/cli/pull/463)).
 * Correctly set ExactArgs if generated command has positional arguments ([#488](https://github.com/databricks/cli/pull/488)).
 * Do not use white color as string output ([#489](https://github.com/databricks/cli/pull/489)).
 * Update README to reflect public preview status ([#491](https://github.com/databricks/cli/pull/491)).

Bundles:
 * Fix force flag not working for bundle destroy ([#434](https://github.com/databricks/cli/pull/434)).
 * Fix locker unlock for destroy ([#492](https://github.com/databricks/cli/pull/492)).
 * Use better error assertions and clean up locker API ([#490](https://github.com/databricks/cli/pull/490)).

Dependencies:
 * Bump golang.org/x/mod from 0.10.0 to 0.11.0 ([#496](https://github.com/databricks/cli/pull/496)).
 * Bump golang.org/x/sync from 0.2.0 to 0.3.0 ([#495](https://github.com/databricks/cli/pull/495)).

## 0.100.4

CLI:
 * Add workspace import-dir command ([#456](https://github.com/databricks/cli/pull/456)).
 * Annotate generated commands with OpenAPI package name ([#466](https://github.com/databricks/cli/pull/466)).
 * Associate generated commands with command groups ([#475](https://github.com/databricks/cli/pull/475)).
 * Disable shell completions for generated commands ([#483](https://github.com/databricks/cli/pull/483)).
 * Include [DEFAULT] section header when writing ~/.databrickscfg ([#464](https://github.com/databricks/cli/pull/464)).
 * Pass through proxy related environment variables ([#465](https://github.com/databricks/cli/pull/465)).
 * Restore flags to original values on test completion ([#470](https://github.com/databricks/cli/pull/470)).
 * Update configure command ([#482](https://github.com/databricks/cli/pull/482)).

Dependencies:
 * Bump SDK to latest ([#473](https://github.com/databricks/cli/pull/473)).

## 0.100.3

CLI:
 * Add directory tracking to sync ([#425](https://github.com/databricks/cli/pull/425)).
 * Add fs cat command for dbfs files ([#430](https://github.com/databricks/cli/pull/430)).
 * Add fs ls command for dbfs ([#429](https://github.com/databricks/cli/pull/429)).
 * Add fs mkdirs command for dbfs ([#432](https://github.com/databricks/cli/pull/432)).
 * Add fs rm command for dbfs ([#433](https://github.com/databricks/cli/pull/433)).
 * Add installation instructions ([#458](https://github.com/databricks/cli/pull/458)).
 * Add new line to cmdio JSON rendering ([#443](https://github.com/databricks/cli/pull/443)).
 * Add profile on `databricks auth login` ([#423](https://github.com/databricks/cli/pull/423)).
 * Add readable console logger ([#370](https://github.com/databricks/cli/pull/370)).
 * Add workspace export-dir command ([#449](https://github.com/databricks/cli/pull/449)).
 * Added secrets input prompt for secrets put-secret command ([#413](https://github.com/databricks/cli/pull/413)).
 * Added spinner when loading command prompts ([#420](https://github.com/databricks/cli/pull/420)).
 * Better error message if can not load prompts ([#437](https://github.com/databricks/cli/pull/437)).
 * Changed service template to correctly handle required positional arguments ([#405](https://github.com/databricks/cli/pull/405)).
 * Do not generate prompts for certain commands ([#438](https://github.com/databricks/cli/pull/438)).
 * Do not prompt for List methods ([#411](https://github.com/databricks/cli/pull/411)).
 * Do not use FgWhite and FgBlack for terminal output ([#435](https://github.com/databricks/cli/pull/435)).
 * Skip path translation of job task for jobs with a Git source ([#404](https://github.com/databricks/cli/pull/404)).
 * Tweak profile prompt ([#454](https://github.com/databricks/cli/pull/454)).
 * Update with the latest Go SDK ([#457](https://github.com/databricks/cli/pull/457)).
 * Use cmdio in version command for `--output` flag ([#419](https://github.com/databricks/cli/pull/419)).

Bundles:
 * Check for nil environment before accessing it ([#453](https://github.com/databricks/cli/pull/453)).

Dependencies:
 * Bump github.com/hashicorp/terraform-json from 0.16.0 to 0.17.0 ([#459](https://github.com/databricks/cli/pull/459)).
 * Bump github.com/mattn/go-isatty from 0.0.18 to 0.0.19 ([#412](https://github.com/databricks/cli/pull/412)).

Internal:
 * Add Mkdir and ReadDir functions to filer.Filer interface ([#414](https://github.com/databricks/cli/pull/414)).
 * Add Stat function to filer.Filer interface ([#421](https://github.com/databricks/cli/pull/421)).
 * Add check for path is a directory in filer.ReadDir ([#426](https://github.com/databricks/cli/pull/426)).
 * Add fs.FS adapter for the filer interface ([#422](https://github.com/databricks/cli/pull/422)).
 * Add implementation of filer.Filer for local filesystem ([#460](https://github.com/databricks/cli/pull/460)).
 * Allow equivalence checking of filer errors to fs errors ([#416](https://github.com/databricks/cli/pull/416)).
 * Fix locker integration test ([#417](https://github.com/databricks/cli/pull/417)).
 * Implement DBFS filer ([#139](https://github.com/databricks/cli/pull/139)).
 * Include recursive deletion in filer interface ([#442](https://github.com/databricks/cli/pull/442)).
 * Make filer.Filer return fs.DirEntry from ReadDir ([#415](https://github.com/databricks/cli/pull/415)).
 * Speed up sync integration tests ([#428](https://github.com/databricks/cli/pull/428)).

## 0.100.2

CLI:
* Reduce parallellism in locker integration test ([#407](https://github.com/databricks/bricks/pull/407)).

Bundles:
* Don't pass synthesized TMPDIR if not already set ([#409](https://github.com/databricks/bricks/pull/409)).
* Added support for bundle.Seq, simplified Mutator.Apply interface ([#403](https://github.com/databricks/bricks/pull/403)).
* Regenerated internal schema structs based on Terraform provider schemas ([#401](https://github.com/databricks/bricks/pull/401)).

## 0.100.1

CLI:
* Sync: Gracefully handle broken notebook files ([#398](https://github.com/databricks/cli/pull/398)).
* Add version flag to print version and exit ([#394](https://github.com/databricks/cli/pull/394)).
* Pass temporary directory environment variables to subprocesses ([#395](https://github.com/databricks/cli/pull/395)).
* Rename environment variables `BRICKS_` -> `DATABRICKS_` ([#393](https://github.com/databricks/cli/pull/393)).
* Update to Go SDK v0.9.0 ([#396](https://github.com/databricks/cli/pull/396)).

## 0.100.0

This release bumps the minor version to 100 to disambiguate between Databricks CLI "v1" (the Python version)
and this version, Databricks CLI "v2". This release is a major rewrite of the CLI, and is not backwards compatible.

CLI:
* Rename bricks -> databricks ([#389](https://github.com/databricks/cli/pull/389)).

Bundles:
* Added ability for deferred mutator execution ([#380](https://github.com/databricks/cli/pull/380)).
* Do not truncate local state file when pulling remote changes ([#382](https://github.com/databricks/cli/pull/382)).

## 0.0.32

* Add support for variables in bundle config. Introduces 4 ways of setting variable values, which in decreasing order of priority are: ([#383](https://github.com/databricks/cli/pull/383))([#359](https://github.com/databricks/cli/pull/359)).
  1. Command line flag. For example: `--var="foo=bar"`
  2. Environment variable. eg: BUNDLE_VAR_foo=bar
  3. Default value as defined in the applicable environments block
  4. Default value defined in variable definition
* Make the git details bundle config block optional ([#372](https://github.com/databricks/cli/pull/372)).
* Fix api post integration tests ([#371](https://github.com/databricks/cli/pull/371)).
* Fix table of content by removing not required top-level item ([#366](https://github.com/databricks/cli/pull/366)).
* Fix printing the tasks in job output in DAG execution order ([#377](https://github.com/databricks/cli/pull/377)).
* Improved error message when 'bricks bundle run' is executed before 'bricks bundle deploy' ([#378](https://github.com/databricks/cli/pull/378)).

## 0.0.31

* Add OpenAPI command coverage (both workspace and account level APIs).

### Bundles

* Automatically populate a bundle's Git repository details in its configuration tree.

## 0.0.30

* Initial preview release of the Databricks CLI.
