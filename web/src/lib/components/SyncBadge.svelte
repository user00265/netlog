<script lang="ts">
  import { syncState } from "../sync.svelte";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import Wifi from "@lucide/svelte/icons/wifi";
  import WifiOff from "@lucide/svelte/icons/wifi-off";

  // --- Sync indicator: green = synced, amber + spinning = syncing, red = not synced.
  type SyncTone = "synced" | "syncing" | "error";
  const syncTone: SyncTone = $derived(
    syncState.syncing
      ? "syncing"
      : syncState.pending === 0 && !syncState.error
        ? "synced"
        : "error",
  );
  const syncColor = $derived(
    syncTone === "synced"
      ? "text-emerald-600 dark:text-emerald-500"
      : syncTone === "syncing"
        ? "text-amber-500"
        : "text-accent-600 dark:text-accent-500",
  );
  const syncTitle = $derived(
    syncTone === "synced"
      ? "Synced"
      : syncTone === "syncing"
        ? "Syncing…"
        : syncState.error ||
          `${syncState.pending} change${syncState.pending === 1 ? "" : "s"} waiting to sync`,
  );

  // --- Connectivity indicator: green wifi = backend reachable, red wifi-off = offline/unreachable.
  const connected = $derived(syncState.online && syncState.reachable);
  const wifiColor = $derived(
    connected
      ? "text-emerald-600 dark:text-emerald-500"
      : "text-accent-600 dark:text-accent-500",
  );
  const wifiTitle = $derived(
    connected
      ? "Online — server reachable"
      : "Offline — can't reach the server",
  );
</script>

<div class="flex items-center gap-2.5">
  <!-- Sync status -->
  <span
    class={syncColor}
    title={syncTitle}
    role="img"
    aria-label={`Sync status: ${syncTitle}`}
  >
    <RefreshCw
      class={`h-4 w-4 ${syncTone === "syncing" ? "animate-spin" : ""}`}
    />
  </span>

  <!-- Connectivity -->
  <span class={wifiColor} title={wifiTitle} role="img" aria-label={wifiTitle}>
    {#if connected}
      <Wifi class="h-4 w-4" />
    {:else}
      <WifiOff class="h-4 w-4" />
    {/if}
  </span>
</div>
