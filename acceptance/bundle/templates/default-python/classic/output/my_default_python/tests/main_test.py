from my_default_python import main


def test_find_all_taxis():
    taxis = main.find_all_taxis()
    assert taxis.count() > 5
