import js from "@eslint/js";
import tseslint from "typescript-eslint";

export default tseslint.config(
  // Generated outputs: the typed client and the mock fixtures lifted from the spec.
  { ignores: ["dist", "src/api/generated", "src/api/mock/examples.gen.ts"] },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  // Build tooling scripts run under Node.
  {
    files: ["scripts/**/*.mjs"],
    languageOptions: { globals: { console: "readonly", process: "readonly" } },
  },
);
