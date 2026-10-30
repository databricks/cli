* Fix `-e ./folder` in environment dependencies, which failed before deploy. A local project folder after `-e`, with optional extras such as `-e ./project[test]`, now maps to its deployed folder.
