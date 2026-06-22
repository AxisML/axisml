import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import path from "node:path";

// SPA build → dist/. Dev server proxies /api to the platform-backend so the
// generated fetch client (baseUrl "") talks to a real backend in development.
//
// Set VITE_USE_MOCK_API=true to serve the whole app from an in-browser mock
// (src/api/mock) — the frontend then never touches the backend or the proxy.
export default defineConfig({
  plugins: [react(), tailwindcss()],
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
