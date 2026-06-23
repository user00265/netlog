<script lang="ts">
  import { liveQuery } from "dexie";
  import { db } from "../lib/db";
  import { api } from "../lib/api";
  import { auth } from "../lib/stores/auth.svelte";
  import { setNetStatus, loadCallsign } from "../lib/sync.svelte";
  import { link } from "../lib/router.svelte";
  import { formatDate, elapsed } from "../lib/format";
  import { dual } from "../lib/datetime";
  import Button from "../lib/components/Button.svelte";
  import StatusPill from "../lib/components/StatusPill.svelte";
  import CheckInForm from "../lib/components/CheckInForm.svelte";
  import CheckInRow from "../lib/components/CheckInRow.svelte";
  import NetNotes from "../lib/components/NetNotes.svelte";

  let { id }: { id: string } = $props();

  const netQ = liveQuery(() => db.nets.get(id));
  const checkinsQ = liveQuery(() =>
    db.checkins.where("netId").equals(id).sortBy("seq"),
  );
  const callsignsQ = liveQuery(() => db.callsigns.toArray());

  const net = $derived($netQ ?? null);
  const checkins = $derived($checkinsQ ?? []);
  const callsignMap = $derived(
    new Map(($callsignsQ ?? []).map((c) => [c.callsign, c])),
  );

  const canManage = $derived(
    !!net && !!auth.user && (auth.isAdmin || auth.user.id === net.ncsUserId),
  );
  const editable = $derived(!!net && net.status === "open" && canManage);

  // A 30s tick keeps the running-duration label fresh while a net is live.
  let tick = $state(0);
  $effect(() => {
    const t = setInterval(() => (tick = tick + 1), 30_000);
    return () => clearInterval(t);
  });
  const runtime = $derived.by(() => {
    void tick; // re-evaluate on each tick so a live net's duration updates
    return net ? elapsed(net.startAt, net.endAt) : "—";
  });

  // If the net isn't in the local store yet (e.g. a direct link), fetch it.
  $effect(() => {
    if ($netQ === undefined) {
      api
        .getNet(id)
        .then(async (detail) => {
          const { checkins: cis, ...netFields } = detail;
          await db.nets.put(netFields);
          await db.checkins.bulkPut(cis);
        })
        .catch(() => {
          /* offline or gone; the not-found view handles it */
        });
    }
  });

  // Ensure callbook data is available locally for every check-in's callsign.
  $effect(() => {
    const have = callsignMap;
    for (const c of checkins) {
      if (!have.has(c.callsign)) void loadCallsign(c.callsign);
    }
  });
</script>

{#if net === null}
  <div class="mx-auto max-w-3xl px-4 py-16 text-center">
    <p class="text-zinc-600 dark:text-zinc-300">Net not found.</p>
    <a
      href="/"
      use:link
      class="mt-3 inline-block text-sm text-accent-600 dark:text-accent-500"
      >← Back to nets</a
    >
  </div>
{:else}
  <div class="mx-auto max-w-3xl px-4 py-6">
    <a
      href="/"
      use:link
      class="mb-3 inline-block text-sm text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300"
      >← All nets</a
    >

    <!-- Net header -->
    <div class="nl-card mb-5 p-4 sm:p-5">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div class="flex items-center gap-2">
            <h1 class="text-xl font-bold">{net.name}</h1>
            <StatusPill status={net.status} />
          </div>
          <p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
            {formatDate(net.netDate)} · NCS
            <span class="nl-call">{net.ncsCallsign || "—"}</span>
          </p>
        </div>

        {#if canManage}
          <div class="flex gap-2">
            {#if net.status === "pending"}
              <Button variant="green" onclick={() => setNetStatus(net, "open")}
                >Open net</Button
              >
            {:else if net.status === "open"}
              <Button
                variant="yellow"
                onclick={() => setNetStatus(net, "closed")}>Close net</Button
              >
            {/if}
          </div>
        {/if}
      </div>

      <dl class="mt-4 grid grid-cols-2 gap-3 text-sm sm:grid-cols-4">
        <div>
          <dt
            class="text-xs uppercase tracking-wide text-zinc-500 dark:text-zinc-400"
          >
            Started
          </dt>
          <dd class="nl-mono">{dual(net.startAt)}</dd>
        </div>
        <div>
          <dt
            class="text-xs uppercase tracking-wide text-zinc-500 dark:text-zinc-400"
          >
            Ended
          </dt>
          <dd class="nl-mono">{dual(net.endAt)}</dd>
        </div>
        <div>
          <dt
            class="text-xs uppercase tracking-wide text-zinc-500 dark:text-zinc-400"
          >
            Duration
          </dt>
          <dd class="nl-mono">{runtime}</dd>
        </div>
        <div>
          <dt
            class="text-xs uppercase tracking-wide text-zinc-500 dark:text-zinc-400"
          >
            Check-ins
          </dt>
          <dd class="nl-mono">{checkins.length}</dd>
        </div>
      </dl>
    </div>

    <!-- NCS notes: editable while open, read-only otherwise. -->
    <div class="mb-5">
      {#key net.id}
        <NetNotes {net} {editable} />
      {/key}
    </div>

    <div class="grid grid-cols-1 gap-5 lg:grid-cols-[20rem_1fr]">
      <!-- Logging form, only while open & permitted -->
      <div class="order-2 lg:order-1">
        {#if editable}
          <CheckInForm netId={net.id} />
        {:else if net.status === "closed"}
          <div class="nl-card p-4 text-sm text-zinc-500 dark:text-zinc-400">
            This net is closed and read-only.
          </div>
        {:else if net.status === "pending"}
          <div class="nl-card p-4 text-sm text-zinc-500 dark:text-zinc-400">
            Open the net to begin logging check-ins.
          </div>
        {:else}
          <div class="nl-card p-4 text-sm text-zinc-500 dark:text-zinc-400">
            Only the Net Control operator can log check-ins.
          </div>
        {/if}
      </div>

      <!-- Check-in log -->
      <div class="order-1 lg:order-2">
        <div class="nl-card overflow-hidden">
          <div
            class="border-b border-zinc-200 px-4 py-2.5 text-sm font-semibold dark:border-zinc-800"
          >
            Check-in log
          </div>
          {#if checkins.length === 0}
            <p
              class="px-4 py-8 text-center text-sm text-zinc-500 dark:text-zinc-400"
            >
              No check-ins logged yet.
            </p>
          {:else}
            <ul class="divide-y divide-zinc-200 dark:divide-zinc-800">
              {#each checkins as c (c.id)}
                <CheckInRow
                  checkin={c}
                  callbook={callsignMap.get(c.callsign)}
                  {editable}
                />
              {/each}
            </ul>
          {/if}
        </div>
      </div>
    </div>
  </div>
{/if}
