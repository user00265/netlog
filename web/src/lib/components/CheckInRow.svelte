<script lang="ts">
  import type { CheckIn, CallsignData } from "../types";
  import { editCheckIn, removeCheckIn, refreshCallsign } from "../sync.svelte";
  import { zulu } from "../datetime";
  import Flag from "./Flag.svelte";
  import Button from "./Button.svelte";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import Pencil from "@lucide/svelte/icons/pencil";
  import Trash2 from "@lucide/svelte/icons/trash-2";

  let {
    checkin,
    callbook,
    editable,
    canManage = false,
  }: {
    checkin: CheckIn;
    callbook?: CallsignData;
    // editable: net is open AND the user controls it — gates field edits.
    editable: boolean;
    // canManage: the user controls the net (admin/controller) — gates removal,
    // which (unlike editing) is allowed even after the net is closed.
    canManage?: boolean;
  } = $props();

  let editing = $state(false);
  let confirmDelete = $state(false);
  let refreshing = $state(false);

  let eCall = $state("");
  let eNick = $state("");
  let eTraffic = $state(false);
  let eShort = $state(false);

  function startEdit() {
    eCall = checkin.callsign;
    eNick = checkin.nickname;
    eTraffic = checkin.hasTraffic;
    eShort = checkin.shortTime;
    editing = true;
  }

  async function save(e: SubmitEvent) {
    e.preventDefault();
    await editCheckIn({
      ...checkin,
      callsign: eCall.toUpperCase().trim(),
      nickname: eNick.trim(),
      hasTraffic: eTraffic,
      shortTime: eShort,
    });
    editing = false;
  }

  function onEditKeydown(e: KeyboardEvent) {
    if (e.key === "Escape") {
      e.preventDefault();
      editing = false; // Esc cancels the edit
    }
  }

  // Enter on a toggle pill saves the edit (matching the text fields) instead of
  // flipping the pill; Space still toggles it via the native button activation.
  function onTogglePillKeydown(e: KeyboardEvent) {
    if (e.key === "Enter") {
      e.preventDefault();
      (e.currentTarget as HTMLButtonElement).form?.requestSubmit();
    }
  }

  async function refresh() {
    refreshing = true;
    try {
      await refreshCallsign(checkin.callsign);
    } catch {
      /* surfaced via sync status */
    } finally {
      refreshing = false;
    }
  }

  function toggleClass(on: boolean, accent: "blue" | "amber"): string {
    if (!on)
      return "border border-zinc-300 text-zinc-600 dark:border-zinc-600 dark:text-zinc-300";
    return accent === "blue"
      ? "nl-tag-blue ring-1 ring-blue-500/60"
      : "nl-tag-amber ring-1 ring-amber-500/60";
  }

  // Callbook display name only (no country, per the logging-screen preference).
  const name = $derived(
    callbook
      ? [callbook.firstName, callbook.lastName].filter(Boolean).join(" ")
      : "",
  );
</script>

<li class="flex items-center gap-2 px-3 py-2.5 sm:gap-3 sm:px-4">
  <span class="w-6 shrink-0 text-right text-xs text-zinc-400 tabular-nums"
    >{checkin.seq}</span
  >
  <Flag code={callbook?.flagIso2 ?? ""} title={callbook?.country ?? ""} />

  {#if editing}
    <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
    <form
      class="flex flex-1 flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center"
      onsubmit={save}
      onkeydown={onEditKeydown}
    >
      <input
        bind:value={eCall}
        class="nl-input w-full font-mono uppercase sm:w-28"
        aria-label="Callsign"
        autocapitalize="characters"
      />
      <input
        bind:value={eNick}
        class="nl-input w-full sm:w-36"
        placeholder="Name"
        aria-label="Name on air"
      />
      <div class="flex flex-wrap items-center gap-2">
        <button
          type="button"
          class={`nl-tag px-2 py-1 ${toggleClass(eTraffic, "blue")}`}
          aria-pressed={eTraffic}
          onkeydown={onTogglePillKeydown}
          onclick={() => (eTraffic = !eTraffic)}>Traffic</button
        >
        <button
          type="button"
          class={`nl-tag px-2 py-1 ${toggleClass(eShort, "amber")}`}
          aria-pressed={eShort}
          onkeydown={onTogglePillKeydown}
          onclick={() => (eShort = !eShort)}>Short</button
        >
        <!-- Refresh callbook data lives here in edit mode, left of Save. -->
        <button
          type="button"
          class="nl-icon-btn"
          onclick={refresh}
          disabled={refreshing}
          title="Refresh callbook data"
          aria-label="Refresh callbook data"
        >
          <RefreshCw class={`h-4 w-4 ${refreshing ? "animate-spin" : ""}`} />
        </button>
        <Button variant="green" size="sm" type="submit">Save</Button>
        <Button variant="gray" size="sm" onclick={() => (editing = false)}
          >Cancel</Button
        >
      </div>
    </form>
  {:else}
    <div class="flex min-w-0 flex-1 flex-col">
      <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
        <span class="nl-call">{checkin.callsign}</span>
        {#if checkin.nickname}
          <span class="text-sm text-zinc-600 dark:text-zinc-300"
            >{checkin.nickname}</span
          >
        {/if}
        {#if checkin.hasTraffic}<span class="nl-tag nl-tag-blue" title="Traffic"
            >T</span
          >{/if}
        {#if checkin.shortTime}<span
            class="nl-tag nl-tag-amber"
            title="Short-time">S</span
          >{/if}
      </div>
      <!-- Callbook name and check-in time share the second line: name on the
           left, time on the right — freeing the first line's full width for the
           callsign, nickname and tags. -->
      <div class="flex items-center gap-2 text-xs">
        {#if name}
          <span class="truncate text-zinc-500 dark:text-zinc-400">{name}</span>
        {/if}
        <span class="nl-mono ml-auto shrink-0">{zulu(checkin.checkedInAt)}</span
        >
      </div>
    </div>

    {#if confirmDelete}
      <div class="flex shrink-0 items-center gap-1">
        <Button
          variant="primary"
          size="sm"
          onclick={() => removeCheckIn(checkin)}>Remove</Button
        >
        <Button variant="gray" size="sm" onclick={() => (confirmDelete = false)}
          >Cancel</Button
        >
      </div>
    {:else}
      {#if canManage}
        <button
          class="nl-icon-btn text-accent-600/70 hover:bg-accent-500/10 hover:text-accent-600 dark:text-accent-500/70 dark:hover:text-accent-500"
          onclick={() => (confirmDelete = true)}
          title="Remove"
          aria-label="Remove check-in"
        >
          <Trash2 class="h-4 w-4" />
        </button>
      {/if}
      {#if editable}
        <button
          class="nl-icon-btn"
          onclick={startEdit}
          title="Edit"
          aria-label="Edit check-in"
        >
          <Pencil class="h-4 w-4" />
        </button>
      {/if}
    {/if}
  {/if}
</li>
