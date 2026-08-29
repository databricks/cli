export SCHEMA_NAME=schema-$UNIQUE_NAME
trace envsubst < $TESTDIR/../databricks.yml.tmpl > databricks.yml
trap "trace '$CLI' schemas delete main.$SCHEMA_NAME" EXIT
# effective_predictive_optimization_flag is inherited from the metastore and backend-controlled; drop it so the test does not depend on metastore settings.
trace $CLI schemas create $SCHEMA_NAME main | jq 'del(.effective_predictive_optimization_flag)'
trace musterr $CLI bundle deploy
