import os

from flask import Flask

app = Flask(__name__)


@app.route("/")
def home():
    return "Hello from config-no-deployment app!"
