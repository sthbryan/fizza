import path from "node:path";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [svelte(), tailwindcss()],
  base: "/",
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "src"),
    },
  },
  server: {
    port: 5173,
    proxy: {
      "/v1": {
        target: "http://127.0.0.1:6500",
        changeOrigin: true,
      },
      "/healthz": "http://127.0.0.1:6500",
    },
  },
  build: {
    outDir: path.resolve(__dirname, "../internal/httpapi/static/app"),
    emptyOutDir: true,
    assetsDir: "assets",
  },
});
