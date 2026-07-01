<script lang="ts">
  import { liveQuery } from "dexie";
  import { db } from "../lib/db";
  import { auth } from "../lib/stores/auth.svelte";
  import { createNet, deleteNet } from "../lib/sync.svelte";
  import { navigate, link } from "../lib/router.svelte";
  import { formatDate, todayDate } from "../lib/format";
  import { dual } from "../lib/datetime";
  import type { Net } from "../lib/types";
  import Button from "../lib/components/Button.svelte";
  import StatusPill from "../lib/components/StatusPill.svelte";
  import ExportModal from "../lib/components/ExportModal.svelte";
  import Upload from "@lucide/svelte/icons/upload";
  import Trash2 from "@lucide/svelte/icons/trash-2";

  let exportNet = $state<{ id: string; name: string } | null>(null);
  let confirmDeleteId = $state<string | null>(null);

  async function doDelete(n: Net) {
    confirmDeleteId = null;
    await deleteNet(n);
  }

  const netsQ = liveQuery(() => db.nets.toArray());
  const checkinsQ = liveQuery(() => db.checkins.toArray());

  let showForm = $state(false);
  let name = $state("");
  let netDate = $state(todayDate());
  let creating = $state(false);

  // Count check-ins per net.
  const counts = $derived.by(() => {
    const map = new Map<string, number>();
    for (const c of $checkinsQ ?? [])
      map.set(c.netId, (map.get(c.netId) ?? 0) + 1);
    return map;
  });

  // Sort: active nets (no end time) first, then most-recently-ended.
  const nets = $derived.by(() => {
    const list = ($netsQ ?? []).slice();
    list.sort((a, b) => {
      if (!a.endAt && b.endAt) return -1;
      if (a.endAt && !b.endAt) return 1;
      if (a.endAt && b.endAt) return a.endAt < b.endAt ? 1 : -1;
      return a.createdAt < b.createdAt ? 1 : -1;
    });
    return list;
  });

  let error = $state("");

  async function create(e: SubmitEvent) {
    e.preventDefault();
    if (!auth.user || !name.trim()) return;
    creating = true;
    error = "";
    try {
      const id = await createNet(name.trim(), netDate);
      navigate(`/nets/${id}`);
    } catch (err) {
      error = err instanceof Error ? err.message : "Could not create the net.";
    } finally {
      creating = false;
    }
  }

  function href(n: Net): string {
    return `/nets/${n.id}`;
  }
</script>

