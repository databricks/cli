# Regression Test Report

Tested commit: 3faddda473 direct: also suppress condition on the multi-trigger list

<!-- Acceptance tests: 5 (5 added, 0 modified) — 3 regression, 2 coverage -->

| test | branch | main (0a8aae1655) | latest (v1.14.1) |
| --- | --- | --- | --- |
| TestAccept/bundle/invariant/continue_293/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl | ✅ | ❌ | ❌ |
| TestAccept/bundle/invariant/delete_idempotent/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl/READPLAN= | ✅ | ✅ | ✅ |
| TestAccept/bundle/invariant/delete_idempotent/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl/READPLAN=1 | ✅ | ✅ | ✅ |
| TestAccept/bundle/invariant/destroy_idempotent/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl/READPLAN= | ✅ | ✅ | ✅ |
| TestAccept/bundle/invariant/destroy_idempotent/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl/READPLAN=1 | ✅ | ✅ | ✅ |
| TestAccept/bundle/invariant/migrate/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl | ✅ | ❌ | ❌ |
| TestAccept/bundle/invariant/no_drift/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl/READPLAN= | ✅ | ❌ | ❌ |
| TestAccept/bundle/invariant/no_drift/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl/READPLAN=1 | ✅ | ❌ | ❌ |

<details>
<summary>TestAccept/bundle/invariant/continue_293/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl ✅ | main (0a8aae1655) ❌ | latest (v1.14.1) ❌</summary>

**main (0a8aae1655):**
```
=== RUN   TestAccept/bundle/invariant/continue_293/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl
=== PAUSE TestAccept/bundle/invariant/continue_293/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl
=== CONT  TestAccept/bundle/invariant/continue_293/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl
    acceptance_test.go:1176: Diff:
        --- bundle/invariant/continue_293/output.txt
        +++ /var/folders/5y/9kkdnjw91p11vsqwk0cvmk200000gp/T/TestAcceptbundleinvariantcontinue_293DATABRICKS_BUNDLE_ENGIN1606462624/001/output.txt
        @@ -2,3 +2,96 @@
         >>> [CLI_293] --version
         Databricks CLI v0.293.0
         INPUT_CONFIG_OK
        +Unexpected action='update' for resources.jobs.foo
        +{
        +  "plan_version": 2,
        +  "cli_version": "[CLI_VERSION]",
        +  "lineage": "[UUID]",
        +  "serial": 2,
        +  "plan": {
        +    "resources.jobs.foo": {
        +      "action": "update",
        +      "new_state": {
        +        "value": {
        +          "deployment": {
        +            "kind": "BUNDLE",
        +            "metadata_file_path": "/Workspace/Users/[USERNAME]/.bundle/test-bundle-[UNIQUE_NAME]/default/state/metadata.json"
        +          },
        +          "edit_mode": "UI_LOCKED",
        +          "format": "MULTI_TASK",
        +          "max_concurrent_runs": 1,
        +          "name": "test-job-[UNIQUE_NAME]",
        +          "queue": {
        +            "enabled": true
        +          },
        +          "trigger": {
        +            "pause_status": "UNPAUSED",
        +            "table_update": {
        +              "condition": "ANY_UPDATED",
        +              "table_names": [
        +                "samples.nyctaxi.trips"
        +              ],
        +              "wait_after_last_change_seconds": 60
        +            }
        +          }
        +        }
        +      },
        +      "remote_state": {
        +        "created_time": [UNIX_TIME_MILLIS],
        +        "creator_user_name": "[USERNAME]",
        +        "deployment": {
        +          "kind": "BUNDLE",
        +          "metadata_file_path": "/Workspace/Users/[USERNAME]/.bundle/test-bundle-[UNIQUE_NAME]/default/state/metadata.json"
        +        },
        +        "edit_mode": "UI_LOCKED",
        +        "email_notifications": {},
        +        "format": "MULTI_TASK",
        +        "job_id": [NUMID],
        +        "max_concurrent_runs": 1,
        +        "name": "test-job-[UNIQUE_NAME]",
        +        "queue": {
        +          "enabled": true
        +        },
        +        "run_as_user_name": "[USERNAME]",
        +        "timeout_seconds": 0,
        +        "trigger": {
        +          "pause_status": "UNPAUSED",
        +          "table_update": {
        +            "condition": "",
        +            "table_names": [
        +              "samples.nyctaxi.trips"
        +            ],
        +            "wait_after_last_change_seconds": 60
        +          }
        +        },
        +        "webhook_notifications": {}
        +      },
        +      "changes": {
        +        "email_notifications": {
        +          "action": "skip",
        +          "reason": "empty",
        +          "remote": {}
        +        },
        +        "timeout_seconds": {
        +          "action": "skip",
        +          "reason": "empty",
        +          "remote": 0
        +        },
        +        "trigger.table_update.condition": {
        +          "action": "update",
        +          "old": "ANY_UPDATED",
        +          "new": "ANY_UPDATED",
        +          "remote": ""
        +        },
        +        "webhook_notifications": {
        +          "action": "skip",
        +          "reason": "empty",
        +          "remote": {}
        +        }
        +      }
        +    }
        +  }
        +}
        +
        +
        +Exit code: 10
        
    acceptance_test.go:1035: 
        LOG.config: bundle:
        LOG.config:   name: test-bundle-ykar3t433be6lhbtyyaemmoo5e
        LOG.config: 
        LOG.config: resources:
        LOG.config:   jobs:
        LOG.config:     foo:
        LOG.config:       name: test-job-ykar3t433be6lhbtyyaemmoo5e
        LOG.config:       trigger:
        LOG.config:         table_update:
        LOG.config:           table_names:
        LOG.config:             - samples.nyctaxi.trips
        LOG.config:           condition: ANY_UPDATED
        LOG.config:           wait_after_last_change_seconds: 60
    acceptance_test.go:1035: LOG.cp: 
    acceptance_test.go:1035: 
        LOG.deploy: Uploading bundle files to /Workspace/Users/tester@databricks.com/.bundle/test-bundle-ykar3t433be6lhbtyyaemmoo5e/default/files...
        LOG.deploy: Updated jobs.foo
        LOG.deploy: Files: 3 uploaded, 0 deleted
        LOG.deploy: Resources: 0 created, 1 changed, 0 deleted, 0 unchanged
    acceptance_test.go:1035: 
        LOG.deploy.293: 
        LOG.deploy.293: >>> /Users/denis.bilenko/work/cli-trees/investigate-6315/acceptance/build/darwin_arm64/0.293.0/databricks bundle deploy
        LOG.deploy.293: Uploading bundle files to /Workspace/Users/tester@databricks.com/.bundle/test-bundle-ykar3t433be6lhbtyyaemmoo5e/default/files...
        LOG.deploy.293: Deploying resources...
        LOG.deploy.293: Updating deployment state...
        LOG.deploy.293: Deployment complete!
    acceptance_test.go:1035: 
        LOG.destroy: 
        LOG.destroy: >>> /var/folders/5y/9kkdnjw91p11vsqwk0cvmk200000gp/T/regression_main_cli_013rzbtg/src/databricks bundle destroy --auto-approve
        LOG.destroy: The following resources will be deleted:
        LOG.destroy:   delete resources.jobs.foo
        LOG.destroy: 
        LOG.destroy: All files and directories at the following location will be deleted: /Workspace/Users/tester@databricks.com/.bundle/test-bundle-ykar3t433be6lhbtyyaemmoo5e/default
        LOG.destroy: 
        LOG.destroy: Destroy: 1 deleted
    acceptance_test.go:1035: 
        LOG.planjson: {
        LOG.planjson:   "plan_version": 2,
        LOG.planjson:   "cli_version": "1.15.0-dev",
        LOG.planjson:   "lineage": "2b161cec-1a4a-4f36-9510-abdd769035e6",
        LOG.planjson:   "serial": 2,
        LOG.planjson:   "plan": {
        LOG.planjson:     "resources.jobs.foo": {
        LOG.planjson:       "action": "update",
        LOG.planjson:       "new_state": {
        LOG.planjson:         "value": {
        LOG.planjson:           "deployment": {
        LOG.planjson:             "kind": "BUNDLE",
        LOG.planjson:             "metadata_file_path": "/Workspace/Users/tester@databricks.com/.bundle/test-bundle-ykar3t433be6lhbtyyaemmoo5e/default/state/metadata.json"
        LOG.planjson:           },
        LOG.planjson:           "edit_mode": "UI_LOCKED",
        LOG.planjson:           "format": "MULTI_TASK",
        LOG.planjson:           "max_concurrent_runs": 1,
        LOG.planjson:           "name": "test-job-ykar3t433be6lhbtyyaemmoo5e",
        LOG.planjson:           "queue": {
        LOG.planjson:             "enabled": true
        LOG.planjson:           },
        LOG.planjson:           "trigger": {
        LOG.planjson:             "pause_status": "UNPAUSED",
        LOG.planjson:             "table_update": {
        LOG.planjson:               "condition": "ANY_UPDATED",
        LOG.planjson:               "table_names": [
        LOG.planjson:                 "samples.nyctaxi.trips"
        LOG.planjson:               ],
        LOG.planjson:               "wait_after_last_change_seconds": 60
        LOG.planjson:             }
        LOG.planjson:           }
        LOG.planjson:         }
        LOG.planjson:       },
        LOG.planjson:       "remote_state": {
        LOG.planjson:         "created_time": 1788175173424,
        LOG.planjson:         "creator_user_name": "tester@databricks.com",
        LOG.planjson:         "deployment": {
        LOG.planjson:           "kind": "BUNDLE",
        LOG.planjson:           "metadata_file_path": "/Workspace/Users/tester@databricks.com/.bundle/test-bundle-ykar3t433be6lhbtyyaemmoo5e/default/state/metadata.json"
        LOG.planjson:         },
        LOG.planjson:         "edit_mode": "UI_LOCKED",
        LOG.planjson:         "email_notifications": {},
        LOG.planjson:         "format": "MULTI_TASK",
        LOG.planjson:         "job_id": 8788175173424648000,
        LOG.planjson:         "max_concurrent_runs": 1,
        LOG.planjson:         "name": "test-job-ykar3t433be6lhbtyyaemmoo5e",
        LOG.planjson:         "queue": {
        LOG.planjson:           "enabled": true
        LOG.planjson:         },
        LOG.planjson:         "run_as_user_name": "tester@databricks.com",
        LOG.planjson:         "timeout_seconds": 0,
        LOG.planjson:         "trigger": {
        LOG.planjson:           "pause_status": "UNPAUSED",
        LOG.planjson:           "table_update": {
        LOG.planjson:             "condition": "",
        LOG.planjson:             "table_names": [
        LOG.planjson:               "samples.nyctaxi.trips"
        LOG.planjson:             ],
        LOG.planjson:             "wait_after_last_change_seconds": 60
        LOG.planjson:           }
        LOG.planjson:         },
        LOG.planjson:         "webhook_notifications": {}
        LOG.planjson:       },
        LOG.planjson:       "changes": {
        LOG.planjson:         "email_notifications": {
        LOG.planjson:           "action": "skip",
        LOG.planjson:           "reason": "empty",
        LOG.planjson:           "remote": {}
        LOG.planjson:         },
        LOG.planjson:         "timeout_seconds": {
        LOG.planjson:           "action": "skip",
        LOG.planjson:           "reason": "empty",
        LOG.planjson:           "remote": 0
        LOG.planjson:         },
        LOG.planjson:         "trigger.table_update.condition": {
        LOG.planjson:           "action": "update",
        LOG.planjson:           "old": "ANY_UPDATED",
        LOG.planjson:           "new": "ANY_UPDATED",
        LOG.planjson:           "remote": ""
        LOG.planjson:         },
        LOG.planjson:         "webhook_notifications": {
        LOG.planjson:           "action": "skip",
        LOG.planjson:           "reason": "empty",
        LOG.planjson:           "remote": {}
        LOG.planjson:         }
        LOG.planjson:       }
        LOG.planjson:     }
        LOG.planjson:   }
        LOG.planjson: }
--- FAIL: TestAccept/bundle/invariant/continue_293/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl (0.34s)
```

