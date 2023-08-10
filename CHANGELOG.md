# Version changelog

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
