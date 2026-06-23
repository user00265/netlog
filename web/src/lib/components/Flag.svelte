<script lang="ts">
  // Renders a DXCC country flag via flag-icons, or a neutral marker when the
  // entity has no mapped flag (e.g. an international organization).
  let {
    code = "",
    title = "",
    size = "md",
  }: { code?: string; title?: string; size?: "sm" | "md" } = $props();

  const dim = $derived(size === "sm" ? "h-3.5 w-5" : "h-4 w-6");
</script>

{#if code}
  <span
    class={`fi fi-${code} ${dim} inline-block rounded-[2px] shadow-sm ring-1 ring-black/10`}
    title={title || code.toUpperCase()}
    aria-label={title || code}
  ></span>
{:else}
  <span
    class={`${dim} inline-flex items-center justify-center rounded-[2px] bg-zinc-300 text-[0.6rem] text-zinc-600 dark:bg-zinc-700 dark:text-zinc-300`}
    title={title || "Unknown entity"}
    aria-label="unknown"
  >
    ?
  </span>
{/if}
