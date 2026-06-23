import { defineConfig } from "@hey-api/openapi-ts";

// Generates a typed client + SDK from the platform-backend OpenAPI contract.
// Output lands in src/api/generated/ (the single source of truth for HTTP types).
// Regenerate with `npm run gen:api` whenever axisml-platform/docs/apis/platform.yaml changes.
export default defineConfig({
  input: "../docs/apis/platform.yaml",
  output: {
    path: "src/api/generated",
    format: "prettier",
  },
  plugins: [
    {
      name: "@hey-api/client-fetch",
      runtimeConfigPath: "./src/api/client.ts",
    },
    "@hey-api/typescript",
    "@hey-api/sdk",
  ],
});
