from setuptools import setup, find_packages

import sys

sys.path.append("./src")

import my_package

setup(
    name="my_package",
    version=my_package.__version__,
    url="https://databricks.com",
    author="[USERNAME]",
    description="wheel file based on my_package/src",
    packages=find_packages(where="./src"),
    package_dir={"": "src"},
    entry_points={
        "packages": [
            "main=my_package.main:main",
        ],
    },
    install_requires=[
        # Dependencies in case the output wheel file is used as a library dependency.
        # For defining dependencies, when this package is used in Databricks, see:
        # https://docs.databricks.com/dev-tools/bundles/library-dependencies.html
        "setuptools"
    ],
)
