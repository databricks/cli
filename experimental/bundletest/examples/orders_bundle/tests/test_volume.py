"""Volumes — upload a file and read it back.

The local backend copies the uploaded file into a temp volume filesystem and reads it with
DuckDB, so you can assert on the file you're seeding.

CAN test locally:
- an upload lands and the file exists in the volume
- the uploaded file's row count and columns (CSV / JSON / Parquet)

CANNOT test locally — needs the cloud backend:
- that a deployed job reading the volume produced the right table
- listing many files, overwrite semantics, permissions
- formats DuckDB can't read locally -> skips
"""


def test_uploaded_csv_is_readable(env, tmp_path):
    csv = tmp_path / "orders.csv"
    csv.write_text("order_id,total_price\n1,10.0\n2,5.0\n")

    env.volume("raw_data").upload(str(csv))

    orders = env.volume("raw_data").file("orders.csv")
    assert orders.exists()
    assert orders.row_count() == 2
    assert "order_id" in orders.columns