**latest (v1.14.1):**
```
=== RUN   TestAccept/bundle/invariant/continue_293/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl
=== PAUSE TestAccept/bundle/invariant/continue_293/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl
=== CONT  TestAccept/bundle/invariant/continue_293/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl
    acceptance_test.go:1176: Diff:
        --- bundle/invariant/continue_293/output.txt
        +++ /var/folders/5y/9kkdnjw91p11vsqwk0cvmk200000gp/T/TestAcceptbundleinvariantcontinue_293DATABRICKS_BUNDLE_ENGIN521289262/001/output.txt
        @@ -2,3 +2,96 @@
         >>> [CLI_293] --version
         Databricks CLI v0.293.0
         INPUT_CONFIG_OK
        +Unexpected action='update' for resources.jobs.foo
        +{
        +  "plan_version": 2,
        +  "cli_version": "[CLI_VERSION]",
        +  "lineage": "[UUID]",
        +  "serial": 2,
        +  "plan": {
        +    "resources.jobs.foo": {
        +      "action": "update",
        +      "new_state": {
        +        "value": {
        +          "deployment": {
        +            "kind": "BUNDLE",
        +            "metadata_file_path": "/Workspace/Users/[USERNAME]/.bundle/test-bundle-[UNIQUE_NAME]/default/state/metadata.json"
        +          },
        +          "edit_mode": "UI_LOCKED",
        +          "format": "MULTI_TASK",
        +          "max_concurrent_runs": 1,
        +          "name": "test-job-[UNIQUE_NAME]",
        +          "queue": {
        +            "enabled": true
        +          },
        +          "trigger": {
        +            "pause_status": "UNPAUSED",
        +            "table_update": {
        +              "condition": "ANY_UPDATED",
        +              "table_names": [
        +                "samples.nyctaxi.trips"
        +              ],
        +              "wait_after_last_change_seconds": 60
        +            }
        +          }
        +        }
        +      },
        +      "remote_state": {
        +        "created_time": [UNIX_TIME_MILLIS],
        +        "creator_user_name": "[USERNAME]",
        +        "deployment": {
        +          "kind": "BUNDLE",
        +          "metadata_file_path": "/Workspace/Users/[USERNAME]/.bundle/test-bundle-[UNIQUE_NAME]/default/state/metadata.json"
        +        },
        +        "edit_mode": "UI_LOCKED",
        +        "email_notifications": {},
        +        "format": "MULTI_TASK",
        +        "job_id": [NUMID],
        +        "max_concurrent_runs": 1,
        +        "name": "test-job-[UNIQUE_NAME]",
        +        "queue": {
        +          "enabled": true
        +        },
        +        "run_as_user_name": "[USERNAME]",
        +        "timeout_seconds": 0,
        +        "trigger": {
        +          "pause_status": "UNPAUSED",
        +          "table_update": {
        +            "condition": "",
        +            "table_names": [
        +              "samples.nyctaxi.trips"
        +            ],
        +            "wait_after_last_change_seconds": 60
        +          }
        +        },
        +        "webhook_notifications": {}
        +      },
        +      "changes": {
        +        "email_notifications": {
        +          "action": "skip",
        +          "reason": "empty",
        +          "remote": {}
        +        },
        +        "timeout_seconds": {
        +          "action": "skip",
        +          "reason": "empty",
        +          "remote": 0
        +        },
        +        "trigger.table_update.condition": {
        +          "action": "update",
        +          "old": "ANY_UPDATED",
        +          "new": "ANY_UPDATED",
        +          "remote": ""
        +        },
        +        "webhook_notifications": {
        +          "action": "skip",
        +          "reason": "empty",
        +          "remote": {}
        +        }
        +      }
        +    }
        +  }
        +}
        +
        +
        +Exit code: 10
        
    acceptance_test.go:1035: 
        LOG.config: bundle:
        LOG.config:   name: test-bundle-n4vgbsfk4zhq7nw4kom25anvka
        LOG.config: 
        LOG.config: resources:
        LOG.config:   jobs:
        LOG.config:     foo:
        LOG.config:       name: test-job-n4vgbsfk4zhq7nw4kom25anvka
        LOG.config:       trigger:
        LOG.config:         table_update:
        LOG.config:           table_names:
        LOG.config:             - samples.nyctaxi.trips
        LOG.config:           condition: ANY_UPDATED
        LOG.config:           wait_after_last_change_seconds: 60
    acceptance_test.go:1035: LOG.cp: 
    acceptance_test.go:1035: 
        LOG.deploy: Uploading bundle files to /Workspace/Users/tester@databricks.com/.bundle/test-bundle-n4vgbsfk4zhq7nw4kom25anvka/default/files...
        LOG.deploy: Updated jobs.foo
        LOG.deploy: Files: 3 uploaded, 0 deleted
        LOG.deploy: Resources: 0 created, 1 changed, 0 deleted, 0 unchanged
    acceptance_test.go:1035: 
        LOG.deploy.293: 
        LOG.deploy.293: >>> /Users/denis.bilenko/work/cli-trees/investigate-6315/acceptance/build/darwin_arm64/0.293.0/databricks bundle deploy
        LOG.deploy.293: Uploading bundle files to /Workspace/Users/tester@databricks.com/.bundle/test-bundle-n4vgbsfk4zhq7nw4kom25anvka/default/files...
        LOG.deploy.293: Deploying resources...
        LOG.deploy.293: Updating deployment state...
        LOG.deploy.293: Deployment complete!
    acceptance_test.go:1035: 
        LOG.destroy: 
        LOG.destroy: >>> /Users/denis.bilenko/work/cli-trees/investigate-6315/acceptance/build/darwin_arm64/1.14.1/databricks bundle destroy --auto-approve
        LOG.destroy: The following resources will be deleted:
        LOG.destroy:   delete resources.jobs.foo
        LOG.destroy: 
        LOG.destroy: All files and directories at the following location will be deleted: /Workspace/Users/tester@databricks.com/.bundle/test-bundle-n4vgbsfk4zhq7nw4kom25anvka/default
        LOG.destroy: 
        LOG.destroy: Destroy: 1 deleted
    acceptance_test.go:1035: 
        LOG.planjson: {
        LOG.planjson:   "plan_version": 2,
        LOG.planjson:   "cli_version": "1.14.1",
        LOG.planjson:   "lineage": "3a6545f4-baf4-4575-9243-f5de4b7e938d",
        LOG.planjson:   "serial": 2,
        LOG.planjson:   "plan": {
        LOG.planjson:     "resources.jobs.foo": {
        LOG.planjson:       "action": "update",
        LOG.planjson:       "new_state": {
        LOG.planjson:         "value": {
        LOG.planjson:           "deployment": {
        LOG.planjson:             "kind": "BUNDLE",
        LOG.planjson:             "metadata_file_path": "/Workspace/Users/tester@databricks.com/.bundle/test-bundle-n4vgbsfk4zhq7nw4kom25anvka/default/state/metadata.json"
        LOG.planjson:           },
        LOG.planjson:           "edit_mode": "UI_LOCKED",
        LOG.planjson:           "format": "MULTI_TASK",
        LOG.planjson:           "max_concurrent_runs": 1,
        LOG.planjson:           "name": "test-job-n4vgbsfk4zhq7nw4kom25anvka",
        LOG.planjson:           "queue": {
        LOG.planjson:             "enabled": true
        LOG.planjson:           },
        LOG.planjson:           "trigger": {
        LOG.planjson:             "pause_status": "UNPAUSED",
        LOG.planjson:             "table_update": {
        LOG.planjson:               "condition": "ANY_UPDATED",
        LOG.planjson:               "table_names": [
        LOG.planjson:                 "samples.nyctaxi.trips"
        LOG.planjson:               ],
        LOG.planjson:               "wait_after_last_change_seconds": 60
        LOG.planjson:             }
        LOG.planjson:           }
        LOG.planjson:         }
        LOG.planjson:       },
        LOG.planjson:       "remote_state": {
        LOG.planjson:         "created_time": 1788175210109,
        LOG.planjson:         "creator_user_name": "tester@databricks.com",
        LOG.planjson:         "deployment": {
        LOG.planjson:           "kind": "BUNDLE",
        LOG.planjson:           "metadata_file_path": "/Workspace/Users/tester@databricks.com/.bundle/test-bundle-n4vgbsfk4zhq7nw4kom25anvka/default/state/metadata.json"
        LOG.planjson:         },
        LOG.planjson:         "edit_mode": "UI_LOCKED",
        LOG.planjson:         "email_notifications": {},
        LOG.planjson:         "format": "MULTI_TASK",
        LOG.planjson:         "job_id": 8788175210109128000,
        LOG.planjson:         "max_concurrent_runs": 1,
        LOG.planjson:         "name": "test-job-n4vgbsfk4zhq7nw4kom25anvka",
        LOG.planjson:         "queue": {
        LOG.planjson:           "enabled": true
        LOG.planjson:         },
        LOG.planjson:         "run_as_user_name": "tester@databricks.com",
        LOG.planjson:         "timeout_seconds": 0,
        LOG.planjson:         "trigger": {
        LOG.planjson:           "pause_status": "UNPAUSED",
        LOG.planjson:           "table_update": {
        LOG.planjson:             "condition": "",
        LOG.planjson:             "table_names": [
        LOG.planjson:               "samples.nyctaxi.trips"
        LOG.planjson:             ],
        LOG.planjson:             "wait_after_last_change_seconds": 60
        LOG.planjson:           }
        LOG.planjson:         },
        LOG.planjson:         "webhook_notifications": {}
        LOG.planjson:       },
        LOG.planjson:       "changes": {
        LOG.planjson:         "email_notifications": {
        LOG.planjson:           "action": "skip",
        LOG.planjson:           "reason": "empty",
        LOG.planjson:           "remote": {}
        LOG.planjson:         },
        LOG.planjson:         "timeout_seconds": {
        LOG.planjson:           "action": "skip",
        LOG.planjson:           "reason": "empty",
        LOG.planjson:           "remote": 0
        LOG.planjson:         },
        LOG.planjson:         "trigger.table_update.condition": {
        LOG.planjson:           "action": "update",
        LOG.planjson:           "old": "ANY_UPDATED",
        LOG.planjson:           "new": "ANY_UPDATED",
        LOG.planjson:           "remote": ""
        LOG.planjson:         },
        LOG.planjson:         "webhook_notifications": {
        LOG.planjson:           "action": "skip",
        LOG.planjson:           "reason": "empty",
        LOG.planjson:           "remote": {}
        LOG.planjson:         }
        LOG.planjson:       }
        LOG.planjson:     }
        LOG.planjson:   }
        LOG.planjson: }
--- FAIL: TestAccept/bundle/invariant/continue_293/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl (1.28s)
```

