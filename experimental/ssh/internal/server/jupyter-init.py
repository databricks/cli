from typing import List, Optional
from IPython.core.getipython import get_ipython
from IPython.display import display as ip_display
import os


def _log_exceptions(func):
    from functools import wraps

    @wraps(func)
    def wrapper(*args, **kwargs):
        try:
            print(f"Executing {func.__name__}")
            return func(*args, **kwargs)
        except Exception as e:
            print(f"Error in {func.__name__}: {e}")

    return wrapper


@_log_exceptions
def _register_runtime_hooks():
    from dbruntime.monkey_patches import apply_dataframe_display_patch
    from dbruntime.IPythonShellHooks import load_ipython_hooks, IPythonShellHook
    from IPython.core.interactiveshell import ExecutionInfo

    # Setting executing_raw_cell before cell execution is required to make dbutils.library.restartPython() work
    class PreRunHook(IPythonShellHook):
        def pre_run_cell(self, info: ExecutionInfo) -> None:
            get_ipython().executing_raw_cell = info.raw_cell

    load_ipython_hooks(get_ipython(), PreRunHook())
    apply_dataframe_display_patch(ip_display)


def _warn_for_dbr_alternative(magic: str):
    import warnings

    """Warn users about magics that have Databricks alternatives."""
    local_magic_dbr_alternative = {"%%sh": "%sh"}
    if magic in local_magic_dbr_alternative:
        warnings.warn(
            f"\\n{magic} is not supported on Databricks. This notebook might fail when running on a Databricks cluster.\\n"
            f"Consider using %{local_magic_dbr_alternative[magic]} instead."
        )


def _throw_if_not_supported(magic: str):
    """Throw an error for magics that are not supported locally."""
    unsupported_dbr_magics = ["%r", "%scala"]
    if magic in unsupported_dbr_magics:
        raise NotImplementedError(f"{magic} is not supported for local Databricks Notebooks.")


def _get_cell_magic(lines: List[str]) -> Optional[str]:
    """Extract cell magic from the first line if it exists."""
    if len(lines) == 0:
        return None
    if lines[0].strip().startswith("%%"):
        return lines[0].split(" ")[0].strip()
    return None


def _get_line_magic(lines: List[str]) -> Optional[str]:
    """Extract line magic from the first line if it exists."""
    if len(lines) == 0:
        return None
    if lines[0].strip().startswith("%"):
        return lines[0].split(" ")[0].strip().strip("%")
    return None


def _handle_cell_magic(lines: List[str]) -> List[str]:
    """Process cell magic commands."""
    cell_magic = _get_cell_magic(lines)
    if cell_magic is None:
        return lines

    _warn_for_dbr_alternative(cell_magic)
    _throw_if_not_supported(cell_magic)
    return lines


def _handle_line_magic(lines: List[str]) -> List[str]:
    """Process line magic commands and transform them appropriately."""
    lmagic = _get_line_magic(lines)
    if lmagic is None:
        return lines

    _warn_for_dbr_alternative(lmagic)
    _throw_if_not_supported(lmagic)

    if lmagic in ["md", "md-sandbox"]:
        lines[0] = "%%markdown" + lines[0].partition("%" + lmagic)[2]
        return lines

    if lmagic == "sh":
        lines[0] = "%%sh" + lines[0].partition("%" + lmagic)[2]
        return lines

    if lmagic == "sql":
        lines = lines[1:]
        spark_string = (
            "global _sqldf\n"
            + "_sqldf = spark.sql('''"
            + "".join(lines).replace("'", "\\'")
            + "''')\n"
            + "display(_sqldf)\n"
        )
        return spark_string.splitlines(keepends=True)

    if lmagic == "python":
        return lines[1:]

    return lines


def _strip_hash_magic(lines: List[str]) -> List[str]:
    if len(lines) == 0:
        return lines
    if lines[0].startswith("# MAGIC"):
        return [line.partition("# MAGIC ")[2] for line in lines]
    return lines


