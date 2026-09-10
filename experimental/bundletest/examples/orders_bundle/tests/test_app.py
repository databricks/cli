"""App wiring — its launch command and source path.

CAN test locally:
- the app exists, its launch command, and source_code_path

CANNOT test locally — needs the cloud backend:
- deploying / starting the app
"""


def test_app_command(env):
    app = env.app("orders_app")
    assert app.exists()
    assert app.command() == ["python", "app.py"]
    assert app.source_code_path == "./app"
