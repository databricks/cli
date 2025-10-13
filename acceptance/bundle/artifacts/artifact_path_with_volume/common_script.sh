export SCHEMA_NAME=schema-$UNIQUE_NAME
trace envsubst < $TESTDIR/../databricks.yml.tmpl > databricks.yml
trap "trace '$CLI' schemas delete main.$SCHEMA_NAME" EXIT
trace $CLI schemas create $SCHEMA_NAME main | jq 'del(.effective_predictive_optimization_flag.inherited_from_name)'
trace musterr $CLI bundle deploy
