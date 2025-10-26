import js from "@eslint/js";
import globals from "globals";
import { defineConfig, globalIgnores } from "eslint/config";

export default defineConfig([
  {
    files: ["**/*.{js,mjs,cjs}"],
    plugins: { js },
    extends: ["js/recommended"],
    languageOptions: { globals: globals.browser },
    rules: {
      "no-unused-vars": "error",
      "semi": ["error", "always" ],
      "quotes": ["error", "double"],
      "eqeqeq": ["error", "always"],
    },
  },
  globalIgnores(["public/*.min.js"])
]);
