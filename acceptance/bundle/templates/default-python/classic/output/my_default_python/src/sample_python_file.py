import argparse
from datetime import datetime
from shared import taxis


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--catalog", default="hive_metastore")
    parser.add_argument("--schema", default="default")
    args = parser.parse_args()

    df = taxis.find_all_taxis()

    table_name = f"{args.catalog}.{args.schema}.taxis_my_default_python"
    df.write.mode("overwrite").saveAsTable(table_name)

    print(f"Wrote {df.count()} taxi records to {table_name}")


if __name__ == "__main__":
    main()