def _parse_line_for_databricks_magics(lines: List[str]) -> List[str]:
    """Main parser function for Databricks magic commands."""
    if len(lines) == 0:
        return lines

    lines_to_ignore = ("# Databricks notebook source", "# COMMAND ----------", "# DBTITLE")
    lines = [line for line in lines if not line.strip().startswith(lines_to_ignore)]
    lines = "".join(lines).strip().splitlines(keepends=True)
    lines = _strip_hash_magic(lines)

    if _get_cell_magic(lines):
        return _handle_cell_magic(lines)

    if _get_line_magic(lines):
        return _handle_line_magic(lines)

    return lines


@_log_exceptions
def _register_common_magics():
    """Register the common magic command parser with IPython."""
    ip = get_ipython()
    ip.input_transformers_cleanup.append(_parse_line_for_databricks_magics)


@_log_exceptions
def _register_pip_magics():
    """Register the pip magic command parser with IPython."""
    from dbruntime.DatasetInfo import UserNamespaceDict
    from dbruntime.PipMagicOverrides import PipMagicOverrides
    from dbruntime import UserNamespaceInitializer

    user_namespace_initializer = UserNamespaceInitializer.getOrCreate()
    entry_point = user_namespace_initializer.get_spark_entry_point()
    user_ns = UserNamespaceDict(
        user_namespace_initializer.get_namespace_globals(),
        entry_point.getDriverConf(),
        entry_point,
    )
    ip = get_ipython()

    try:
        # Older DBRs
        pip_magic = PipMagicOverrides(entry_point, ip.user_ns["sc"]._conf, user_ns)
    except Exception:
        # Newer DBRs
        pip_magic = PipMagicOverrides(entry_point, user_ns, ip)

    ip.register_magics(pip_magic)


@_log_exceptions
def _register_formatters():
    from pyspark.sql import DataFrame
    from pyspark.sql.connect.dataframe import DataFrame as SparkConnectDataframe

    def df_html(df: DataFrame) -> str:
        return df.toPandas().to_html()

    ip = get_ipython()
    html_formatter = ip.display_formatter.formatters["text/html"]
    html_formatter.for_type(SparkConnectDataframe, df_html)
    html_formatter.for_type(DataFrame, df_html)


def _create_spark_session(builder_fn):
    from databricks.connect import DatabricksSession
    user_ns = get_ipython().user_ns
    existing_session = user_ns.get("spark")
    # Clear the existing local spark session, otherwise DatabricksSession will re-use it.
    user_ns["spark"] = None
    try:
        return builder_fn(DatabricksSession.builder).getOrCreate()
    except Exception:
        user_ns["spark"] = existing_session
        raise


def _initialize_spark(is_serverless: bool, existing_spark: any):
    from pyspark.sql.session import SparkSession

    # On serverless always initialize a new remote Databricks Connect session.
    if is_serverless:
        return _create_spark_session(lambda b: b.serverless(True))
    # On dedicated or standard initialize a new remote session if the existing spark session is local.
    if existing_spark is None or isinstance(existing_spark, SparkSession):
        return _create_spark_session(lambda b: b.remote(
            host=os.environ["DATABRICKS_HOST"],
            token=os.environ["DATABRICKS_TOKEN"],
            cluster_id=os.environ["DATABRICKS_CLUSTER_ID"],
        ))
    # Otherwise re-use the existing remote session.
    return existing_spark


@_log_exceptions
def _setup_globals(is_serverless: bool):
    from dbruntime import UserNamespaceInitializer

    ns = UserNamespaceInitializer.getOrCreate()
    ns_globals = ns.get_namespace_globals()
    existing_spark = ns_globals.get("spark")
    spark = _initialize_spark(is_serverless, existing_spark)
    try:
        ns.db_connection.spark_provider.set_spark(spark)
    except Exception as e:
        print(f"Error updating spark provider: {e}")
    ns_globals["spark"] = spark
    if spark is not None:
        ns_globals["table"] = spark.table
        ns_globals["sql"] = spark.sql
    user_ns = get_ipython().user_ns
    for name, value in ns_globals.items():
        print(f"Registering global: {name} = {value}")
        user_ns[name] = value
    # 'display' from the runtime uses custom widgets that don't work in Jupyter.
    # We use the IPython display instead (in combination with the html formatter for DataFrames).
    user_ns["display"] = ip_display


_setup_globals(os.environ.get("DATABRICKS_JUPYTER_SERVERLESS") == "true")
_register_pip_magics()
_register_common_magics()
_register_formatters()
_register_runtime_hooks()
