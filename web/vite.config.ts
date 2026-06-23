import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import tailwindcss from "@tailwindcss/vite";
import { VitePWA } from "vite-plugin-pwa";

// A short, per-build identifier shown in the footer (separate from the backend
// version/commit, which are stamped by GoReleaser).
const frontendBuild = Math.floor(Date.now() / 1000).toString(36);

// The SPA is built to web/dist and embedded into the Go binary (see embed.go).
// During development `pnpm dev` proxies /api to the Go backend on :8080.
export default defineConfig({
  define: {
    __FRONTEND_BUILD__: JSON.stringify(frontendBuild),
  },
  plugins: [
    svelte(),
    tailwindcss(),
    VitePWA({
      strategies: "injectManifest",
      srcDir: "src",
      filename: "sw.ts",
      injectRegister: null, // we register manually in src/lib/pwa.ts
      manifest: {
        name: "NetLog",
        short_name: "NetLog",
        description: "Offline-first amateur radio directed-net logging",
        theme_color: "#0a0a0a",
        background_color: "#0a0a0a",
        display: "standalone",
        start_url: "/",
        scope: "/",
        icons: [
          { src: "/icon-192.png", sizes: "192x192", type: "image/png" },
          { src: "/icon-512.png", sizes: "512x512", type: "image/png" },
        ],
      },
      injectManifest: {
        // Precache the app shell only. The many flag SVGs are cosmetic and load
        // at runtime, so they're kept out of the precache to keep installs small.
        globPatterns: ["**/*.{js,css,html,woff2}"],
      },
      devOptions: { enabled: false },
    }),
  ],
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
});
