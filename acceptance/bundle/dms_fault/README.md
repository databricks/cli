Each API case runs the same linear workflow with a job and a pipeline. DMS records
deployment history while files remain the source of truth. All bundle commands
must succeed with the selected API configured to return HTTP 500.

| Step | Command or action | Assertion |
| --- | --- | --- |
| 1 | Deploy with DMS healthy. | Establish an existing deployment and its DMS deployment ID. |
| 2 | Enable the API fault, rename the bundle and resources with the workspace root pinned, then deploy. | Both resources exist with their updated names. State retains file-backed history recording. The bundle rename exercises `UpdateDeployment`. |
| 3 | Run `bundle summary`. | Output contains both resource IDs. |
| 4 | Run `bundle generate` into the export directory. | The resource YAML exists; pipeline SQL is downloaded with its original content. |
| 5 | Run `bundle deployment unbind`. | The resource ID is removed from bundle state. |
| 6 | Run `bundle generate --bind` into the bundle's resource directory. | Generation and binding succeed for the original resource ID, proving unbind preserved the remote resource. Pipeline SQL is downloaded with its original content. |
| 7 | Run `bundle validate`, then deploy again. | Generated configuration is valid, both resource IDs stay unchanged, and the pipeline references the newly downloaded SQL path. |
| 8 | Run `bundle destroy`. | Both remote resources and remote state are gone. |

Steps 4–6 run first for the job, then for the pipeline. The fault stays enabled
through step 8 and is cleared in `finally`. For `CreateDeployment`, skip the healthy
setup deployment so the fault applies to the first deployment.

Each snapshot records the command invocations and output, including warnings and
deployment-version links, then reports whether the injected endpoint was called.
An unused endpoint records `DMS API called: no`.

Fault cases cover all DMS APIs:

- `CompleteVersion`
- `CreateDeployment`
- `CreateVersion`
- `DeleteDeployment`
- `GetDeployment`
- `GetOperation`
- `GetResource`
- `GetVersion`
- `Heartbeat`
- `ListDeployments`
- `ListOperations`
- `ListResources`
- `ListVersions`
- `UpdateDeployment`
- `UpdateOperation`

Missing APIs: none.
