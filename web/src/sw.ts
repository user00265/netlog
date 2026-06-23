/// <reference lib="webworker" />
import { clientsClaim } from "workbox-core";
import { createHandlerBoundToURL, precacheAndRoute } from "workbox-precaching";
import { NavigationRoute, registerRoute } from "workbox-routing";

declare const self: ServiceWorkerGlobalScope;

// vite-plugin-pwa injects the precache manifest here at build time. A new deploy
// changes this manifest, so the browser installs a fresh worker (precaching the
// new SPA) and the app is prompted to reload (see lib/pwa.svelte.ts).
precacheAndRoute(self.__WB_MANIFEST);

// Serve the precached app shell for every navigation (initial load, refresh,
// deep link) straight from the cache, without touching the network. The SPA's
// client-side router then resolves the path in the browser.
//
// This is what makes the app survive an unreachable backend: a refresh on, say,
// /nets/abc123 matches no precached URL, so without this route it would fall
// through to the network — and a reverse proxy in front of a down backend
// answers with its own 5xx page (e.g. Cloudflare's 502), which is a real HTTP
// response, not a network error, so it would be shown to the operator instead
// of the offline-capable SPA. Routing navigations to the cached shell keeps
// that proxy error from ever reaching the page. /api/* is excluded so data
// requests still hit the network (and queue in the outbox when offline).
registerRoute(
  new NavigationRoute(createHandlerBoundToURL("index.html"), {
    denylist: [/^\/api\//],
  }),
);

// Take control of open pages as soon as we activate, so an applied update serves
// the new assets immediately.
clientsClaim();

// The app asks the waiting worker to activate (apply the update) by posting
// SKIP_WAITING; vite-plugin-pwa then reloads the page on controllerchange.
self.addEventListener("message", (event: ExtendableMessageEvent) => {
  if (event.data?.type === "SKIP_WAITING") self.skipWaiting();
});

// Outbox flush is signalled from the Background Sync API; the app performs the
// actual Dexie reads/writes, so the SW just wakes the clients.
self.addEventListener("sync", (event: Event) => {
  const e = event as Event & {
    tag: string;
    waitUntil(p: Promise<unknown>): void;
  };
  if (e.tag === "netlog-flush-outbox") {
    e.waitUntil(
      self.clients.matchAll().then((clients) => {
        for (const client of clients)
          client.postMessage({ type: "FLUSH_OUTBOX" });
      }),
    );
  }
});
