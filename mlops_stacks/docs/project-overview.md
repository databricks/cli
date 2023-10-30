# Project Overview

[(back to main README)](../README.md)

## ML pipeline structure
This project defines an ML pipeline for automated retraining and batch inference of an ML model
on tabular data.

See the full pipeline structure below. The [MLOps Stacks README](https://github.com/databricks/mlops-stacks/blob/main/Pipeline.md)
contains additional details on how ML pipelines are tested and deployed across each of the dev, staging, prod environments below.

![MLOps Stacks diagram](images/mlops-stack-summary.png)


## Code structure
This project contains the following components:

| Component                  | Description                                                                                                                                                                                                                                                                                                                                             |
|----------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| ML Code                    | Example ML project code, with unit tested Python modules and notebooks                                                                                                                                                                                                                                                                                  |
| ML Resource Config as Code | ML pipeline resource config (training and batch inference job schedules, etc) configured and deployed through [databricks CLI bundles](https://learn.microsoft.com/azure/databricks/dev-tools/cli/bundle-cli)                                                                                              |
| CI/CD                      | [GitHub Actions](https://github.com/actions) workflows to test and deploy ML code and resources       |

contained in the following files:

```
mlops_stacks        <- Root directory. Both monorepo and polyrepo are supported.
│
├── mlops_stacks       <- Contains python code, notebooks and ML resources related to one ML project. 
│   │
│   ├── requirements.txt        <- Specifies Python dependencies for ML code (for example: model training, batch inference).
│   │
│   ├── databricks.yml          <- databricks.yml is the root ML resource config file for the ML project that can be loaded by databricks CLI bundles. It defines the bundle name, workspace URL and resource config component to be included.
│   │
│   ├── training                <- Training folder contains Notebook that trains and registers the model.
│   │
│   ├── validation              <- Optional model validation step before deploying a model.
│   │
│   ├── monitoring              <- Model monitoring, feature monitoring, etc.
│   │
│   ├── deployment              <- Deployment and Batch inference workflows
│   │   │
│   │   ├── batch_inference     <- Batch inference code that will run as part of scheduled workflow.
│   │   │
│   │   ├── model_deployment    <- As part of CD workflow, promote model to Production stage in model registry.
│   │
│   │
│   ├── tests                   <- Unit tests for the ML project, including the modules under `features`.
│   │
│   ├── resources               <- ML resource (ML jobs, MLflow models) config definitions expressed as code, across dev/staging/prod/test.
│       │
│       ├── model-workflow-resource.yml                <- ML resource config definition for model training, validation, deployment workflow
│       │
│       ├── batch-inference-workflow-resource.yml      <- ML resource config definition for batch inference workflow
│       │
│       ├── ml-artifacts-resource.yml                  <- ML resource config definition for model and experiment
│       │
│       ├── monitoring-workflow-resource.yml           <- ML resource config definition for data monitoring workflow
│
├── .github                     <- Configuration folder for CI/CD using GitHub Actions. The CI/CD workflows run the notebooks
                                   under `notebooks` to test and deploy model training code
```

## Next Steps
See the [main README](../README.md#using-this-repo) for additional links on how to work with this repo.
