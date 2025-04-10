Core
===============================

.. currentmodule:: databricks.bundles.core

**Package:** ``databricks.bundles.core``

.. comment:
    We don't use automodule to manually order the classes,
    and ensure that every class is documented well before
    adding it to the documentation.

Classes
-----------

.. autoclass:: databricks.bundles.core.Resources
.. autoclass:: databricks.bundles.core.Resource
.. autoclass:: databricks.bundles.core.ResourceMutator
.. autoclass:: databricks.bundles.core.Bundle
.. autoclass:: databricks.bundles.core.Variable
.. autoclass:: databricks.bundles.core.Diagnostics
.. autoclass:: databricks.bundles.core.Diagnostic
.. autoclass:: databricks.bundles.core.Location
.. autoclass:: databricks.bundles.core.Severity

.. class:: T

    :class:`~typing.TypeVar` for variable value

Methods
-----------

.. automethod:: databricks.bundles.core.load_resources_from_current_package_module
.. automethod:: databricks.bundles.core.load_resources_from_module
.. automethod:: databricks.bundles.core.load_resources_from_modules
.. automethod:: databricks.bundles.core.load_resources_from_package_module

Decorators
-----------
.. autodecorator:: databricks.bundles.core.job_mutator
.. autodecorator:: databricks.bundles.core.variables
