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
        confirm: "readonly",
        setTimeout: "readonly",
        Error: "readonly",
      },
    },
  },
];
