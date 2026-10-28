These tests exercise file-backed deployment history while one DMS API persistently returns HTTP 500. Each directory selects an endpoint; the shared runner installs an endpoint-specific `fault.py` rule and checks deploy, validate, summary, generation, unbind, generation with binding, redeploy, and destroy for a job and a pipeline.

The setup deliberately reaches the selected API: metadata updates rename the bundle with its workspace root pinned, and resource updates force version creation and operation uploads. Destroy uses a separate healthy deployment so a failed deploy-history call cannot prevent the destroy call from being exercised. `UpdateDeployment` is not used by destroy. Deployment creation is also tested when earlier history registration failed.

Every used fault must produce the injected error and a recorded request. Assertions check live resource names, authoritative file-state IDs, generated pipeline source, successful rebinding, and deletion of the resources and workspace state.

The file-backed path uses `GetDeployment` for version metadata; it does not call `GetVersion` or read resources from DMS. Fault cases are local only because `fault.py` is provided by the fake server, not the cloud recording proxy.
