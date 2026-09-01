# Regression Test Report

Tested commit: 0308da23f Shorten the changelog entry and link the PR

<!-- Acceptance tests: 2 (1 added, 1 modified) — 2 regression -->

| test | branch | main (0a8aae165) | latest |
| --- | --- | --- | --- |
| TestAccept/bundle/resources/postgres_projects/update_default_endpoint_autoscaling/DATABRICKS_BUNDLE_ENGINE=direct | ✅ | ❌ | ➖ |
| TestAccept/bundle/resources/postgres_projects/update_default_endpoint_autoscaling/DATABRICKS_BUNDLE_ENGINE=terraform | ✅ | ✅ | ➖ |
| TestAccept/bundle/resources/postgres_projects/update_default_endpoint_suspend/DATABRICKS_BUNDLE_ENGINE=direct | ✅ | ❌ | ➖ |
| TestAccept/bundle/resources/postgres_projects/update_default_endpoint_suspend/DATABRICKS_BUNDLE_ENGINE=terraform | ✅ | ✅ | ➖ |

<details>
<summary>TestAccept/bundle/resources/postgres_projects/update_default_endpoint_autoscaling/DATABRICKS_BUNDLE_ENGINE=direct ✅ | main (0a8aae165) ❌ | latest ➖</summary>

**main (0a8aae165):**
```
=== RUN   TestAccept/bundle/resources/postgres_projects/update_default_endpoint_autoscaling/DATABRICKS_BUNDLE_ENGINE=direct
=== PAUSE TestAccept/bundle/resources/postgres_projects/update_default_endpoint_autoscaling/DATABRICKS_BUNDLE_ENGINE=direct
=== CONT  TestAccept/bundle/resources/postgres_projects/update_default_endpoint_autoscaling/DATABRICKS_BUNDLE_ENGINE=direct
    acceptance_test.go:1176: Diff:
        --- bundle/resources/postgres_projects/update_default_endpoint_autoscaling/output.txt
        +++ /var/folders/5y/9kkdnjw91p11vsqwk0cvmk200000gp/T/TestAcceptbundleresourcespostgres_projectsupdate_default_end1384276257/001/output.txt
        @@ -16,23 +16,15 @@
         
         >>> [CLI] bundle deploy
         Uploading bundle files to /Workspace/Users/[USERNAME]/.bundle/test-bundle-[UNIQUE_NAME]/default/files...
        -Updated postgres_projects.my_project
        -Files: 0 uploaded, 0 deleted
        -Resources: 0 created, 1 changed, 0 deleted, 0 unchanged
        +Error: cannot update resources.postgres_projects.my_project: updating id=projects/test-pg-proj-[UNIQUE_NAME]: Field 'spec.default_endpoint_settings.suspension' is in update_mask but not provided in request (400 INVALID_PARAMETER_VALUE)
         
        ->>> print_requests.py --del-body project_id --sort //postgres ^//workspace-files/ ^//workspace/ ^//telemetry-ext ^//operations/
        +Endpoint: PATCH [DATABRICKS_URL]/api/2.0/postgres/projects/test-pg-proj-[UNIQUE_NAME]?update_mask=spec.default_endpoint_settings%2Cspec.default_endpoint_settings.autoscaling_limit_max_cu
        +HTTP Status: 400 Bad Request
        +API error_code: INVALID_PARAMETER_VALUE
        +API message: Field 'spec.default_endpoint_settings.suspension' is in update_mask but not provided in request
         
        ->>> [CLI] postgres get-project projects/test-pg-proj-[UNIQUE_NAME]
        -{
        -  "autoscaling_limit_max_cu": 8,
        -  "autoscaling_limit_min_cu": 0.5,
        -  "suspend_timeout_duration": "86400s"
        -}
        +Files: 0 uploaded, 0 deleted
         
        -=== Plan again: the change must leave no drift behind
        ->>> [CLI] bundle plan
        -Plan: 0 to add, 0 to change, 0 to delete, 1 unchanged
        -
         >>> [CLI] bundle destroy --auto-approve
         The following resources will be deleted:
           delete resources.postgres_projects.my_project
        @@ -44,3 +33,5 @@
         All files and directories at the following location will be deleted: /Workspace/Users/[USERNAME]/.bundle/test-bundle-[UNIQUE_NAME]/default
         
         Destroy: 1 deleted
        +
        +Exit code: 1
        
    acceptance_test.go:1147: Missing output file: out.requests.direct.json
--- FAIL: TestAccept/bundle/resources/postgres_projects/update_default_endpoint_autoscaling/DATABRICKS_BUNDLE_ENGINE=direct (23.68s)
```

