# Databricks notebook source
import os
import sys
import json

out = {"PYTHONPATH": sys.path, "CWD": os.getcwd()}
json_object = json.dumps(out, indent=4)
dbutils.notebook.exit(json_object)