</details>

<details>
<summary>TestAccept/bundle/invariant/migrate/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl ✅ | main (0a8aae1655) ❌ | latest (v1.14.1) ❌</summary>

**main (0a8aae1655):**
```
=== RUN   TestAccept/bundle/invariant/migrate/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl
=== PAUSE TestAccept/bundle/invariant/migrate/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl
=== CONT  TestAccept/bundle/invariant/migrate/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl
    acceptance_test.go:1176: Diff:
        --- bundle/invariant/migrate/output.txt
        +++ /var/folders/5y/9kkdnjw91p11vsqwk0cvmk200000gp/T/TestAcceptbundleinvariantmigrateDATABRICKS_BUNDLE_ENGINE=dir4115529494/001/output.txt
        @@ -1 +1,94 @@
         INPUT_CONFIG_OK
        +Unexpected action='update' for resources.jobs.foo
        +{
        +  "plan_version": 2,
        +  "cli_version": "[CLI_VERSION]",
        +  "lineage": "[UUID]",
        +  "serial": 3,
        +  "plan": {
        +    "resources.jobs.foo": {
        +      "action": "update",
        +      "new_state": {
        +        "value": {
        +          "deployment": {
        +            "kind": "BUNDLE",
        +            "metadata_file_path": "/Workspace/Users/[USERNAME]/.bundle/test-bundle-[UNIQUE_NAME]/default/state/metadata.json"
        +          },
        +          "edit_mode": "UI_LOCKED",
        +          "format": "MULTI_TASK",
        +          "max_concurrent_runs": 1,
        +          "name": "test-job-[UNIQUE_NAME]",
        +          "queue": {
        +            "enabled": true
        +          },
        +          "trigger": {
        +            "pause_status": "UNPAUSED",
        +            "table_update": {
        +              "condition": "ANY_UPDATED",
        +              "table_names": [
        +                "samples.nyctaxi.trips"
        +              ],
        +              "wait_after_last_change_seconds": 60
        +            }
        +          }
        +        }
        +      },
        +      "remote_state": {
        +        "created_time": [UNIX_TIME_MILLIS],
        +        "creator_user_name": "[USERNAME]",
        +        "deployment": {
        +          "kind": "BUNDLE",
        +          "metadata_file_path": "/Workspace/Users/[USERNAME]/.bundle/test-bundle-[UNIQUE_NAME]/default/state/metadata.json"
        +        },
        +        "edit_mode": "UI_LOCKED",
        +        "email_notifications": {},
        +        "format": "MULTI_TASK",
        +        "job_id": [NUMID],
        +        "max_concurrent_runs": 1,
        +        "name": "test-job-[UNIQUE_NAME]",
        +        "queue": {
        +          "enabled": true
        +        },
        +        "run_as_user_name": "[USERNAME]",
        +        "timeout_seconds": 0,
        +        "trigger": {
        +          "pause_status": "UNPAUSED",
        +          "table_update": {
        +            "condition": "",
        +            "table_names": [
        +              "samples.nyctaxi.trips"
        +            ],
        +            "wait_after_last_change_seconds": 60
        +          }
        +        },
        +        "webhook_notifications": {}
        +      },
        +      "changes": {
        +        "email_notifications": {
        +          "action": "skip",
        +          "reason": "empty",
        +          "remote": {}
        +        },
        +        "timeout_seconds": {
        +          "action": "skip",
        +          "reason": "empty",
        +          "remote": 0
        +        },
        +        "trigger.table_update.condition": {
        +          "action": "update",
        +          "old": "ANY_UPDATED",
        +          "new": "ANY_UPDATED",
        +          "remote": ""
        +        },
        +        "webhook_notifications": {
        +          "action": "skip",
        +          "reason": "empty",
        +          "remote": {}
        +        }
        +      }
        +    }
        +  }
        +}
        +
        +
        +Exit code: 10
        
    acceptance_test.go:1035: 
        LOG.config: bundle:
        LOG.config:   name: test-bundle-rxizc4ur4bahbo3ioqdhkia43q
        LOG.config: 
        LOG.config: resources:
        LOG.config:   jobs:
        LOG.config:     foo:
        LOG.config:       name: test-job-rxizc4ur4bahbo3ioqdhkia43q
        LOG.config:       trigger:
        LOG.config:         table_update:
        LOG.config:           table_names:
        LOG.config:             - samples.nyctaxi.trips
        LOG.config:           condition: ANY_UPDATED
        LOG.config:           wait_after_last_change_seconds: 60
    acceptance_test.go:1035: LOG.cp: 
    acceptance_test.go:1035: 
        LOG.deploy: 
        LOG.deploy: >>> DATABRICKS_BUNDLE_ENGINE=terraform /var/folders/5y/9kkdnjw91p11vsqwk0cvmk200000gp/T/regression_main_cli_013rzbtg/src/databricks bundle deploy
        LOG.deploy: Uploading bundle files to /Workspace/Users/tester@databricks.com/.bundle/test-bundle-rxizc4ur4bahbo3ioqdhkia43q/default/files...
        LOG.deploy: Created jobs.foo
        LOG.deploy: Files: 15 uploaded, 0 deleted
        LOG.deploy: Resources: 1 created, 0 changed, 0 deleted, 0 unchanged
    acceptance_test.go:1035: 
        LOG.destroy: 
        LOG.destroy: >>> /var/folders/5y/9kkdnjw91p11vsqwk0cvmk200000gp/T/regression_main_cli_013rzbtg/src/databricks bundle destroy --auto-approve
        LOG.destroy: The following resources will be deleted:
        LOG.destroy:   delete resources.jobs.foo
        LOG.destroy: 
        LOG.destroy: All files and directories at the following location will be deleted: /Workspace/Users/tester@databricks.com/.bundle/test-bundle-rxizc4ur4bahbo3ioqdhkia43q/default
        LOG.destroy: 
        LOG.destroy: Destroy: 1 deleted
    acceptance_test.go:1035: 
        LOG.migrate: 
        LOG.migrate: >>> /var/folders/5y/9kkdnjw91p11vsqwk0cvmk200000gp/T/regression_main_cli_013rzbtg/src/databricks bundle deployment migrate
        LOG.migrate: Success! Migrated 1 resources to direct engine state file: /private/var/folders/5y/9kkdnjw91p11vsqwk0cvmk200000gp/T/TestAcceptbundleinvariantmigrateDATABRICKS_BUNDLE_ENGINE=dir4115529494/001/.databricks/bundle/default/resources.json
        LOG.migrate: 
        LOG.migrate: Validate the migration by running "databricks bundle plan", there should be no actions planned.
        LOG.migrate: 
        LOG.migrate: The state file is not synchronized to the workspace yet. To do that and finalize the migration, run "bundle deploy".
        LOG.migrate: 
        LOG.migrate: To undo the migration, remove /private/var/folders/5y/9kkdnjw91p11vsqwk0cvmk200000gp/T/TestAcceptbundleinvariantmigrateDATABRICKS_BUNDLE_ENGINE=dir4115529494/001/.databricks/bundle/default/resources.json and rename /private/var/folders/5y/9kkdnjw91p11vsqwk0cvmk200000gp/T/TestAcceptbundleinvariantmigrateDATABRICKS_BUNDLE_ENGINE=dir4115529494/001/.databricks/bundle/default/terraform/terraform.tfstate.backup to /private/var/folders/5y/9kkdnjw91p11vsqwk0cvmk200000gp/T/TestAcceptbundleinvariantmigrateDATABRICKS_BUNDLE_ENGINE=dir4115529494/001/.databricks/bundle/default/terraform/terraform.tfstate
    acceptance_test.go:1035: 
        LOG.planjson: {
        LOG.planjson:   "plan_version": 2,
        LOG.planjson:   "cli_version": "1.15.0-dev",
        LOG.planjson:   "lineage": "f4e6560d-011c-ef4d-d4f3-065e3b99b6a9",
        LOG.planjson:   "serial": 3,
        LOG.planjson:   "plan": {
        LOG.planjson:     "resources.jobs.foo": {
        LOG.planjson:       "action": "update",
        LOG.planjson:       "new_state": {
        LOG.planjson:         "value": {
        LOG.planjson:           "deployment": {
        LOG.planjson:             "kind": "BUNDLE",
        LOG.planjson:             "metadata_file_path": "/Workspace/Users/tester@databricks.com/.bundle/test-bundle-rxizc4ur4bahbo3ioqdhkia43q/default/state/metadata.json"
        LOG.planjson:           },
        LOG.planjson:           "edit_mode": "UI_LOCKED",
        LOG.planjson:           "format": "MULTI_TASK",
        LOG.planjson:           "max_concurrent_runs": 1,
        LOG.planjson:           "name": "test-job-rxizc4ur4bahbo3ioqdhkia43q",
        LOG.planjson:           "queue": {
        LOG.planjson:             "enabled": true
        LOG.planjson:           },
        LOG.planjson:           "trigger": {
        LOG.planjson:             "pause_status": "UNPAUSED",
        LOG.planjson:             "table_update": {
        LOG.planjson:               "condition": "ANY_UPDATED",
        LOG.planjson:               "table_names": [
        LOG.planjson:                 "samples.nyctaxi.trips"
        LOG.planjson:               ],
        LOG.planjson:               "wait_after_last_change_seconds": 60
        LOG.planjson:             }
        LOG.planjson:           }
        LOG.planjson:         }
        LOG.planjson:       },
        LOG.planjson:       "remote_state": {
        LOG.planjson:         "created_time": 1788175193300,
        LOG.planjson:         "creator_user_name": "tester@databricks.com",
        LOG.planjson:         "deployment": {
        LOG.planjson:           "kind": "BUNDLE",
        LOG.planjson:           "metadata_file_path": "/Workspace/Users/tester@databricks.com/.bundle/test-bundle-rxizc4ur4bahbo3ioqdhkia43q/default/state/metadata.json"
        LOG.planjson:         },
        LOG.planjson:         "edit_mode": "UI_LOCKED",
        LOG.planjson:         "email_notifications": {},
        LOG.planjson:         "format": "MULTI_TASK",
        LOG.planjson:         "job_id": 8788175193300712000,
        LOG.planjson:         "max_concurrent_runs": 1,
        LOG.planjson:         "name": "test-job-rxizc4ur4bahbo3ioqdhkia43q",
        LOG.planjson:         "queue": {
        LOG.planjson:           "enabled": true
        LOG.planjson:         },
        LOG.planjson:         "run_as_user_name": "tester@databricks.com",
        LOG.planjson:         "timeout_seconds": 0,
        LOG.planjson:         "trigger": {
        LOG.planjson:           "pause_status": "UNPAUSED",
        LOG.planjson:           "table_update": {
        LOG.planjson:             "condition": "",
        LOG.planjson:             "table_names": [
        LOG.planjson:               "samples.nyctaxi.trips"
        LOG.planjson:             ],
        LOG.planjson:             "wait_after_last_change_seconds": 60
        LOG.planjson:           }
        LOG.planjson:         },
        LOG.planjson:         "webhook_notifications": {}
        LOG.planjson:       },
        LOG.planjson:       "changes": {
        LOG.planjson:         "email_notifications": {
        LOG.planjson:           "action": "skip",
        LOG.planjson:           "reason": "empty",
        LOG.planjson:           "remote": {}
        LOG.planjson:         },
        LOG.planjson:         "timeout_seconds": {
        LOG.planjson:           "action": "skip",
        LOG.planjson:           "reason": "empty",
        LOG.planjson:           "remote": 0
        LOG.planjson:         },
        LOG.planjson:         "trigger.table_update.condition": {
        LOG.planjson:           "action": "update",
        LOG.planjson:           "old": "ANY_UPDATED",
        LOG.planjson:           "new": "ANY_UPDATED",
        LOG.planjson:           "remote": ""
        LOG.planjson:         },
        LOG.planjson:         "webhook_notifications": {
        LOG.planjson:           "action": "skip",
        LOG.planjson:           "reason": "empty",
        LOG.planjson:           "remote": {}
        LOG.planjson:         }
        LOG.planjson:       }
        LOG.planjson:     }
        LOG.planjson:   }
        LOG.planjson: }
--- FAIL: TestAccept/bundle/invariant/migrate/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl (1.24s)
```

