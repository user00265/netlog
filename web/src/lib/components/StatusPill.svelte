<script lang="ts">
  import type { NetStatus } from "../types";
  let { status }: { status: NetStatus } = $props();
</script>

{#if status === "open"}
  <span class="nl-pill nl-pill-live"><span class="nl-live-dot"></span>Live</span
  >
{:else if status === "pending"}
  <span class="nl-pill nl-pill-pending">Scheduled</span>
{:else}
  <span class="nl-pill nl-pill-closed">Closed</span>
{/if}
