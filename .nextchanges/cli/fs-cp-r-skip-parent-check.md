* `databricks fs cp -r` no longer checks that the parent directory exists before each file it copies, cutting the number of requests for a copy of many small files by about half.
