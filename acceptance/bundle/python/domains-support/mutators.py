from dataclasses import replace

from databricks.bundles.core import domain_mutator
from databricks.bundles.domains import Domain


@domain_mutator
def update_domain(domain: Domain) -> Domain:
    assert isinstance(domain.subtitle, str)

    return replace(domain, subtitle=f"{domain.subtitle} (updated)")
