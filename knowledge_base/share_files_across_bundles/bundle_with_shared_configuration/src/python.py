import json
import os

# Traverse to the sync root path.
# The working directory is equal to the directory containing the notebook.
# Note: this requires DBR >= 14 or serverless.
shared_path = os.getcwd() + "/../../shared"

# Load the configuration stored in the shared directory.
with open(shared_path + "/config/data.json") as file:
    config = json.load(file)

# Print the configuration
print(config)
