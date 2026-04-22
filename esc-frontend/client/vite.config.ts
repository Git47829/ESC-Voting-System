import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      "/crud-api": {
        target: "http://localhost:8000",
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/crud-api/, "")
      },
      "/eurostats": {
        target: "http://localhost:8880",
        ws: true,
        changeOrigin: true
      }
    }
  }
});
