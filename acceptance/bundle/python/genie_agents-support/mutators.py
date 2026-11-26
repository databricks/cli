from dataclasses import replace

from databricks.bundles.core import genie_agent_mutator
from databricks.bundles.genie_agents import GenieAgent


@genie_agent_mutator
def update_genie_agent(genie_agent: GenieAgent) -> GenieAgent:
    assert isinstance(genie_agent.display_name, str)

    return replace(genie_agent, display_name=f"{genie_agent.display_name} (updated)")
