# __init__.py defines the 'transformations' Python package
import importlib
import pkgutil


# Import all modules in the package except those starting with '_', like '__init__.py'
for _, module_name, _ in pkgutil.iter_modules(__path__):
    if not module_name.startswith("_"):
        importlib.import_module(f"{__name__}.{module_name}")