**latest (v1.14.1):**
```
=== RUN   TestAccept/bundle/invariant/migrate/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl
=== PAUSE TestAccept/bundle/invariant/migrate/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl
=== CONT  TestAccept/bundle/invariant/migrate/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl
    acceptance_test.go:1176: Diff:
        --- bundle/invariant/migrate/output.txt
        +++ /var/folders/5y/9kkdnjw91p11vsqwk0cvmk200000gp/T/TestAcceptbundleinvariantmigrateDATABRICKS_BUNDLE_ENGINE=dir2695796446/001/output.txt
        @@ -1 +1,94 @@
         INPUT_CONFIG_OK
        +Unexpected action='update' for resources.jobs.foo
        +{
        +  "plan_version": 2,
        +  "cli_version": "[CLI_VERSION]",
        +  "lineage": "[UUID]",
        +  "serial": 3,
        +  "plan": {
        +    "resources.jobs.foo": {
        +      "action": "update",
        +      "new_state": {
        +        "value": {
        +          "deployment": {
        +            "kind": "BUNDLE",
        +            "metadata_file_path": "/Workspace/Users/[USERNAME]/.bundle/test-bundle-[UNIQUE_NAME]/default/state/metadata.json"
        +          },
        +          "edit_mode": "UI_LOCKED",
        +          "format": "MULTI_TASK",
        +          "max_concurrent_runs": 1,
        +          "name": "test-job-[UNIQUE_NAME]",
        +          "queue": {
        +            "enabled": true
        +          },
        +          "trigger": {
        +            "pause_status": "UNPAUSED",
        +            "table_update": {
        +              "condition": "ANY_UPDATED",
        +              "table_names": [
        +                "samples.nyctaxi.trips"
        +              ],
        +              "wait_after_last_change_seconds": 60
        +            }
        +          }
        +        }
        +      },
        +      "remote_state": {
        +        "created_time": [UNIX_TIME_MILLIS],
        +        "creator_user_name": "[USERNAME]",
        +        "deployment": {
        +          "kind": "BUNDLE",
        +          "metadata_file_path": "/Workspace/Users/[USERNAME]/.bundle/test-bundle-[UNIQUE_NAME]/default/state/metadata.json"
        +        },
        +        "edit_mode": "UI_LOCKED",
        +        "email_notifications": {},
        +        "format": "MULTI_TASK",
        +        "job_id": [NUMID],
        +        "max_concurrent_runs": 1,
        +        "name": "test-job-[UNIQUE_NAME]",
        +        "queue": {
        +          "enabled": true
        +        },
        +        "run_as_user_name": "[USERNAME]",
        +        "timeout_seconds": 0,
        +        "trigger": {
        +          "pause_status": "UNPAUSED",
        +          "table_update": {
        +            "condition": "",
        +            "table_names": [
        +              "samples.nyctaxi.trips"
        +            ],
        +            "wait_after_last_change_seconds": 60
        +          }
        +        },
        +        "webhook_notifications": {}
        +      },
        +      "changes": {
        +        "email_notifications": {
        +          "action": "skip",
        +          "reason": "empty",
        +          "remote": {}
        +        },
        +        "timeout_seconds": {
        +          "action": "skip",
        +          "reason": "empty",
        +          "remote": 0
        +        },
        +        "trigger.table_update.condition": {
        +          "action": "update",
        +          "old": "ANY_UPDATED",
        +          "new": "ANY_UPDATED",
        +          "remote": ""
        +        },
        +        "webhook_notifications": {
        +          "action": "skip",
        +          "reason": "empty",
        +          "remote": {}
        +        }
        +      }
        +    }
        +  }
        +}
        +
        +
        +Exit code: 10
        
    acceptance_test.go:1035: 
        LOG.config: bundle:
        LOG.config:   name: test-bundle-phrvp4kgonazppmpqjf4qxz5xi
        LOG.config: 
        LOG.config: resources:
        LOG.config:   jobs:
        LOG.config:     foo:
        LOG.config:       name: test-job-phrvp4kgonazppmpqjf4qxz5xi
        LOG.config:       trigger:
        LOG.config:         table_update:
        LOG.config:           table_names:
        LOG.config:             - samples.nyctaxi.trips
        LOG.config:           condition: ANY_UPDATED
        LOG.config:           wait_after_last_change_seconds: 60
    acceptance_test.go:1035: LOG.cp: 
    acceptance_test.go:1035: 
        LOG.deploy: 
        LOG.deploy: >>> DATABRICKS_BUNDLE_ENGINE=terraform /Users/denis.bilenko/work/cli-trees/investigate-6315/acceptance/build/darwin_arm64/1.14.1/databricks bundle deploy
        LOG.deploy: Uploading bundle files to /Workspace/Users/tester@databricks.com/.bundle/test-bundle-phrvp4kgonazppmpqjf4qxz5xi/default/files...
        LOG.deploy: Created jobs.foo
        LOG.deploy: Files: 15 uploaded, 0 deleted
        LOG.deploy: Resources: 1 created, 0 changed, 0 deleted, 0 unchanged
    acceptance_test.go:1035: 
        LOG.destroy: 
        LOG.destroy: >>> /Users/denis.bilenko/work/cli-trees/investigate-6315/acceptance/build/darwin_arm64/1.14.1/databricks bundle destroy --auto-approve
        LOG.destroy: The following resources will be deleted:
        LOG.destroy:   delete resources.jobs.foo
        LOG.destroy: 
        LOG.destroy: All files and directories at the following location will be deleted: /Workspace/Users/tester@databricks.com/.bundle/test-bundle-phrvp4kgonazppmpqjf4qxz5xi/default
        LOG.destroy: 
        LOG.destroy: Destroy: 1 deleted
    acceptance_test.go:1035: 
        LOG.migrate: 
        LOG.migrate: >>> /Users/denis.bilenko/work/cli-trees/investigate-6315/acceptance/build/darwin_arm64/1.14.1/databricks bundle deployment migrate
        LOG.migrate: Success! Migrated 1 resources to direct engine state file: /private/var/folders/5y/9kkdnjw91p11vsqwk0cvmk200000gp/T/TestAcceptbundleinvariantmigrateDATABRICKS_BUNDLE_ENGINE=dir2695796446/001/.databricks/bundle/default/resources.json
        LOG.migrate: 
        LOG.migrate: Validate the migration by running "databricks bundle plan", there should be no actions planned.
        LOG.migrate: 
        LOG.migrate: The state file is not synchronized to the workspace yet. To do that and finalize the migration, run "bundle deploy".
        LOG.migrate: 
        LOG.migrate: To undo the migration, remove /private/var/folders/5y/9kkdnjw91p11vsqwk0cvmk200000gp/T/TestAcceptbundleinvariantmigrateDATABRICKS_BUNDLE_ENGINE=dir2695796446/001/.databricks/bundle/default/resources.json and rename /private/var/folders/5y/9kkdnjw91p11vsqwk0cvmk200000gp/T/TestAcceptbundleinvariantmigrateDATABRICKS_BUNDLE_ENGINE=dir2695796446/001/.databricks/bundle/default/terraform/terraform.tfstate.backup to /private/var/folders/5y/9kkdnjw91p11vsqwk0cvmk200000gp/T/TestAcceptbundleinvariantmigrateDATABRICKS_BUNDLE_ENGINE=dir2695796446/001/.databricks/bundle/default/terraform/terraform.tfstate
    acceptance_test.go:1035: 
        LOG.planjson: {
        LOG.planjson:   "plan_version": 2,
        LOG.planjson:   "cli_version": "1.14.1",
        LOG.planjson:   "lineage": "99ac88c1-a901-e108-88ec-2132af74868f",
        LOG.planjson:   "serial": 3,
        LOG.planjson:   "plan": {
        LOG.planjson:     "resources.jobs.foo": {
        LOG.planjson:       "action": "update",
        LOG.planjson:       "new_state": {
        LOG.planjson:         "value": {
        LOG.planjson:           "deployment": {
        LOG.planjson:             "kind": "BUNDLE",
        LOG.planjson:             "metadata_file_path": "/Workspace/Users/tester@databricks.com/.bundle/test-bundle-phrvp4kgonazppmpqjf4qxz5xi/default/state/metadata.json"
        LOG.planjson:           },
        LOG.planjson:           "edit_mode": "UI_LOCKED",
        LOG.planjson:           "format": "MULTI_TASK",
        LOG.planjson:           "max_concurrent_runs": 1,
        LOG.planjson:           "name": "test-job-phrvp4kgonazppmpqjf4qxz5xi",
        LOG.planjson:           "queue": {
        LOG.planjson:             "enabled": true
        LOG.planjson:           },
        LOG.planjson:           "trigger": {
        LOG.planjson:             "pause_status": "UNPAUSED",
        LOG.planjson:             "table_update": {
        LOG.planjson:               "condition": "ANY_UPDATED",
        LOG.planjson:               "table_names": [
        LOG.planjson:                 "samples.nyctaxi.trips"
        LOG.planjson:               ],
        LOG.planjson:               "wait_after_last_change_seconds": 60
        LOG.planjson:             }
        LOG.planjson:           }
        LOG.planjson:         }
        LOG.planjson:       },
        LOG.planjson:       "remote_state": {
        LOG.planjson:         "created_time": 1788175229756,
        LOG.planjson:         "creator_user_name": "tester@databricks.com",
        LOG.planjson:         "deployment": {
        LOG.planjson:           "kind": "BUNDLE",
        LOG.planjson:           "metadata_file_path": "/Workspace/Users/tester@databricks.com/.bundle/test-bundle-phrvp4kgonazppmpqjf4qxz5xi/default/state/metadata.json"
        LOG.planjson:         },
        LOG.planjson:         "edit_mode": "UI_LOCKED",
        LOG.planjson:         "email_notifications": {},
        LOG.planjson:         "format": "MULTI_TASK",
        LOG.planjson:         "job_id": 8788175229756196000,
        LOG.planjson:         "max_concurrent_runs": 1,
        LOG.planjson:         "name": "test-job-phrvp4kgonazppmpqjf4qxz5xi",
        LOG.planjson:         "queue": {
        LOG.planjson:           "enabled": true
        LOG.planjson:         },
        LOG.planjson:         "run_as_user_name": "tester@databricks.com",
        LOG.planjson:         "timeout_seconds": 0,
        LOG.planjson:         "trigger": {
        LOG.planjson:           "pause_status": "UNPAUSED",
        LOG.planjson:           "table_update": {
        LOG.planjson:             "condition": "",
        LOG.planjson:             "table_names": [
        LOG.planjson:               "samples.nyctaxi.trips"
        LOG.planjson:             ],
        LOG.planjson:             "wait_after_last_change_seconds": 60
        LOG.planjson:           }
        LOG.planjson:         },
        LOG.planjson:         "webhook_notifications": {}
        LOG.planjson:       },
        LOG.planjson:       "changes": {
        LOG.planjson:         "email_notifications": {
        LOG.planjson:           "action": "skip",
        LOG.planjson:           "reason": "empty",
        LOG.planjson:           "remote": {}
        LOG.planjson:         },
        LOG.planjson:         "timeout_seconds": {
        LOG.planjson:           "action": "skip",
        LOG.planjson:           "reason": "empty",
        LOG.planjson:           "remote": 0
        LOG.planjson:         },
        LOG.planjson:         "trigger.table_update.condition": {
        LOG.planjson:           "action": "update",
        LOG.planjson:           "old": "ANY_UPDATED",
        LOG.planjson:           "new": "ANY_UPDATED",
        LOG.planjson:           "remote": ""
        LOG.planjson:         },
        LOG.planjson:         "webhook_notifications": {
        LOG.planjson:           "action": "skip",
        LOG.planjson:           "reason": "empty",
        LOG.planjson:           "remote": {}
        LOG.planjson:         }
        LOG.planjson:       }
        LOG.planjson:     }
        LOG.planjson:   }
        LOG.planjson: }
--- FAIL: TestAccept/bundle/invariant/migrate/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl (1.25s)
```

