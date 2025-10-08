#!/usr/bin/env node
/**
 * CLI entry point for the Databricks bundles TypeScript build system
 *
 * This executable is invoked by the Databricks CLI during bundle processing.
 */

import { main } from "./build/index.js";

main(process.argv).then(
  (exitCode) => {
    process.exit(exitCode);
  },
  (error) => {
    console.error("Fatal error:", error);
    process.exit(1);
  }
);
