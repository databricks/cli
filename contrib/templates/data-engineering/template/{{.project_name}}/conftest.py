# conftest.py is used to configure pytest.
# This file is in the root since it affects all tests through this bundle.
# It makes sure all 'assets/*' directories are added to `sys.path` so that
# tests can import them.
import os
import sys
import dlt
import pathlib
import pytest
import warnings
from pyspark.sql import SparkSession
from databricks.connect import DatabricksSession

# Dynamically find and add all `assets/*` directories to `sys.path`
for path in pathlib.Path(pathlib.Path(__file__).parent / "assets").glob("*"):
    resolved_path = str(path.resolve())
    if resolved_path not in sys.path:
        sys.path.append(resolved_path)

# For older databricks-connect, work around issues importing SparkSession
# and errors when SPARK_REMOTE is set.
SparkSession.builder = DatabricksSession.builder
os.environ.pop("SPARK_REMOTE", None)

# Make dlt.views in 'sources/dev' available for tests
warnings.filterwarnings(
    "ignore",
    message="This is a stub that only contains the interfaces to Delta Live Tables.*",
    category=UserWarning,
)
dlt.enable_local_execution()
dlt.view = lambda func=None, *args, **kwargs: func or (lambda f: f)


# Provide a 'spark' fixture for tests and make sure the session is eagerly initialized
@pytest.fixture(scope="session", autouse=True)
def spark() -> SparkSession:
    if hasattr(DatabricksSession.builder, "validateSession"):
        return DatabricksSession.builder.validateSession().getOrCreate()
    return DatabricksSession.builder.getOrCreate()
