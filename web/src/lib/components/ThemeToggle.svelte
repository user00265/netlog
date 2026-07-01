<script lang="ts">
  import { theme } from "../stores/theme.svelte";
  import Sun from "@lucide/svelte/icons/sun";
  import Moon from "@lucide/svelte/icons/moon";
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
    <Sun class="h-5 w-5" />
  {:else}
    <Moon class="h-5 w-5" />
  {/if}
</button>
