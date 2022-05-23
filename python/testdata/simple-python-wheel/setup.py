from setuptools import setup, find_packages

setup(
    name='dummy',
    version='0.0.1',
    packages=find_packages(exclude=['tests', 'tests.*']),
    install_requires=['requests==2.27.1']
)