</details>

<details>
<summary>TestAccept/bundle/invariant/no_drift/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl/READPLAN= ✅ | main (0a8aae1655) ❌ | latest (v1.14.1) ❌</summary>

**main (0a8aae1655):**
```
=== RUN   TestAccept/bundle/invariant/no_drift/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl/READPLAN=
=== PAUSE TestAccept/bundle/invariant/no_drift/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl/READPLAN=
=== CONT  TestAccept/bundle/invariant/no_drift/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl/READPLAN=
    acceptance_test.go:1176: Diff:
        --- bundle/invariant/no_drift/output.txt
        +++ /var/folders/5y/9kkdnjw91p11vsqwk0cvmk200000gp/T/TestAcceptbundleinvariantno_driftDATABRICKS_BUNDLE_ENGINE=di4208376309/001/output.txt
        @@ -1 +1,94 @@
         INPUT_CONFIG_OK
        +Unexpected action='update' for resources.jobs.foo
        +{
        +  "plan_version": 2,
        +  "cli_version": "[CLI_VERSION]",
        +  "lineage": "[UUID]",
        +  "serial": 1,
        +  "plan": {
        +    "resources.jobs.foo": {
        +      "action": "update",
        +      "new_state": {
        +        "value": {
        +          "deployment": {
        +            "kind": "BUNDLE",
        +            "metadata_file_path": "/Workspace/Users/[USERNAME]/.bundle/test-bundle-[UNIQUE_NAME]/default/state/metadata.json"
        +          },
        +          "edit_mode": "UI_LOCKED",
        +          "format": "MULTI_TASK",
        +          "max_concurrent_runs": 1,
        +          "name": "test-job-[UNIQUE_NAME]",
        +          "queue": {
        +            "enabled": true
        +          },
        +          "trigger": {
        +            "pause_status": "UNPAUSED",
        +            "table_update": {
        +              "condition": "ANY_UPDATED",
        +              "table_names": [
        +                "samples.nyctaxi.trips"
        +              ],
        +              "wait_after_last_change_seconds": 60
        +            }
        +          }
        +        }
        +      },
        +      "remote_state": {
        +        "created_time": [UNIX_TIME_MILLIS],
        +        "creator_user_name": "[USERNAME]",
        +        "deployment": {
        +          "kind": "BUNDLE",
        +          "metadata_file_path": "/Workspace/Users/[USERNAME]/.bundle/test-bundle-[UNIQUE_NAME]/default/state/metadata.json"
        +        },
        +        "edit_mode": "UI_LOCKED",
        +        "email_notifications": {},
        +        "format": "MULTI_TASK",
        +        "job_id": [NUMID],
        +        "max_concurrent_runs": 1,
        +        "name": "test-job-[UNIQUE_NAME]",
        +        "queue": {
        +          "enabled": true
        +        },
        +        "run_as_user_name": "[USERNAME]",
        +        "timeout_seconds": 0,
        +        "trigger": {
        +          "pause_status": "UNPAUSED",
        +          "table_update": {
        +            "condition": "",
        +            "table_names": [
        +              "samples.nyctaxi.trips"
        +            ],
        +            "wait_after_last_change_seconds": 60
        +          }
        +        },
        +        "webhook_notifications": {}
        +      },
        +      "changes": {
        +        "email_notifications": {
        +          "action": "skip",
        +          "reason": "empty",
        +          "remote": {}
        +        },
        +        "timeout_seconds": {
        +          "action": "skip",
        +          "reason": "empty",
        +          "remote": 0
        +        },
        +        "trigger.table_update.condition": {
        +          "action": "update",
        +          "old": "ANY_UPDATED",
        +          "new": "ANY_UPDATED",
        +          "remote": ""
        +        },
        +        "webhook_notifications": {
        +          "action": "skip",
        +          "reason": "empty",
        +          "remote": {}
        +        }
        +      }
        +    }
        +  }
        +}
        +
        +
        +Exit code: 10
        
    acceptance_test.go:1035: 
        LOG.config: bundle:
        LOG.config:   name: test-bundle-7qu6zonevnaltdicqxxraw3xqm
        LOG.config: 
        LOG.config: resources:
        LOG.config:   jobs:
        LOG.config:     foo:
        LOG.config:       name: test-job-7qu6zonevnaltdicqxxraw3xqm
        LOG.config:       trigger:
        LOG.config:         table_update:
        LOG.config:           table_names:
        LOG.config:             - samples.nyctaxi.trips
        LOG.config:           condition: ANY_UPDATED
        LOG.config:           wait_after_last_change_seconds: 60
    acceptance_test.go:1035: LOG.cp: 
    acceptance_test.go:1035: 
        LOG.deploy: 
        LOG.deploy: >>> /var/folders/5y/9kkdnjw91p11vsqwk0cvmk200000gp/T/regression_main_cli_013rzbtg/src/databricks bundle deploy
        LOG.deploy: Uploading bundle files to /Workspace/Users/tester@databricks.com/.bundle/test-bundle-7qu6zonevnaltdicqxxraw3xqm/default/files...
        LOG.deploy: Created jobs.foo
        LOG.deploy: Files: 15 uploaded, 0 deleted
        LOG.deploy: Resources: 1 created, 0 changed, 0 deleted, 0 unchanged
    acceptance_test.go:1035: 
        LOG.destroy: 
        LOG.destroy: >>> /var/folders/5y/9kkdnjw91p11vsqwk0cvmk200000gp/T/regression_main_cli_013rzbtg/src/databricks bundle destroy --auto-approve
        LOG.destroy: The following resources will be deleted:
        LOG.destroy:   delete resources.jobs.foo
        LOG.destroy: 
        LOG.destroy: All files and directories at the following location will be deleted: /Workspace/Users/tester@databricks.com/.bundle/test-bundle-7qu6zonevnaltdicqxxraw3xqm/default
        LOG.destroy: 
        LOG.destroy: Destroy: 1 deleted
    acceptance_test.go:1035: 
        LOG.planjson: {
        LOG.planjson:   "plan_version": 2,
        LOG.planjson:   "cli_version": "1.15.0-dev",
        LOG.planjson:   "lineage": "80b8ed8b-878f-476f-b68c-935e5c2b8aed",
        LOG.planjson:   "serial": 1,
        LOG.planjson:   "plan": {
        LOG.planjson:     "resources.jobs.foo": {
        LOG.planjson:       "action": "update",
        LOG.planjson:       "new_state": {
        LOG.planjson:         "value": {
        LOG.planjson:           "deployment": {
        LOG.planjson:             "kind": "BUNDLE",
        LOG.planjson:             "metadata_file_path": "/Workspace/Users/tester@databricks.com/.bundle/test-bundle-7qu6zonevnaltdicqxxraw3xqm/default/state/metadata.json"
        LOG.planjson:           },
        LOG.planjson:           "edit_mode": "UI_LOCKED",
        LOG.planjson:           "format": "MULTI_TASK",
        LOG.planjson:           "max_concurrent_runs": 1,
        LOG.planjson:           "name": "test-job-7qu6zonevnaltdicqxxraw3xqm",
        LOG.planjson:           "queue": {
        LOG.planjson:             "enabled": true
        LOG.planjson:           },
        LOG.planjson:           "trigger": {
        LOG.planjson:             "pause_status": "UNPAUSED",
        LOG.planjson:             "table_update": {
        LOG.planjson:               "condition": "ANY_UPDATED",
        LOG.planjson:               "table_names": [
        LOG.planjson:                 "samples.nyctaxi.trips"
        LOG.planjson:               ],
        LOG.planjson:               "wait_after_last_change_seconds": 60
        LOG.planjson:             }
        LOG.planjson:           }
        LOG.planjson:         }
        LOG.planjson:       },
        LOG.planjson:       "remote_state": {
        LOG.planjson:         "created_time": 1788175196853,
        LOG.planjson:         "creator_user_name": "tester@databricks.com",
        LOG.planjson:         "deployment": {
        LOG.planjson:           "kind": "BUNDLE",
        LOG.planjson:           "metadata_file_path": "/Workspace/Users/tester@databricks.com/.bundle/test-bundle-7qu6zonevnaltdicqxxraw3xqm/default/state/metadata.json"
        LOG.planjson:         },
        LOG.planjson:         "edit_mode": "UI_LOCKED",
        LOG.planjson:         "email_notifications": {},
        LOG.planjson:         "format": "MULTI_TASK",
        LOG.planjson:         "job_id": 8788175196853363000,
        LOG.planjson:         "max_concurrent_runs": 1,
        LOG.planjson:         "name": "test-job-7qu6zonevnaltdicqxxraw3xqm",
        LOG.planjson:         "queue": {
        LOG.planjson:           "enabled": true
        LOG.planjson:         },
        LOG.planjson:         "run_as_user_name": "tester@databricks.com",
        LOG.planjson:         "timeout_seconds": 0,
        LOG.planjson:         "trigger": {
        LOG.planjson:           "pause_status": "UNPAUSED",
        LOG.planjson:           "table_update": {
        LOG.planjson:             "condition": "",
        LOG.planjson:             "table_names": [
        LOG.planjson:               "samples.nyctaxi.trips"
        LOG.planjson:             ],
        LOG.planjson:             "wait_after_last_change_seconds": 60
        LOG.planjson:           }
        LOG.planjson:         },
        LOG.planjson:         "webhook_notifications": {}
        LOG.planjson:       },
        LOG.planjson:       "changes": {
        LOG.planjson:         "email_notifications": {
        LOG.planjson:           "action": "skip",
        LOG.planjson:           "reason": "empty",
        LOG.planjson:           "remote": {}
        LOG.planjson:         },
        LOG.planjson:         "timeout_seconds": {
        LOG.planjson:           "action": "skip",
        LOG.planjson:           "reason": "empty",
        LOG.planjson:           "remote": 0
        LOG.planjson:         },
        LOG.planjson:         "trigger.table_update.condition": {
        LOG.planjson:           "action": "update",
        LOG.planjson:           "old": "ANY_UPDATED",
        LOG.planjson:           "new": "ANY_UPDATED",
        LOG.planjson:           "remote": ""
        LOG.planjson:         },
        LOG.planjson:         "webhook_notifications": {
        LOG.planjson:           "action": "skip",
        LOG.planjson:           "reason": "empty",
        LOG.planjson:           "remote": {}
        LOG.planjson:         }
        LOG.planjson:       }
        LOG.planjson:     }
        LOG.planjson:   }
        LOG.planjson: }
--- FAIL: TestAccept/bundle/invariant/no_drift/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl/READPLAN= (0.27s)
```

