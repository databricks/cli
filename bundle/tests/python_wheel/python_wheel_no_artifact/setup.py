from setuptools import setup, find_packages

import my_test_code

setup(
    name="my_test_code",
    version=my_test_code.__version__,
    author=my_test_code.__author__,
    url="https://databricks.com",
    author_email="john.doe@databricks.com",
    description="my test wheel",
    packages=find_packages(include=["my_test_code"]),
    entry_points={"group_1": "run=my_test_code.__main__:main"},
    install_requires=["setuptools"],
)
