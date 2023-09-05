"""
Setup script for default_python.

This script packages and distributes the associated wheel file(s).
Source code is in ./src/. Run 'python setup.py sdist bdist_wheel' to build.
"""
from setuptools import setup, find_packages

import sys
sys.path.append('./src')

import default_python

setup(
    name="default_python",
    version=default_python.__version__,
    url="https://databricks.com",
    author="<no value>",
    description="my test wheel",
    packages=find_packages(where='./src'),
    package_dir={'': 'src'},
    entry_points={"entry_points": "main=default_python.main:main"},
    install_requires=["setuptools"],
)
