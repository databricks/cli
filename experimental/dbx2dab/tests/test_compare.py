from dbx2dab.compare import recursive_subtract


def test_recursive_subtract_equal():
    a = {
        "a": 1,
        "b": 2,
        "c": {
            "d": 3,
            "e": 4,
            "f": {
                "g": 5,
            },
        },
    }
    b = {
        "a": 1,
        "b": 2,
        "c": {
            "d": 3,
            "e": 4,
            "f": {
                "g": 5,
            },
        },
    }
    assert recursive_subtract(a, b) == {}


def test_recursive_subtract_nested_element():
    a = {
        "a": 1,
        "b": 2,
        "c": {
            "d": 3,
            "e": 4,
            "f": {
                "g": 5,
            },
        },
    }
    b = {
        "a": 1,
        "b": 2,
        "c": {
            "d": 3,
            "e": 4,
            "f": {
                "g": 6,
            },
        },
    }
    assert recursive_subtract(a, b) == {"c": {"f": {"g": 5}}}


def test_recursive_subtract_superset():
    a = {
        "a": 1,
        "b": 2,
        "c": {
            "d": 3,
            "e": 4,
            "f": {
                "g": 5,
            },
        },
    }
    b = {
        "a": 1,
        "b": 2,
        "c": {
            "d": 3,
            "e": 4,
            "f": {
                "g": 5,
                "h": 6,
            },
        },
    }
    assert recursive_subtract(a, b) == {}


def test_recursive_subtract_list_job_clusters():
    a = [
            {
                "job_cluster_key": "cluster1",
                "node_type_id": "Standard_DS3_v2",
                "spark_version": "14.3.x-scala2.12",
                "num_workers": 1,
            },
            {
                "job_cluster_key": "cluster2",
                "node_type_id": "Standard_DS3_v2",
                "spark_version": "14.3.x-scala2.12",
                "num_workers": 1,
            },
    ]

    b = [
            {
                "job_cluster_key": "cluster1",
                "node_type_id": "Standard_DS3_v2",
                "spark_version": "14.3.x-scala2.12",
                "num_workers": 1,
            },
            {
                "job_cluster_key": "cluster2",
                "node_type_id": "Standard_DS3_v2",
                "spark_version": "14.3.x-scala2.12",
                "num_workers": 1,
            },
    ]

    assert recursive_subtract(a, b) == []

def test_recursive_subtract_list_job_clusters_reverse_order():
    a = [
            {
                "job_cluster_key": "cluster1",
                "node_type_id": "Standard_DS3_v2",
                "spark_version": "14.3.x-scala2.12",
                "num_workers": 1,
            },
            {
                "job_cluster_key": "cluster2",
                "node_type_id": "Standard_DS3_v2",
                "spark_version": "14.3.x-scala2.12",
                "num_workers": 1,
            },
    ]

    b = [
            {
                "job_cluster_key": "cluster2",
                "node_type_id": "Standard_DS3_v2",
                "spark_version": "14.3.x-scala2.12",
                "num_workers": 1,
            },
            {
                "job_cluster_key": "cluster1",
                "node_type_id": "Standard_DS3_v2",
                "spark_version": "14.3.x-scala2.12",
                "num_workers": 1,
            },
    ]

    assert recursive_subtract(a, b) == []


def test_recursive_subtract_list_job_clusters_nominal():
    a = [
            {
                "job_cluster_key": "cluster1",
                "node_type_id": "Standard_DS3_v2",
                "spark_version": "14.3.x-scala2.12",
                "num_workers": 2,
            },
            {
                "job_cluster_key": "cluster2",
                "node_type_id": "Standard_DS3_v2",
                "spark_version": "14.3.x-scala2.12",
                "num_workers": 2,
            }
    ]

    b = [
            {
                "job_cluster_key": "cluster1",
                "node_type_id": "Standard_DS3_v2",
                "spark_version": "14.3.x-scala2.12",
                "num_workers": 1,
            },
            {
                "job_cluster_key": "cluster2",
                "node_type_id": "Standard_DS3_v2",
                "spark_version": "14.3.x-scala2.12",
                "num_workers": 1,
            },
    ]

    assert recursive_subtract(a, b) == [
            {
                "job_cluster_key": "cluster1",
                "num_workers": 2,
            },
            {
                "job_cluster_key": "cluster2",
                "num_workers": 2,
            }
    ]
