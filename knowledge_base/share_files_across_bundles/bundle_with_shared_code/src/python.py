import os
import sys

# Traverse to the sync root path.
# The working directory is equal to the directory containing the notebook.
# Note: this requires DBR >= 14 or serverless.
shared_path = os.getcwd() + "/../../shared"

# Add the shared directory to the Python path.
sys.path.append(shared_path)

# Import a function from the library in the shared directory.
from shared_library import multiply

# Use the function.
result = multiply(2, 3)
print(result)