<div class="mx-auto max-w-5xl px-4 py-6">
  <div class="mb-5 flex items-center justify-between gap-3">
    <div>
      <h1 class="text-xl font-bold">Nets</h1>
      <p class="text-sm text-zinc-500 dark:text-zinc-400">
        Directed net logs, newest activity first.
      </p>
    </div>
    <Button variant="primary" onclick={() => (showForm = !showForm)}>
      {showForm ? "Cancel" : "+ New"}
    </Button>
  </div>

  {#if showForm}
    <form
      onsubmit={create}
      class="nl-card mb-5 flex flex-col gap-3 p-4 sm:flex-row sm:items-end"
    >
      <div class="flex-1">
        <label class="nl-label" for="net-name">Net name</label>
        <input
          id="net-name"
          bind:value={name}
          class="nl-input"
          placeholder="Tuesday Evening Net"
          maxlength="200"
          required
        />
      </div>
      <div>
        <label class="nl-label" for="net-date">Date</label>
        <input
          id="net-date"
          type="date"
          bind:value={netDate}
          class="nl-input"
          required
        />
      </div>
      <Button type="submit" variant="green" disabled={creating}>
        {creating ? "Creating…" : "Create net"}
      </Button>
    </form>
    {#if error}
      <p class="mb-5 text-sm text-accent-600 dark:text-accent-500">{error}</p>
    {/if}
  {/if}

  {#if nets.length === 0}
    <div class="nl-card p-10 text-center">
      <p class="text-zinc-600 dark:text-zinc-300">No nets yet.</p>
      <p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
        Create a net to open it and start logging check-ins.
      </p>
    </div>
  {:else}
    <div class="nl-card overflow-hidden">
      <!-- Column header (desktop only) — mirrors the row: grid + export spacer. -->
      <div
        class="hidden border-b border-zinc-200 px-4 py-2 text-xs font-semibold uppercase tracking-wide text-zinc-500 lg:flex dark:border-zinc-800 dark:text-zinc-400"
      >
        <div class="grid flex-1 grid-cols-[1fr_7rem_6rem_4rem_13rem] gap-3">
          <span class="text-center">Net / NCS</span>
          <span class="text-center">Date</span>
          <span class="text-center">Status</span>
          <span class="text-center">C/I</span>
          <span class="text-center">Ended</span>
        </div>
        <span class="w-24"></span>
      </div>

      <ul class="divide-y divide-zinc-200 dark:divide-zinc-800">
        {#each nets as n (n.id)}
          <li class="flex items-stretch">
            <a
              href={href(n)}
              use:link
              class="grid min-w-0 flex-1 grid-cols-1 gap-1 px-4 py-3 transition hover:bg-zinc-50 lg:grid-cols-[1fr_7rem_6rem_4rem_13rem] lg:items-center lg:gap-3 dark:hover:bg-zinc-800/50"
            >
              <div class="min-w-0 lg:text-center">
                <div class="truncate font-semibold">{n.name}</div>
                <div class="text-xs text-zinc-500 dark:text-zinc-400">
                  NCS <span class="nl-call">{n.ncsCallsign || "—"}</span>
                </div>
              </div>
              <div
                class="text-sm text-zinc-500 lg:text-center lg:text-zinc-700 dark:text-zinc-400 lg:dark:text-zinc-300"
              >
                {formatDate(n.netDate)}
              </div>
              <div class="lg:text-center"><StatusPill status={n.status} /></div>
              <div
                class="text-sm tabular-nums text-zinc-600 lg:text-center dark:text-zinc-300"
              >
                <span class="text-zinc-500 lg:hidden dark:text-zinc-400"
                  >C/I:
                </span>{counts.get(n.id) ?? 0}
              </div>
              <div class="nl-mono text-xs lg:text-center">
                {dual(n.endAt)}
              </div>
            </a>
            <div
              class="flex min-w-24 shrink-0 items-center justify-center gap-1 px-2"
            >
              {#if confirmDeleteId === n.id}
                <Button variant="primary" size="sm" onclick={() => doDelete(n)}
                  >Delete</Button
                >
                <Button
                  variant="gray"
                  size="sm"
                  onclick={() => (confirmDeleteId = null)}>Cancel</Button
                >
              {:else}
                <!-- Fixed slots: an empty spacer stands in for a missing export
                     or delete so both icons keep the same column across rows. -->
                {#if n.status === "closed"}
                  <button
                    class="nl-icon-btn"
                    title="Export net"
                    aria-label="Export net"
                    onclick={() => (exportNet = { id: n.id, name: n.name })}
                  >
                    <Upload class="h-4 w-4" />
                  </button>
                {:else}
                  <span class="h-9 w-9 shrink-0" aria-hidden="true"></span>
                {/if}
                {#if n.canManage}
                  <button
                    class="nl-icon-btn text-accent-600/70 hover:bg-accent-500/10 hover:text-accent-600 dark:text-accent-500/70 dark:hover:text-accent-500"
                    title="Delete net"
                    aria-label="Delete net"
                    onclick={() => (confirmDeleteId = n.id)}
                  >
                    <Trash2 class="h-4 w-4" />
                  </button>
                {:else}
                  <span class="h-9 w-9 shrink-0" aria-hidden="true"></span>
                {/if}
              {/if}
            </div>
          </li>
        {/each}
      </ul>
    </div>
  {/if}

  {#if exportNet}
    <ExportModal
      netId={exportNet.id}
      netName={exportNet.name}
      onClose={() => (exportNet = null)}
    />
  {/if}
</div>