</details>

<details>
<summary>TestAccept/bundle/resources/postgres_projects/update_default_endpoint_suspend/DATABRICKS_BUNDLE_ENGINE=direct ✅ | main (0a8aae165) ❌ | latest ➖</summary>

**main (0a8aae165):**
```
=== RUN   TestAccept/bundle/resources/postgres_projects/update_default_endpoint_suspend/DATABRICKS_BUNDLE_ENGINE=direct
=== PAUSE TestAccept/bundle/resources/postgres_projects/update_default_endpoint_suspend/DATABRICKS_BUNDLE_ENGINE=direct
=== CONT  TestAccept/bundle/resources/postgres_projects/update_default_endpoint_suspend/DATABRICKS_BUNDLE_ENGINE=direct
    acceptance_test.go:1176: 
        	Error Trace:	/Users/denis.bilenko/work/cli-trees/update-mask-leaf-only/libs/testdiff/testdiff.go:22
        	            				/Users/denis.bilenko/work/cli-trees/update-mask-leaf-only/acceptance/acceptance_test.go:1176
        	            				/Users/denis.bilenko/work/cli-trees/update-mask-leaf-only/acceptance/acceptance_test.go:1011
        	            				/Users/denis.bilenko/work/cli-trees/update-mask-leaf-only/acceptance/acceptance_test.go:586
        	Error:      	Not equal: 
        	            	expected: "{\n  \"method\": \"PATCH\",\n  \"path\": \"/api/2.0/postgres/projects/test-pg-proj-[UNIQUE_NAME]\",\n  \"q\": {\n    \"update_mask\": \"spec.default_endpoint_settings.suspend_timeout_duration\"\n  },\n  \"body\": {\n    \"spec\": {\n      \"default_endpoint_settings\": {\n        \"autoscaling_limit_max_cu\": 4,\n        \"autoscaling_limit_min_cu\": 0.5,\n        \"suspend_timeout_duration\": \"600s\"\n      },\n      \"display_name\": \"Test Project for Default Endpoint Suspend Update\",\n      \"pg_version\": 16\n    }\n  }\n}\n"
        	            	actual  : "{\n  \"method\": \"PATCH\",\n  \"path\": \"/api/2.0/postgres/projects/test-pg-proj-[UNIQUE_NAME]\",\n  \"q\": {\n    \"update_mask\": \"spec.default_endpoint_settings,spec.default_endpoint_settings.suspend_timeout_duration\"\n  },\n  \"body\": {\n    \"spec\": {\n      \"default_endpoint_settings\": {\n        \"autoscaling_limit_max_cu\": 4,\n        \"autoscaling_limit_min_cu\": 0.5,\n        \"suspend_timeout_duration\": \"600s\"\n      },\n      \"display_name\": \"Test Project for Default Endpoint Suspend Update\",\n      \"pg_version\": 16\n    }\n  }\n}\n"
        	            	
        	            	Diff:
        	            	--- Expected
        	            	+++ Actual
        	            	@@ -4,3 +4,3 @@
        	            	   "q": {
        	            	-    "update_mask": "spec.default_endpoint_settings.suspend_timeout_duration"
        	            	+    "update_mask": "spec.default_endpoint_settings,spec.default_endpoint_settings.suspend_timeout_duration"
        	            	   },
        	Test:       	TestAccept/bundle/resources/postgres_projects/update_default_endpoint_suspend/DATABRICKS_BUNDLE_ENGINE=direct
        	Messages:   	bundle/resources/postgres_projects/update_default_endpoint_suspend/out.requests.direct.json vs /var/folders/5y/9kkdnjw91p11vsqwk0cvmk200000gp/T/TestAcceptbundleresourcespostgres_projectsupdate_default_end2869508788/001/out.requests.direct.json
    acceptance_test.go:1176: 
        	Error Trace:	/Users/denis.bilenko/work/cli-trees/update-mask-leaf-only/libs/testdiff/testdiff.go:22
        	            				/Users/denis.bilenko/work/cli-trees/update-mask-leaf-only/acceptance/acceptance_test.go:1176
        	            				/Users/denis.bilenko/work/cli-trees/update-mask-leaf-only/acceptance/acceptance_test.go:1011
        	            				/Users/denis.bilenko/work/cli-trees/update-mask-leaf-only/acceptance/acceptance_test.go:586
        	Error:      	Not equal: 
        	            	expected: "Uploading bundle files to /Workspace/Users/[USERNAME]/.bundle/update-postgres-project-default-suspend-[UNIQUE_NAME]/default/files...\nError: cannot update resources.postgres_projects.my_project: updating id=projects/test-pg-proj-[UNIQUE_NAME]: Unknown field path in update_mask: 'spec.default_endpoint_settings.suspend_timeout_duration' (400 INVALID_PARAMETER_VALUE)\n\nEndpoint: PATCH [DATABRICKS_URL]/api/2.0/postgres/projects/test-pg-proj-[UNIQUE_NAME]?update_mask=spec.default_endpoint_settings.suspend_timeout_duration\nHTTP Status: 400 Bad Request\nAPI error_code: INVALID_PARAMETER_VALUE\nAPI message: Unknown field path in update_mask: 'spec.default_endpoint_settings.suspend_timeout_duration'\n\nFiles: 0 uploaded, 0 deleted\n\nExit code: 1\n"
        	            	actual  : "Uploading bundle files to /Workspace/Users/[USERNAME]/.bundle/update-postgres-project-default-suspend-[UNIQUE_NAME]/default/files...\nError: cannot update resources.postgres_projects.my_project: updating id=projects/test-pg-proj-[UNIQUE_NAME]: Unknown field path in update_mask: 'spec.default_endpoint_settings.suspend_timeout_duration' (400 INVALID_PARAMETER_VALUE)\n\nEndpoint: PATCH [DATABRICKS_URL]/api/2.0/postgres/projects/test-pg-proj-[UNIQUE_NAME]?update_mask=spec.default_endpoint_settings%2Cspec.default_endpoint_settings.suspend_timeout_duration\nHTTP Status: 400 Bad Request\nAPI error_code: INVALID_PARAMETER_VALUE\nAPI message: Unknown field path in update_mask: 'spec.default_endpoint_settings.suspend_timeout_duration'\n\nFiles: 0 uploaded, 0 deleted\n\nExit code: 1\n"
        	            	
        	            	Diff:
        	            	--- Expected
        	            	+++ Actual
        	            	@@ -3,3 +3,3 @@
        	            	 
        	            	-Endpoint: PATCH [DATABRICKS_URL]/api/2.0/postgres/projects/test-pg-proj-[UNIQUE_NAME]?update_mask=spec.default_endpoint_settings.suspend_timeout_duration
        	            	+Endpoint: PATCH [DATABRICKS_URL]/api/2.0/postgres/projects/test-pg-proj-[UNIQUE_NAME]?update_mask=spec.default_endpoint_settings%2Cspec.default_endpoint_settings.suspend_timeout_duration
        	            	 HTTP Status: 400 Bad Request
        	Test:       	TestAccept/bundle/resources/postgres_projects/update_default_endpoint_suspend/DATABRICKS_BUNDLE_ENGINE=direct
        	Messages:   	bundle/resources/postgres_projects/update_default_endpoint_suspend/out.deploy.direct.txt vs /var/folders/5y/9kkdnjw91p11vsqwk0cvmk200000gp/T/TestAcceptbundleresourcespostgres_projectsupdate_default_end2869508788/001/out.deploy.direct.txt
--- FAIL: TestAccept/bundle/resources/postgres_projects/update_default_endpoint_suspend/DATABRICKS_BUNDLE_ENGINE=direct (55.77s)
```

</details>

