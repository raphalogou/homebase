import { fileURLToPath, URL } from "node:url";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";
import { VitePWA } from "vite-plugin-pwa";

const server = process.env.HOMEBASE_DEV_SERVER ?? "http://127.0.0.1:8080";

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    VitePWA({
      strategies: "injectManifest",
      srcDir: "src",
      filename: "sw.ts",
      // main.tsx registers the worker itself, in production builds only.
      injectRegister: false,
      manifest: {
        name: "Homebase",
        short_name: "Homebase",
        description: "A calm personal planner for goals, projects and tasks.",
        start_url: "/",
        scope: "/",
        display: "standalone",
        background_color: "#F2F3EF",
        theme_color: "#F2F3EF",
        icons: [
          { src: "/icon-192.png", sizes: "192x192", type: "image/png" },
          { src: "/icon-512.png", sizes: "512x512", type: "image/png" },
          {
            src: "/icon-maskable-512.png",
            sizes: "512x512",
            type: "image/png",
            purpose: "maskable",
          },
        ],
      },
      injectManifest: {
        // The plugin adds the manifest and its icons itself.
        globPatterns: [
          "**/*.{js,css,html,woff2}",
          "apple-touch-icon.png",
          "badge-96.png",
          "icon.svg",
        ],
      },
    }),
  ],
  resolve: {
    alias: { "@": fileURLToPath(new URL("./src", import.meta.url)) },
  },
  server: {
    port: 5173,
    strictPort: true,
    proxy: {
      "/api": server,
      "/healthz": server,
      "/calendar": server,
    },
  },
  build: {
    target: "es2022",
  },
});
