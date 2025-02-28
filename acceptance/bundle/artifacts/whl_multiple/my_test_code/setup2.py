from setuptools import setup, find_packages

import src

setup(
    name="my_test_code_2",
    version=src.__version__,
    author=src.__author__,
    url="https://databricks.com",
    author_email="john.doe@databricks.com",
    description="my test wheel",
    packages=find_packages(include=["src"]),
    entry_points={"group_1": "run=src.__main__:main"},
    install_requires=["setuptools"],
)
