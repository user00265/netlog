<script lang="ts">
  import { pwaState, applyUpdate } from "../pwa.svelte";
  import Button from "./Button.svelte";

  let busy = $state(false);

  async function reload() {
    busy = true;
    await applyUpdate();
  }
</script>

{#if pwaState.needRefresh}
  <div
    class="fixed inset-x-0 bottom-4 z-40 flex justify-center px-4"
    role="status"
  >
    <div
      class="nl-card flex items-center gap-3 px-4 py-2.5 text-sm shadow-lg ring-1 ring-accent-500/30"
    >
      <span class="nl-live-dot"></span>
      <span>A new version of NetLog is available.</span>
      <Button variant="primary" size="sm" onclick={reload} disabled={busy}>
        {busy ? "Updating…" : "Reload"}
      </Button>
    </div>
  </div>
{/if}
