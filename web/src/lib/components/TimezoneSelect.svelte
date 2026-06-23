<script lang="ts">
  // A timezone picker backed by the browser's IANA zone list, with a plain-text
  // fallback for older engines.
  let { value = $bindable(""), id }: { value: string; id?: string } = $props();

  let zones = $state<string[]>([]);
  try {
    const supported = (
      Intl as unknown as { supportedValuesOf?: (k: string) => string[] }
    ).supportedValuesOf;
    zones = supported ? supported("timeZone") : [];
  } catch {
    zones = [];
  }
</script>

{#if zones.length}
  <select {id} bind:value class="nl-input">
    {#each zones as z (z)}
      <option value={z}>{z}</option>
    {/each}
  </select>
{:else}
  <input
    {id}
    bind:value
    class="nl-input"
    placeholder="e.g. America/New_York"
    aria-label="Timezone"
  />
{/if}
