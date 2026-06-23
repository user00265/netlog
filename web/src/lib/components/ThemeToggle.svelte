<script lang="ts">
  import { theme } from "../stores/theme.svelte";
</script>

<button
  class="nl-icon-btn"
  onclick={() => theme.toggle()}
  title={theme.current === "dark"
    ? "Switch to light mode"
    : "Switch to dark mode"}
  aria-label="Toggle theme"
>
  {#if theme.current === "dark"}
    <!-- sun -->
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="2"
      class="h-5 w-5"
    >
      <circle cx="12" cy="12" r="4" />
      <path
        stroke-linecap="round"
        d="M12 2v2m0 16v2M4.93 4.93l1.41 1.41m11.32 11.32l1.41 1.41M2 12h2m16 0h2M4.93 19.07l1.41-1.41m11.32-11.32l1.41-1.41"
      />
    </svg>
  {:else}
    <!-- moon -->
    <svg viewBox="0 0 24 24" fill="currentColor" class="h-5 w-5">
      <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
    </svg>
  {/if}
</button>
