import path from "node:path";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

// Solid Glass 的 `@custom-variant dark` 与 `@theme inline` 需要 Tailwind 4 以
// Vite 插件方式接入；React 通过 @vitejs/plugin-react 5（兼容 Vite 7）接入。
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(import.meta.dirname, "./src"),
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
    // Wails 把 frontend/dist 嵌入二进制；保持相对路径资源引用。
    assetsDir: "assets",
  },
});
