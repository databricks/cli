* Fix `databricks apps import` to inline the app's `app.yml`/`app.yaml` into `databricks.yml` as `config`; previously the file was checked before it was downloaded, so it was never inlined.
