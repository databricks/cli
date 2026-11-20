-- Recipe setup: materialise a sample table.
CREATE OR REPLACE TABLE main.default.trips AS SELECT * FROM samples.nyctaxi.trips LIMIT 10
