# Contrib Directory

The `contrib` directory contains additional community-contributed examples and resources for Databricks Asset Bundles. These examples may include:

- Custom configurations and extensions
- Advanced usage patterns
- Tools or utilities for enhancing Databricks Asset Bundles workflows

## Structure

Each contribution should be organized into its own subdirectory within `contrib/`.
Templates should go under `contrib/templates/`. For example:

```
contrib/
├── awesome-bundle/
│   ├── README.md
│   ├── databricks.yml
│   └── ...
└── templates/
    └── awesome-template/
        ├── README.md
        ├── databricks_template_schema.json
        ├── library/
        │   └── ...
        └── template/
            └── ...
```

## How to Use Contributions

To use or explore a contributed example, navigate to its subdirectory and follow the instructions in its `README.md` file. Each example should provide details on setup, configuration, and usage.

## Contributing

If you would like to add your own examples or resources, please:
1. Create a new directory under `contrib/` with a descriptive name.
2. Include a `README.md` file explaining the contribution.
3. Ensure that any necessary configuration files, scripts, or dependencies are included.

For more information on Databricks Asset Bundles, see:
- The launch blog post at https://www.databricks.com/blog/announcing-general-availability-databricks-asset-bundles
- The docs at https://docs.databricks.com/dev-tools/bundles/index.html
