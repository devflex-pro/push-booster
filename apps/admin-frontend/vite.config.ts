import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          charts: ["recharts"],
          react: ["react", "react-dom", "react-router"],
          tanstack: ["@tanstack/react-query", "@tanstack/react-table"]
        }
      }
    }
  },
  server: {
    port: 5173,
    proxy: {
      "/api/subscribe": {
        target: "http://localhost:8082",
        changeOrigin: true
      },
      "/api/web-push": {
        target: "http://localhost:8082",
        changeOrigin: true
      },
      "/api/sdk": {
        target: "http://localhost:8082",
        changeOrigin: true
      },
      "/api/push": {
        target: "http://localhost:8083",
        changeOrigin: true
      },
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true
      }
    }
  }
});
