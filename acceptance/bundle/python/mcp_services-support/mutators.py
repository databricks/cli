from dataclasses import replace

from databricks.bundles.mcp_services import McpService
from databricks.bundles.core import mcp_service_mutator


@mcp_service_mutator
def update_mcp_service(mcp_service: McpService) -> McpService:
    assert isinstance(mcp_service.comment, str)

    return replace(mcp_service, comment=f"{mcp_service.comment} (updated)")
