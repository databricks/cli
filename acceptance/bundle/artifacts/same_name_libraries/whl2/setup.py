from setuptools import setup, find_packages

import sys

sys.path.append("./src")

import my_default_python

setup(
    name="my_default_python",
    version=my_default_python.__version__,
    url="https://databricks.com",
    author="[USERNAME]",
    description="wheel file based on my_default_python/src",
    packages=find_packages(where="./src"),
    package_dir={"": "src"},
    entry_points={
        "packages": [
            "main=my_default_python.main:main",
        ],
    },
    install_requires=[
        # Dependencies in case the output wheel file is used as a library dependency.
        # For defining dependencies, when this package is used in Databricks, see:
        # https://docs.databricks.com/dev-tools/bundles/library-dependencies.html
        "setuptools"
    ],
)
