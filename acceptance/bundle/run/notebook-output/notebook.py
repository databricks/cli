# Databricks notebook source
import json

# A notebook task's output is its exit value only: this print must not show up.
print("printed before exit")

dbutils.notebook.exit(json.dumps({"message": "hello from notebook"}))

raise AssertionError("dbutils.notebook.exit did not stop the notebook")
