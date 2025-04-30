import os
import signal
from flask import Flask, request

app = Flask(__name__)

app.logger.warning("Python Flask app has started with: " + os.environ.get("TEST"))


@app.route("/")
def index():
    return dict(request.headers)


@app.route("/shutdown")
def shutdown():
    os._exit(signal.SIGTERM)
    return "Shutting down..."