**latest (v1.14.1):**
```
=== RUN   TestAccept/bundle/invariant/no_drift/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl/READPLAN=
=== PAUSE TestAccept/bundle/invariant/no_drift/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl/READPLAN=
=== CONT  TestAccept/bundle/invariant/no_drift/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl/READPLAN=
    acceptance_test.go:1176: Diff:
        --- bundle/invariant/no_drift/output.txt
        +++ /var/folders/5y/9kkdnjw91p11vsqwk0cvmk200000gp/T/TestAcceptbundleinvariantno_driftDATABRICKS_BUNDLE_ENGINE=di1490066650/001/output.txt
        @@ -1 +1,94 @@
         INPUT_CONFIG_OK
        +Unexpected action='update' for resources.jobs.foo
        +{
        +  "plan_version": 2,
        +  "cli_version": "[CLI_VERSION]",
        +  "lineage": "[UUID]",
        +  "serial": 1,
        +  "plan": {
        +    "resources.jobs.foo": {
        +      "action": "update",
        +      "new_state": {
        +        "value": {
        +          "deployment": {
        +            "kind": "BUNDLE",
        +            "metadata_file_path": "/Workspace/Users/[USERNAME]/.bundle/test-bundle-[UNIQUE_NAME]/default/state/metadata.json"
        +          },
        +          "edit_mode": "UI_LOCKED",
        +          "format": "MULTI_TASK",
        +          "max_concurrent_runs": 1,
        +          "name": "test-job-[UNIQUE_NAME]",
        +          "queue": {
        +            "enabled": true
        +          },
        +          "trigger": {
        +            "pause_status": "UNPAUSED",
        +            "table_update": {
        +              "condition": "ANY_UPDATED",
        +              "table_names": [
        +                "samples.nyctaxi.trips"
        +              ],
        +              "wait_after_last_change_seconds": 60
        +            }
        +          }
        +        }
        +      },
        +      "remote_state": {
        +        "created_time": [UNIX_TIME_MILLIS],
        +        "creator_user_name": "[USERNAME]",
        +        "deployment": {
        +          "kind": "BUNDLE",
        +          "metadata_file_path": "/Workspace/Users/[USERNAME]/.bundle/test-bundle-[UNIQUE_NAME]/default/state/metadata.json"
        +        },
        +        "edit_mode": "UI_LOCKED",
        +        "email_notifications": {},
        +        "format": "MULTI_TASK",
        +        "job_id": [NUMID],
        +        "max_concurrent_runs": 1,
        +        "name": "test-job-[UNIQUE_NAME]",
        +        "queue": {
        +          "enabled": true
        +        },
        +        "run_as_user_name": "[USERNAME]",
        +        "timeout_seconds": 0,
        +        "trigger": {
        +          "pause_status": "UNPAUSED",
        +          "table_update": {
        +            "condition": "",
        +            "table_names": [
        +              "samples.nyctaxi.trips"
        +            ],
        +            "wait_after_last_change_seconds": 60
        +          }
        +        },
        +        "webhook_notifications": {}
        +      },
        +      "changes": {
        +        "email_notifications": {
        +          "action": "skip",
        +          "reason": "empty",
        +          "remote": {}
        +        },
        +        "timeout_seconds": {
        +          "action": "skip",
        +          "reason": "empty",
        +          "remote": 0
        +        },
        +        "trigger.table_update.condition": {
        +          "action": "update",
        +          "old": "ANY_UPDATED",
        +          "new": "ANY_UPDATED",
        +          "remote": ""
        +        },
        +        "webhook_notifications": {
        +          "action": "skip",
        +          "reason": "empty",
        +          "remote": {}
        +        }
        +      }
        +    }
        +  }
        +}
        +
        +
        +Exit code: 10
        
    acceptance_test.go:1035: 
        LOG.config: bundle:
        LOG.config:   name: test-bundle-symimgd46zaorobme2i2ugpy6u
        LOG.config: 
        LOG.config: resources:
        LOG.config:   jobs:
        LOG.config:     foo:
        LOG.config:       name: test-job-symimgd46zaorobme2i2ugpy6u
        LOG.config:       trigger:
        LOG.config:         table_update:
        LOG.config:           table_names:
        LOG.config:             - samples.nyctaxi.trips
        LOG.config:           condition: ANY_UPDATED
        LOG.config:           wait_after_last_change_seconds: 60
    acceptance_test.go:1035: LOG.cp: 
    acceptance_test.go:1035: 
        LOG.deploy: 
        LOG.deploy: >>> /Users/denis.bilenko/work/cli-trees/investigate-6315/acceptance/build/darwin_arm64/1.14.1/databricks bundle deploy
        LOG.deploy: Uploading bundle files to /Workspace/Users/tester@databricks.com/.bundle/test-bundle-symimgd46zaorobme2i2ugpy6u/default/files...
        LOG.deploy: Created jobs.foo
        LOG.deploy: Files: 15 uploaded, 0 deleted
        LOG.deploy: Resources: 1 created, 0 changed, 0 deleted, 0 unchanged
    acceptance_test.go:1035: 
        LOG.destroy: 
        LOG.destroy: >>> /Users/denis.bilenko/work/cli-trees/investigate-6315/acceptance/build/darwin_arm64/1.14.1/databricks bundle destroy --auto-approve
        LOG.destroy: The following resources will be deleted:
        LOG.destroy:   delete resources.jobs.foo
        LOG.destroy: 
        LOG.destroy: All files and directories at the following location will be deleted: /Workspace/Users/tester@databricks.com/.bundle/test-bundle-symimgd46zaorobme2i2ugpy6u/default
        LOG.destroy: 
        LOG.destroy: Destroy: 1 deleted
    acceptance_test.go:1035: 
        LOG.planjson: {
        LOG.planjson:   "plan_version": 2,
        LOG.planjson:   "cli_version": "1.14.1",
        LOG.planjson:   "lineage": "b7b556fd-7d95-4e56-bc49-657b8a3e19c6",
        LOG.planjson:   "serial": 1,
        LOG.planjson:   "plan": {
        LOG.planjson:     "resources.jobs.foo": {
        LOG.planjson:       "action": "update",
        LOG.planjson:       "new_state": {
        LOG.planjson:         "value": {
        LOG.planjson:           "deployment": {
        LOG.planjson:             "kind": "BUNDLE",
        LOG.planjson:             "metadata_file_path": "/Workspace/Users/tester@databricks.com/.bundle/test-bundle-symimgd46zaorobme2i2ugpy6u/default/state/metadata.json"
        LOG.planjson:           },
        LOG.planjson:           "edit_mode": "UI_LOCKED",
        LOG.planjson:           "format": "MULTI_TASK",
        LOG.planjson:           "max_concurrent_runs": 1,
        LOG.planjson:           "name": "test-job-symimgd46zaorobme2i2ugpy6u",
        LOG.planjson:           "queue": {
        LOG.planjson:             "enabled": true
        LOG.planjson:           },
        LOG.planjson:           "trigger": {
        LOG.planjson:             "pause_status": "UNPAUSED",
        LOG.planjson:             "table_update": {
        LOG.planjson:               "condition": "ANY_UPDATED",
        LOG.planjson:               "table_names": [
        LOG.planjson:                 "samples.nyctaxi.trips"
        LOG.planjson:               ],
        LOG.planjson:               "wait_after_last_change_seconds": 60
        LOG.planjson:             }
        LOG.planjson:           }
        LOG.planjson:         }
        LOG.planjson:       },
        LOG.planjson:       "remote_state": {
        LOG.planjson:         "created_time": 1788175233388,
        LOG.planjson:         "creator_user_name": "tester@databricks.com",
        LOG.planjson:         "deployment": {
        LOG.planjson:           "kind": "BUNDLE",
        LOG.planjson:           "metadata_file_path": "/Workspace/Users/tester@databricks.com/.bundle/test-bundle-symimgd46zaorobme2i2ugpy6u/default/state/metadata.json"
        LOG.planjson:         },
        LOG.planjson:         "edit_mode": "UI_LOCKED",
        LOG.planjson:         "email_notifications": {},
        LOG.planjson:         "format": "MULTI_TASK",
        LOG.planjson:         "job_id": 8788175233387903000,
        LOG.planjson:         "max_concurrent_runs": 1,
        LOG.planjson:         "name": "test-job-symimgd46zaorobme2i2ugpy6u",
        LOG.planjson:         "queue": {
        LOG.planjson:           "enabled": true
        LOG.planjson:         },
        LOG.planjson:         "run_as_user_name": "tester@databricks.com",
        LOG.planjson:         "timeout_seconds": 0,
        LOG.planjson:         "trigger": {
        LOG.planjson:           "pause_status": "UNPAUSED",
        LOG.planjson:           "table_update": {
        LOG.planjson:             "condition": "",
        LOG.planjson:             "table_names": [
        LOG.planjson:               "samples.nyctaxi.trips"
        LOG.planjson:             ],
        LOG.planjson:             "wait_after_last_change_seconds": 60
        LOG.planjson:           }
        LOG.planjson:         },
        LOG.planjson:         "webhook_notifications": {}
        LOG.planjson:       },
        LOG.planjson:       "changes": {
        LOG.planjson:         "email_notifications": {
        LOG.planjson:           "action": "skip",
        LOG.planjson:           "reason": "empty",
        LOG.planjson:           "remote": {}
        LOG.planjson:         },
        LOG.planjson:         "timeout_seconds": {
        LOG.planjson:           "action": "skip",
        LOG.planjson:           "reason": "empty",
        LOG.planjson:           "remote": 0
        LOG.planjson:         },
        LOG.planjson:         "trigger.table_update.condition": {
        LOG.planjson:           "action": "update",
        LOG.planjson:           "old": "ANY_UPDATED",
        LOG.planjson:           "new": "ANY_UPDATED",
        LOG.planjson:           "remote": ""
        LOG.planjson:         },
        LOG.planjson:         "webhook_notifications": {
        LOG.planjson:           "action": "skip",
        LOG.planjson:           "reason": "empty",
        LOG.planjson:           "remote": {}
        LOG.planjson:         }
        LOG.planjson:       }
        LOG.planjson:     }
        LOG.planjson:   }
        LOG.planjson: }
--- FAIL: TestAccept/bundle/invariant/no_drift/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl/READPLAN= (0.26s)
```

