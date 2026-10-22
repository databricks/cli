pytest_plugins = ["pytester"]


def test_resource_filter_deselects_unaffected_tests(pytester):
    pytester.makepyfile(
        """
        import pytest

        @pytest.mark.bundle_resource("jobs.transform")
        def test_transform():
            pass

        @pytest.mark.bundle_resource("jobs.aggregate")
        def test_aggregate():
            pass

        def test_unmarked():
            pass
        """
    )

    result = pytester.runpytest("--bundletest-resource=jobs.transform", "-q")

    result.assert_outcomes(passed=1, deselected=2)
