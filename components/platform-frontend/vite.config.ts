import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "node:path";

// SPA build → dist/. Dev server proxies /api to the platform-backend so the
// generated fetch client (baseUrl "") talks to a real backend in development.
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { "@": path.resolve(__dirname, "src") },
  },
  server: {
    port: 5173,
    proxy: {
      "/api": {
        target: process.env.VITE_API_TARGET || "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
  build: { outDir: "dist", sourcemap: true },
});