</details>

<details>
<summary>TestAccept/bundle/invariant/no_drift/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl/READPLAN=1 ✅ | main (0a8aae1655) ❌ | latest (v1.14.1) ❌</summary>

**main (0a8aae1655):**
```
=== RUN   TestAccept/bundle/invariant/no_drift/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl/READPLAN=1
=== PAUSE TestAccept/bundle/invariant/no_drift/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl/READPLAN=1
=== CONT  TestAccept/bundle/invariant/no_drift/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl/READPLAN=1
    acceptance_test.go:1176: Diff:
        --- bundle/invariant/no_drift/output.txt
        +++ /var/folders/5y/9kkdnjw91p11vsqwk0cvmk200000gp/T/TestAcceptbundleinvariantno_driftDATABRICKS_BUNDLE_ENGINE=di1668172078/001/output.txt
        @@ -1 +1,94 @@
         INPUT_CONFIG_OK
        +Unexpected action='update' for resources.jobs.foo
        +{
        +  "plan_version": 2,
        +  "cli_version": "[CLI_VERSION]",
        +  "lineage": "[UUID]",
        +  "serial": 1,
        +  "plan": {
        +    "resources.jobs.foo": {
        +      "action": "update",
        +      "new_state": {
        +        "value": {
        +          "deployment": {
        +            "kind": "BUNDLE",
        +            "metadata_file_path": "/Workspace/Users/[USERNAME]/.bundle/test-bundle-[UNIQUE_NAME]/default/state/metadata.json"
        +          },
        +          "edit_mode": "UI_LOCKED",
        +          "format": "MULTI_TASK",
        +          "max_concurrent_runs": 1,
        +          "name": "test-job-[UNIQUE_NAME]",
        +          "queue": {
        +            "enabled": true
        +          },
        +          "trigger": {
        +            "pause_status": "UNPAUSED",
        +            "table_update": {
        +              "condition": "ANY_UPDATED",
        +              "table_names": [
        +                "samples.nyctaxi.trips"
        +              ],
        +              "wait_after_last_change_seconds": 60
        +            }
        +          }
        +        }
        +      },
        +      "remote_state": {
        +        "created_time": [UNIX_TIME_MILLIS],
        +        "creator_user_name": "[USERNAME]",
        +        "deployment": {
        +          "kind": "BUNDLE",
        +          "metadata_file_path": "/Workspace/Users/[USERNAME]/.bundle/test-bundle-[UNIQUE_NAME]/default/state/metadata.json"
        +        },
        +        "edit_mode": "UI_LOCKED",
        +        "email_notifications": {},
        +        "format": "MULTI_TASK",
        +        "job_id": [NUMID],
        +        "max_concurrent_runs": 1,
        +        "name": "test-job-[UNIQUE_NAME]",
        +        "queue": {
        +          "enabled": true
        +        },
        +        "run_as_user_name": "[USERNAME]",
        +        "timeout_seconds": 0,
        +        "trigger": {
        +          "pause_status": "UNPAUSED",
        +          "table_update": {
        +            "condition": "",
        +            "table_names": [
        +              "samples.nyctaxi.trips"
        +            ],
        +            "wait_after_last_change_seconds": 60
        +          }
        +        },
        +        "webhook_notifications": {}
        +      },
        +      "changes": {
        +        "email_notifications": {
        +          "action": "skip",
        +          "reason": "empty",
        +          "remote": {}
        +        },
        +        "timeout_seconds": {
        +          "action": "skip",
        +          "reason": "empty",
        +          "remote": 0
        +        },
        +        "trigger.table_update.condition": {
        +          "action": "update",
        +          "old": "ANY_UPDATED",
        +          "new": "ANY_UPDATED",
        +          "remote": ""
        +        },
        +        "webhook_notifications": {
        +          "action": "skip",
        +          "reason": "empty",
        +          "remote": {}
        +        }
        +      }
        +    }
        +  }
        +}
        +
        +
        +Exit code: 10
        
    acceptance_test.go:1035: 
        LOG.config: bundle:
        LOG.config:   name: test-bundle-52kiljwburd5dcbdyldxxepmve
        LOG.config: 
        LOG.config: resources:
        LOG.config:   jobs:
        LOG.config:     foo:
        LOG.config:       name: test-job-52kiljwburd5dcbdyldxxepmve
        LOG.config:       trigger:
        LOG.config:         table_update:
        LOG.config:           table_names:
        LOG.config:             - samples.nyctaxi.trips
        LOG.config:           condition: ANY_UPDATED
        LOG.config:           wait_after_last_change_seconds: 60
    acceptance_test.go:1035: LOG.cp: 
    acceptance_test.go:1035: 
        LOG.deploy: 
        LOG.deploy: >>> /var/folders/5y/9kkdnjw91p11vsqwk0cvmk200000gp/T/regression_main_cli_013rzbtg/src/databricks bundle deploy --plan plan.json
        LOG.deploy: Uploading bundle files to /Workspace/Users/tester@databricks.com/.bundle/test-bundle-52kiljwburd5dcbdyldxxepmve/default/files...
        LOG.deploy: Created jobs.foo
        LOG.deploy: Files: 17 uploaded, 0 deleted
        LOG.deploy: Resources: 1 created, 0 changed, 0 deleted, 0 unchanged
    acceptance_test.go:1035: 
        LOG.destroy: 
        LOG.destroy: >>> /var/folders/5y/9kkdnjw91p11vsqwk0cvmk200000gp/T/regression_main_cli_013rzbtg/src/databricks bundle destroy --auto-approve
        LOG.destroy: The following resources will be deleted:
        LOG.destroy:   delete resources.jobs.foo
        LOG.destroy: 
        LOG.destroy: All files and directories at the following location will be deleted: /Workspace/Users/tester@databricks.com/.bundle/test-bundle-52kiljwburd5dcbdyldxxepmve/default
        LOG.destroy: 
        LOG.destroy: Destroy: 1 deleted
    acceptance_test.go:1035: 
        LOG.planjson: {
        LOG.planjson:   "plan_version": 2,
        LOG.planjson:   "cli_version": "1.15.0-dev",
        LOG.planjson:   "lineage": "323a16fb-1aef-4aaa-bc84-ca364e15ec51",
        LOG.planjson:   "serial": 1,
        LOG.planjson:   "plan": {
        LOG.planjson:     "resources.jobs.foo": {
        LOG.planjson:       "action": "update",
        LOG.planjson:       "new_state": {
        LOG.planjson:         "value": {
        LOG.planjson:           "deployment": {
        LOG.planjson:             "kind": "BUNDLE",
        LOG.planjson:             "metadata_file_path": "/Workspace/Users/tester@databricks.com/.bundle/test-bundle-52kiljwburd5dcbdyldxxepmve/default/state/metadata.json"
        LOG.planjson:           },
        LOG.planjson:           "edit_mode": "UI_LOCKED",
        LOG.planjson:           "format": "MULTI_TASK",
        LOG.planjson:           "max_concurrent_runs": 1,
        LOG.planjson:           "name": "test-job-52kiljwburd5dcbdyldxxepmve",
        LOG.planjson:           "queue": {
        LOG.planjson:             "enabled": true
        LOG.planjson:           },
        LOG.planjson:           "trigger": {
        LOG.planjson:             "pause_status": "UNPAUSED",
        LOG.planjson:             "table_update": {
        LOG.planjson:               "condition": "ANY_UPDATED",
        LOG.planjson:               "table_names": [
        LOG.planjson:                 "samples.nyctaxi.trips"
        LOG.planjson:               ],
        LOG.planjson:               "wait_after_last_change_seconds": 60
        LOG.planjson:             }
        LOG.planjson:           }
        LOG.planjson:         }
        LOG.planjson:       },
        LOG.planjson:       "remote_state": {
        LOG.planjson:         "created_time": 1788175200383,
        LOG.planjson:         "creator_user_name": "tester@databricks.com",
        LOG.planjson:         "deployment": {
        LOG.planjson:           "kind": "BUNDLE",
        LOG.planjson:           "metadata_file_path": "/Workspace/Users/tester@databricks.com/.bundle/test-bundle-52kiljwburd5dcbdyldxxepmve/default/state/metadata.json"
        LOG.planjson:         },
        LOG.planjson:         "edit_mode": "UI_LOCKED",
        LOG.planjson:         "email_notifications": {},
        LOG.planjson:         "format": "MULTI_TASK",
        LOG.planjson:         "job_id": 8788175200383320000,
        LOG.planjson:         "max_concurrent_runs": 1,
        LOG.planjson:         "name": "test-job-52kiljwburd5dcbdyldxxepmve",
        LOG.planjson:         "queue": {
        LOG.planjson:           "enabled": true
        LOG.planjson:         },
        LOG.planjson:         "run_as_user_name": "tester@databricks.com",
        LOG.planjson:         "timeout_seconds": 0,
        LOG.planjson:         "trigger": {
        LOG.planjson:           "pause_status": "UNPAUSED",
        LOG.planjson:           "table_update": {
        LOG.planjson:             "condition": "",
        LOG.planjson:             "table_names": [
        LOG.planjson:               "samples.nyctaxi.trips"
        LOG.planjson:             ],
        LOG.planjson:             "wait_after_last_change_seconds": 60
        LOG.planjson:           }
        LOG.planjson:         },
        LOG.planjson:         "webhook_notifications": {}
        LOG.planjson:       },
        LOG.planjson:       "changes": {
        LOG.planjson:         "email_notifications": {
        LOG.planjson:           "action": "skip",
        LOG.planjson:           "reason": "empty",
        LOG.planjson:           "remote": {}
        LOG.planjson:         },
        LOG.planjson:         "timeout_seconds": {
        LOG.planjson:           "action": "skip",
        LOG.planjson:           "reason": "empty",
        LOG.planjson:           "remote": 0
        LOG.planjson:         },
        LOG.planjson:         "trigger.table_update.condition": {
        LOG.planjson:           "action": "update",
        LOG.planjson:           "old": "ANY_UPDATED",
        LOG.planjson:           "new": "ANY_UPDATED",
        LOG.planjson:           "remote": ""
        LOG.planjson:         },
        LOG.planjson:         "webhook_notifications": {
        LOG.planjson:           "action": "skip",
        LOG.planjson:           "reason": "empty",
        LOG.planjson:           "remote": {}
        LOG.planjson:         }
        LOG.planjson:       }
        LOG.planjson:     }
        LOG.planjson:   }
        LOG.planjson: }
--- FAIL: TestAccept/bundle/invariant/no_drift/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl/READPLAN=1 (0.32s)
```

