"""Pipeline (Lakeflow) wiring — read a pipeline's declared config without running it.

Running a pipeline and asserting on the tables it materializes needs a real workspace, so
it's cloud-only. Locally we read the wiring straight from databricks.yml.

CAN test locally:
- a pipeline exists and its target catalog / schema
- which notebook / file libraries it runs, in declaration order

CANNOT test locally — needs the cloud backend:
- running the pipeline and asserting on its output tables (@pytest.mark.cloud_only)
- server-normalized / defaulted config
"""


def test_pipeline_wiring(env):
    pipeline = env.pipeline("enrich_orders")
    assert pipeline.exists()
    assert pipeline.catalog == "shop"
    assert pipeline.schema == "enriched"
    assert pipeline.libraries() == ["src/enrich_orders"]
