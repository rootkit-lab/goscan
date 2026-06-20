import path from "node:path";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const port = Number(process.env.WAILS_VITE_PORT) || 9280;

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { "@": path.resolve(__dirname, "./src") }
  },
  server: { port, strictPort: true, host: "127.0.0.1" },
  build: { outDir: "dist", emptyOutDir: true }
});
