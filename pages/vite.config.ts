import { defineConfig } from "vite";
import react from "@vitejs/plugin-react-swc";
import path from "node:path";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "src"),
    },
  },
  server: {
    port: 3300,
    proxy: {
      "/api": {
        target: "http://localhost:3301",
        changeOrigin: true,
      },
      "/r": {
        target: "http://localhost:3301",
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: "dist",
  },
});
