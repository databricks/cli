from typing import List, Optional
from IPython.core.getipython import get_ipython
from IPython.display import display as ip_display
from dbruntime import UserNamespaceInitializer


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


_user_namespace_initializer = UserNamespaceInitializer.getOrCreate()
_entry_point = _user_namespace_initializer.get_spark_entry_point()
_globals = _user_namespace_initializer.get_namespace_globals()
for name, value in _globals.items():
    print(f"Registering global: {name} = {value}")
    if name not in globals():
        globals()[name] = value


# 'display' from the runtime uses custom widgets that don't work in Jupyter.
# We use the IPython display instead (in combination with the html formatter for DataFrames).
globals()["display"] = ip_display


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
def _register_magics():
    """Register the magic command parser with IPython."""
    from dbruntime.DatasetInfo import UserNamespaceDict
    from dbruntime.PipMagicOverrides import PipMagicOverrides

    user_ns = UserNamespaceDict(
        _user_namespace_initializer.get_namespace_globals(),
        _entry_point.getDriverConf(),
        _entry_point,
    )
    ip = get_ipython()
    ip.input_transformers_cleanup.append(_parse_line_for_databricks_magics)
    ip.register_magics(PipMagicOverrides(_entry_point, _globals["sc"]._conf, user_ns))


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


_register_magics()
_register_formatters()
_register_runtime_hooks()
