// Service-worker registration + app-update handling.
//
// When a new version is deployed, the precache manifest in sw.js changes, so the
// browser installs a fresh worker that downloads the new SPA in the background.
// We surface that as `pwaState.needRefresh` and let the operator reload when it's
// convenient (so an active net isn't interrupted mid-logging). Applying the
// update flushes the outbox first, then activates the new worker and reloads —
// the reloaded app re-syncs on startup.
import { registerSW } from "virtual:pwa-register";
import { flushOutbox } from "./sync.svelte";

export const pwaState = $state({ needRefresh: false });

let updateSW: ((reloadPage?: boolean) => Promise<void>) | undefined;
let registration: ServiceWorkerRegistration | undefined;

export function setupPWA(): void {
  if (typeof navigator === "undefined" || !("serviceWorker" in navigator))
    return;

  updateSW = registerSW({
    immediate: true,
    onNeedRefresh() {
      // A new SPA build is installed and waiting.
      pwaState.needRefresh = true;
    },
    onRegisteredSW(_swScriptUrl, reg) {
      registration = reg;
      // Background Sync: flush the outbox on reconnect even if unfocused.
      if (reg && "sync" in reg) {
        (
          reg as ServiceWorkerRegistration & {
            sync: { register(tag: string): Promise<void> };
          }
        ).sync
          .register("netlog-flush-outbox")
          .catch(() => {
            /* Background Sync unavailable; the online-event flusher covers us. */
          });
      }
    },
  });
}

// applyUpdate flushes pending changes, then activates the new worker and reloads
// to the new SPA.
export async function applyUpdate(): Promise<void> {
  try {
    await flushOutbox();
  } catch {
    // The outbox persists in IndexedDB across the reload, so this is best-effort.
  }
  pwaState.needRefresh = false;
  await updateSW?.(true);
}

// checkForUpdate asks the browser to re-check for a new service worker. Call it
// when connectivity returns so a new deploy is picked up promptly rather than
// only on the next navigation.
export async function checkForUpdate(): Promise<void> {
  try {
    await registration?.update();
  } catch {
    /* offline or transient; ignore */
  }
}
