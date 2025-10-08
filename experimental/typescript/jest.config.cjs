/** @type {import('jest').Config} */
const config = {
  preset: "ts-jest",
  testEnvironment: "node",
  roots: ["<rootDir>/src"],
  testMatch: ["**/__tests__/**/*.ts", "**/?(*.)+(spec|test).ts"],
  moduleFileExtensions: ["ts", "tsx", "js", "jsx", "json", "node"],
  collectCoverageFrom: [
    "src/**/*.ts",
    "!src/**/*.d.ts",
    "!src/**/*.test.ts",
    "!src/**/*.spec.ts",
    "!src/**/index.ts",
  ],
  coverageDirectory: "coverage",
  coverageReporters: ["text", "lcov", "html"],
  coveragePathIgnorePatterns: ["/node_modules/", "/dist/", "/generated/"],
  moduleNameMapper: {
    "^@databricks/bundles/core$": "<rootDir>/src/core",
    "^@databricks/bundles/jobs$": "<rootDir>/generated/jobs",
    "^@databricks/bundles/pipelines$": "<rootDir>/generated/pipelines",
    "^@databricks/bundles/schemas$": "<rootDir>/generated/schemas",
    "^@databricks/bundles/volumes$": "<rootDir>/generated/volumes",
    "^(\\.{1,2}/.*)\\.js$": "$1",
  },
  transform: {
    "^.+\\.ts$": [
      "ts-jest",
      {
        tsconfig: {
          module: "commonjs",
          esModuleInterop: true,
        },
      },
    ],
  },
  clearMocks: true,
  resetMocks: true,
  restoreMocks: true,
};

module.exports = config;
