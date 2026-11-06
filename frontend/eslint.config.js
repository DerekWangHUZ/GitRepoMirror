import js from "@eslint/js";

export default [
  { ignores: ["dist/**", "node_modules/**", "wailsjs/**", "test-results/**"] },
  js.configs.recommended,
  {
    files: ["src/**/*.js", "tests/**/*.js"],
    languageOptions: {
      ecmaVersion: "latest",
      sourceType: "module",
      globals: {
        document: "readonly",
        window: "readonly",
        URLSearchParams: "readonly",
        FormData: "readonly",
        MouseEvent: "readonly",
        Promise: "readonly",
        setTimeout: "readonly",
        Error: "readonly",
      },
    },
  },
];
