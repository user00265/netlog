<script lang="ts">
  import { addCheckIn } from "../sync.svelte";
  import Button from "./Button.svelte";

  let { netId }: { netId: string } = $props();

  let callsign = $state("");
  let nickname = $state("");
  let hasTraffic = $state(false);
  let shortTime = $state(false);
  let error = $state("");
  let inputEl = $state<HTMLInputElement | null>(null);

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    const call = callsign.toUpperCase().trim();
    if (call.length < 3) {
      error = "Enter a valid callsign.";
      return;
    }
    error = "";
    try {
      await addCheckIn(netId, {
        callsign: call,
        nickname: nickname.trim(),
        hasTraffic,
        shortTime,
      });
    } catch (err) {
      error =
        err instanceof Error ? err.message : "Could not save the check-in.";
      return;
    }
    clearForm();
  }

  // clearForm resets the fields and returns focus to the callsign input so the
  // operator can immediately log the next station.
  function clearForm() {
    callsign = "";
    nickname = "";
    hasTraffic = false;
    shortTime = false;
    error = "";
    inputEl?.focus();
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key === "Escape") {
      e.preventDefault();
      clearForm();
    }
  }

  // Enter on a toggle pill saves the check-in (matching the text fields) instead
  // of flipping the pill, so the operator can tab callsign → name → traffic →
  // short → Save and commit with Enter at any stop. Space still toggles the pill
  // via the native button activation.
  function onTogglePillKeydown(e: KeyboardEvent) {
    if (e.key === "Enter") {
      e.preventDefault();
      (e.currentTarget as HTMLButtonElement).form?.requestSubmit();
    }
  }
</script>

<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<form
  onsubmit={submit}
  onkeydown={onKeydown}
  class="nl-card p-4 sm:sticky sm:top-20"
  aria-label="Log a check-in"
>
  <h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-zinc-500">
    Log a check-in
  </h2>

  <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
    <div>
      <label class="nl-label" for="ci-call">Callsign</label>
      <input
        id="ci-call"
        bind:this={inputEl}
        bind:value={callsign}
        class="nl-input font-mono uppercase tracking-wide"
        placeholder="W1AW"
        maxlength="16"
        autocomplete="off"
        autocapitalize="characters"
        spellcheck="false"
      />
    </div>
    <div>
      <label class="nl-label" for="ci-nick">Name on the air</label>
      <input
        id="ci-nick"
        bind:value={nickname}
        class="nl-input"
        placeholder="Optional"
        maxlength="80"
        autocomplete="off"
      />
    </div>
  </div>

  <div class="mt-3 flex flex-wrap items-center gap-2">
    <button
      type="button"
      class={`nl-tag cursor-pointer px-2.5 py-1 ${hasTraffic ? "nl-tag-blue ring-1 ring-blue-500/60" : "border border-zinc-300 text-zinc-600 dark:border-zinc-600 dark:text-zinc-300"}`}
      aria-pressed={hasTraffic}
      onclick={() => (hasTraffic = !hasTraffic)}
      onkeydown={onTogglePillKeydown}
    >
      Has traffic
    </button>
    <button
      type="button"
      class={`nl-tag cursor-pointer px-2.5 py-1 ${shortTime ? "nl-tag-amber ring-1 ring-amber-500/60" : "border border-zinc-300 text-zinc-600 dark:border-zinc-600 dark:text-zinc-300"}`}
      aria-pressed={shortTime}
      onclick={() => (shortTime = !shortTime)}
      onkeydown={onTogglePillKeydown}
    >
      Short-time
    </button>
    <div class="ml-auto flex items-center gap-2">
      <Button variant="gray" onclick={clearForm}>Clear</Button>
      <Button type="submit" variant="green">Save check-in</Button>
    </div>
  </div>
  <p class="mt-2 text-xs text-zinc-500 dark:text-zinc-400">
    Enter saves · Esc clears
  </p>

  {#if error}
    <p class="mt-2 text-sm text-accent-600 dark:text-accent-500">{error}</p>
  {/if}
</form>
