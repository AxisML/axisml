import js from "@eslint/js";
import tseslint from "typescript-eslint";

export default tseslint.config(
  { ignores: ["dist", "src/api/generated"] },
  js.configs.recommended,
  ...tseslint.configs.recommended,
);
