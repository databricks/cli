"""Volumes — file upload.

The local backend records the upload but has no file store to read back yet, so this is a
stub-level capability.

CAN test locally:
- an upload call is wired and does not error

CANNOT test locally — needs the cloud backend:
- that the file actually landed in the volume
- reading a table loaded FROM the uploaded file (row count, schema)
- file listing / overwrite / permissions
"""


def test_upload_is_wired(env):
    # No public read-back locally; this only exercises the call path.
    env.volume("raw_data").upload("fixtures/orders.csv")
