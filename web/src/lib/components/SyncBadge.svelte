<script lang="ts">
  import { syncState } from "../sync.svelte";

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
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="2"
      class={`h-4 w-4 ${syncTone === "syncing" ? "animate-spin" : ""}`}
    >
      <path
        stroke-linecap="round"
        stroke-linejoin="round"
        d="M3 12a9 9 0 0 1 15-6.7L21 8"
      />
      <path stroke-linecap="round" stroke-linejoin="round" d="M21 3v5h-5" />
      <path
        stroke-linecap="round"
        stroke-linejoin="round"
        d="M21 12a9 9 0 0 1-15 6.7L3 16"
      />
      <path stroke-linecap="round" stroke-linejoin="round" d="M3 21v-5h5" />
    </svg>
  </span>

  <!-- Connectivity -->
  <span class={wifiColor} title={wifiTitle} role="img" aria-label={wifiTitle}>
    {#if connected}
      <svg
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        class="h-4 w-4"
      >
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 20h.01" />
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          d="M2 8.82a15 15 0 0 1 20 0"
        />
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          d="M5 12.86a10 10 0 0 1 14 0"
        />
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          d="M8.5 16.43a5 5 0 0 1 7 0"
        />
      </svg>
    {:else}
      <svg
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        class="h-4 w-4"
      >
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 20h.01" />
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          d="M8.5 16.43a5 5 0 0 1 7 0"
        />
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          d="M5 12.86a10 10 0 0 1 5.17-2.7"
        />
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          d="M19 12.86a10 10 0 0 0-2.01-1.52"
        />
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          d="M2 8.82a15 15 0 0 1 4.18-2.64"
        />
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          d="M22 8.82a15 15 0 0 0-11.29-3.76"
        />
        <path stroke-linecap="round" stroke-linejoin="round" d="m2 2 20 20" />
      </svg>
    {/if}
  </span>
</div>
