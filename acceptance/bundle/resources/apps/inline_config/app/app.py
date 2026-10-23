import os

from flask import Flask

app = Flask(__name__)


@app.route("/")
def home():
    return "Hello from inline config app!"


@app.route("/health")
def health():
    return "OK"
