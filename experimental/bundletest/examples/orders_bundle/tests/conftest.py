"""Shared fixture for the example gallery: deploy the bundle once per test."""

from pathlib import Path

import pytest
from bundletest import bundle_env

BUNDLE = str(Path(__file__).resolve().parent.parent)


@pytest.fixture
def env():
    with bundle_env(BUNDLE) as e:
        yield e
