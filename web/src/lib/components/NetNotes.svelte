<script lang="ts">
  import { onDestroy } from "svelte";
  import type { Net } from "../types";
  import { setNetNotes } from "../sync.svelte";

  let { net, editable }: { net: Net; editable: boolean } = $props();

  // Local draft owns the textarea while typing so incoming syncs don't clobber
  // keystrokes. The component is keyed by net id in the parent, so the draft
  // re-initializes when switching nets.
  // svelte-ignore state_referenced_locally
  let draft = $state(net.notes ?? "");
  let saved = $state(true);
  let timer: ReturnType<typeof setTimeout> | null = null;

  function onInput() {
    saved = false;
    if (timer) clearTimeout(timer);
    timer = setTimeout(() => {
      void setNetNotes(net, draft);
      saved = true;
      timer = null;
    }, 800);
  }

  // On unmount (e.g. switching nets), cancel the pending debounce and flush any
  // unsaved edit immediately, so the timer can't fire later against a torn-down
  // component or write a stale draft over newer state.
  onDestroy(() => {
    if (timer) {
      clearTimeout(timer);
      timer = null;
      if (!saved) void setNetNotes(net, draft);
    }
  });
</script>

{#if editable}
  <div class="nl-card p-4">
    <div class="mb-2 flex items-center justify-between">
      <h2 class="text-sm font-semibold uppercase tracking-wide text-zinc-500">
        Net notes
      </h2>
      <span class="text-xs text-zinc-500 dark:text-zinc-400"
        >{saved ? "Saved" : "Saving…"}</span
      >
    </div>
    <textarea
      bind:value={draft}
      oninput={onInput}
      rows="4"
      maxlength="20000"
      class="nl-input resize-y"
      placeholder="Running notes for this net — saved automatically."
    ></textarea>
  </div>
{:else if net.notes}
  <div class="nl-card p-4">
    <h2
      class="mb-2 text-sm font-semibold uppercase tracking-wide text-zinc-500"
    >
      Net notes
    </h2>
    <p class="whitespace-pre-wrap text-sm text-zinc-700 dark:text-zinc-300">
      {net.notes}
    </p>
  </div>
{/if}
