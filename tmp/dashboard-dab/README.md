# Iterating


```
databricks lakeview get 01ef69c6a1b61c85a97505155d58015e --output json | jq -r .serialized_dashboard | jq -S . > dashboard.lvdash.json
```
