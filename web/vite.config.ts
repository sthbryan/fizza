import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import tailwindcss from "@tailwindcss/vite";
import path from "node:path";

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
      "/v1": "http://127.0.0.1:6500",
      "/healthz": "http://127.0.0.1:6500",
    },
  },
  build: {
    // Built UI is gitignored; only static/fallback.html is committed for embed.
    outDir: path.resolve(__dirname, "../internal/httpapi/static/app"),
    emptyOutDir: true,
    assetsDir: "assets",
  },
});