**latest (v1.14.1):**
```
=== RUN   TestAccept/bundle/invariant/no_drift/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl/READPLAN=1
=== PAUSE TestAccept/bundle/invariant/no_drift/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl/READPLAN=1
=== CONT  TestAccept/bundle/invariant/no_drift/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl/READPLAN=1
    acceptance_test.go:1176: Diff:
        --- bundle/invariant/no_drift/output.txt
        +++ /var/folders/5y/9kkdnjw91p11vsqwk0cvmk200000gp/T/TestAcceptbundleinvariantno_driftDATABRICKS_BUNDLE_ENGINE=di1058852012/001/output.txt
        @@ -1 +1,94 @@
         INPUT_CONFIG_OK
        +Unexpected action='update' for resources.jobs.foo
        +{
        +  "plan_version": 2,
        +  "cli_version": "[CLI_VERSION]",
        +  "lineage": "[UUID]",
        +  "serial": 1,
        +  "plan": {
        +    "resources.jobs.foo": {
        +      "action": "update",
        +      "new_state": {
        +        "value": {
        +          "deployment": {
        +            "kind": "BUNDLE",
        +            "metadata_file_path": "/Workspace/Users/[USERNAME]/.bundle/test-bundle-[UNIQUE_NAME]/default/state/metadata.json"
        +          },
        +          "edit_mode": "UI_LOCKED",
        +          "format": "MULTI_TASK",
        +          "max_concurrent_runs": 1,
        +          "name": "test-job-[UNIQUE_NAME]",
        +          "queue": {
        +            "enabled": true
        +          },
        +          "trigger": {
        +            "pause_status": "UNPAUSED",
        +            "table_update": {
        +              "condition": "ANY_UPDATED",
        +              "table_names": [
        +                "samples.nyctaxi.trips"
        +              ],
        +              "wait_after_last_change_seconds": 60
        +            }
        +          }
        +        }
        +      },
        +      "remote_state": {
        +        "created_time": [UNIX_TIME_MILLIS],
        +        "creator_user_name": "[USERNAME]",
        +        "deployment": {
        +          "kind": "BUNDLE",
        +          "metadata_file_path": "/Workspace/Users/[USERNAME]/.bundle/test-bundle-[UNIQUE_NAME]/default/state/metadata.json"
        +        },
        +        "edit_mode": "UI_LOCKED",
        +        "email_notifications": {},
        +        "format": "MULTI_TASK",
        +        "job_id": [NUMID],
        +        "max_concurrent_runs": 1,
        +        "name": "test-job-[UNIQUE_NAME]",
        +        "queue": {
        +          "enabled": true
        +        },
        +        "run_as_user_name": "[USERNAME]",
        +        "timeout_seconds": 0,
        +        "trigger": {
        +          "pause_status": "UNPAUSED",
        +          "table_update": {
        +            "condition": "",
        +            "table_names": [
        +              "samples.nyctaxi.trips"
        +            ],
        +            "wait_after_last_change_seconds": 60
        +          }
        +        },
        +        "webhook_notifications": {}
        +      },
        +      "changes": {
        +        "email_notifications": {
        +          "action": "skip",
        +          "reason": "empty",
        +          "remote": {}
        +        },
        +        "timeout_seconds": {
        +          "action": "skip",
        +          "reason": "empty",
        +          "remote": 0
        +        },
        +        "trigger.table_update.condition": {
        +          "action": "update",
        +          "old": "ANY_UPDATED",
        +          "new": "ANY_UPDATED",
        +          "remote": ""
        +        },
        +        "webhook_notifications": {
        +          "action": "skip",
        +          "reason": "empty",
        +          "remote": {}
        +        }
        +      }
        +    }
        +  }
        +}
        +
        +
        +Exit code: 10
        
    acceptance_test.go:1035: 
        LOG.config: bundle:
        LOG.config:   name: test-bundle-znxdqaxm3jcyvok3p3hxq6rp7e
        LOG.config: 
        LOG.config: resources:
        LOG.config:   jobs:
        LOG.config:     foo:
        LOG.config:       name: test-job-znxdqaxm3jcyvok3p3hxq6rp7e
        LOG.config:       trigger:
        LOG.config:         table_update:
        LOG.config:           table_names:
        LOG.config:             - samples.nyctaxi.trips
        LOG.config:           condition: ANY_UPDATED
        LOG.config:           wait_after_last_change_seconds: 60
    acceptance_test.go:1035: LOG.cp: 
    acceptance_test.go:1035: 
        LOG.deploy: 
        LOG.deploy: >>> /Users/denis.bilenko/work/cli-trees/investigate-6315/acceptance/build/darwin_arm64/1.14.1/databricks bundle deploy --plan plan.json
        LOG.deploy: Uploading bundle files to /Workspace/Users/tester@databricks.com/.bundle/test-bundle-znxdqaxm3jcyvok3p3hxq6rp7e/default/files...
        LOG.deploy: Created jobs.foo
        LOG.deploy: Files: 17 uploaded, 0 deleted
        LOG.deploy: Resources: 1 created, 0 changed, 0 deleted, 0 unchanged
    acceptance_test.go:1035: 
        LOG.destroy: 
        LOG.destroy: >>> /Users/denis.bilenko/work/cli-trees/investigate-6315/acceptance/build/darwin_arm64/1.14.1/databricks bundle destroy --auto-approve
        LOG.destroy: The following resources will be deleted:
        LOG.destroy:   delete resources.jobs.foo
        LOG.destroy: 
        LOG.destroy: All files and directories at the following location will be deleted: /Workspace/Users/tester@databricks.com/.bundle/test-bundle-znxdqaxm3jcyvok3p3hxq6rp7e/default
        LOG.destroy: 
        LOG.destroy: Destroy: 1 deleted
    acceptance_test.go:1035: 
        LOG.planjson: {
        LOG.planjson:   "plan_version": 2,
        LOG.planjson:   "cli_version": "1.14.1",
        LOG.planjson:   "lineage": "49560766-3eb5-4845-9041-3db9e44e62e6",
        LOG.planjson:   "serial": 1,
        LOG.planjson:   "plan": {
        LOG.planjson:     "resources.jobs.foo": {
        LOG.planjson:       "action": "update",
        LOG.planjson:       "new_state": {
        LOG.planjson:         "value": {
        LOG.planjson:           "deployment": {
        LOG.planjson:             "kind": "BUNDLE",
        LOG.planjson:             "metadata_file_path": "/Workspace/Users/tester@databricks.com/.bundle/test-bundle-znxdqaxm3jcyvok3p3hxq6rp7e/default/state/metadata.json"
        LOG.planjson:           },
        LOG.planjson:           "edit_mode": "UI_LOCKED",
        LOG.planjson:           "format": "MULTI_TASK",
        LOG.planjson:           "max_concurrent_runs": 1,
        LOG.planjson:           "name": "test-job-znxdqaxm3jcyvok3p3hxq6rp7e",
        LOG.planjson:           "queue": {
        LOG.planjson:             "enabled": true
        LOG.planjson:           },
        LOG.planjson:           "trigger": {
        LOG.planjson:             "pause_status": "UNPAUSED",
        LOG.planjson:             "table_update": {
        LOG.planjson:               "condition": "ANY_UPDATED",
        LOG.planjson:               "table_names": [
        LOG.planjson:                 "samples.nyctaxi.trips"
        LOG.planjson:               ],
        LOG.planjson:               "wait_after_last_change_seconds": 60
        LOG.planjson:             }
        LOG.planjson:           }
        LOG.planjson:         }
        LOG.planjson:       },
        LOG.planjson:       "remote_state": {
        LOG.planjson:         "created_time": 1788175237043,
        LOG.planjson:         "creator_user_name": "tester@databricks.com",
        LOG.planjson:         "deployment": {
        LOG.planjson:           "kind": "BUNDLE",
        LOG.planjson:           "metadata_file_path": "/Workspace/Users/tester@databricks.com/.bundle/test-bundle-znxdqaxm3jcyvok3p3hxq6rp7e/default/state/metadata.json"
        LOG.planjson:         },
        LOG.planjson:         "edit_mode": "UI_LOCKED",
        LOG.planjson:         "email_notifications": {},
        LOG.planjson:         "format": "MULTI_TASK",
        LOG.planjson:         "job_id": 8788175237043371000,
        LOG.planjson:         "max_concurrent_runs": 1,
        LOG.planjson:         "name": "test-job-znxdqaxm3jcyvok3p3hxq6rp7e",
        LOG.planjson:         "queue": {
        LOG.planjson:           "enabled": true
        LOG.planjson:         },
        LOG.planjson:         "run_as_user_name": "tester@databricks.com",
        LOG.planjson:         "timeout_seconds": 0,
        LOG.planjson:         "trigger": {
        LOG.planjson:           "pause_status": "UNPAUSED",
        LOG.planjson:           "table_update": {
        LOG.planjson:             "condition": "",
        LOG.planjson:             "table_names": [
        LOG.planjson:               "samples.nyctaxi.trips"
        LOG.planjson:             ],
        LOG.planjson:             "wait_after_last_change_seconds": 60
        LOG.planjson:           }
        LOG.planjson:         },
        LOG.planjson:         "webhook_notifications": {}
        LOG.planjson:       },
        LOG.planjson:       "changes": {
        LOG.planjson:         "email_notifications": {
        LOG.planjson:           "action": "skip",
        LOG.planjson:           "reason": "empty",
        LOG.planjson:           "remote": {}
        LOG.planjson:         },
        LOG.planjson:         "timeout_seconds": {
        LOG.planjson:           "action": "skip",
        LOG.planjson:           "reason": "empty",
        LOG.planjson:           "remote": 0
        LOG.planjson:         },
        LOG.planjson:         "trigger.table_update.condition": {
        LOG.planjson:           "action": "update",
        LOG.planjson:           "old": "ANY_UPDATED",
        LOG.planjson:           "new": "ANY_UPDATED",
        LOG.planjson:           "remote": ""
        LOG.planjson:         },
        LOG.planjson:         "webhook_notifications": {
        LOG.planjson:           "action": "skip",
        LOG.planjson:           "reason": "empty",
        LOG.planjson:           "remote": {}
        LOG.planjson:         }
        LOG.planjson:       }
        LOG.planjson:     }
        LOG.planjson:   }
        LOG.planjson: }
--- FAIL: TestAccept/bundle/invariant/no_drift/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job_table_update_trigger.yml.tmpl/READPLAN=1 (0.32s)
```

</details>

