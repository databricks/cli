#!/bin/bash

# Creating a feature spec resolves its lookup table in Unity Catalog, and the invariant harness
# provisions no tables of its own.
feature_spec_fixtures "fspec_inv_$UNIQUE_NAME"
