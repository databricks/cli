import os
import sys
import json

out = {"PYTHONPATH": sys.path, "CWD": os.getcwd()}
json_object = json.dumps(out, indent=4)
print(json_object)
