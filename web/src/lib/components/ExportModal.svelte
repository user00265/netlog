<script lang="ts">
  import { db } from "../db";
  import { auth } from "../stores/auth.svelte";
  import { exportNetCSV, exportNetPDF } from "../export";
  import type { CallsignData } from "../types";
  import Button from "./Button.svelte";

  let {
    netId,
    netName,
    onClose,
  }: { netId: string; netName: string; onClose: () => void } = $props();

  let firstBtn = $state<HTMLButtonElement | null>(null);
  $effect(() => firstBtn?.focus());

  async function gather() {
    const net = await db.nets.get(netId);
    const checkins = await db.checkins
      .where("netId")
      .equals(netId)
      .sortBy("seq");
    const calls = await db.callsigns.toArray();
    const map = new Map<string, CallsignData>(
      calls.map((c) => [c.callsign, c]),
    );
    return { net, checkins, map };
  }

  async function csv() {
    const { net, checkins, map } = await gather();
    if (net) exportNetCSV(net, checkins, map);
    onClose();
  }

  async function pdf() {
    const { net, checkins, map } = await gather();
    if (net) exportNetPDF(net, checkins, map, auth.user);
    onClose();
  }
</script>

<svelte:window onkeydown={(e) => e.key === "Escape" && onClose()} />

<div class="fixed inset-0 z-30 flex items-center justify-center p-4">
  <button
    class="absolute inset-0 bg-black/50"
    aria-label="Close"
    onclick={onClose}
  ></button>
  <div
    class="nl-card relative z-10 w-full max-w-sm p-5"
    role="dialog"
    aria-modal="true"
    aria-labelledby="export-title"
  >
    <h2 id="export-title" class="text-lg font-semibold">Export net</h2>
    <p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
      {netName} — choose a format.
    </p>

    <div class="mt-4 grid grid-cols-2 gap-3">
      <button
        bind:this={firstBtn}
        class="nl-card flex flex-col items-center gap-2 p-4 transition hover:border-zinc-400 dark:hover:border-zinc-600"
        onclick={csv}
      >
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.6"
          class="h-7 w-7 text-emerald-600"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            d="M14 3v5h5M8 13h8M8 17h8M14 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"
          />
        </svg>
        <span class="text-sm font-semibold">CSV</span>
        <span class="text-xs text-zinc-500 dark:text-zinc-400">Spreadsheet</span
        >
      </button>
      <button
        class="nl-card flex flex-col items-center gap-2 p-4 transition hover:border-zinc-400 dark:hover:border-zinc-600"
        onclick={pdf}
      >
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.6"
          class="h-7 w-7 text-accent-600"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            d="M14 3v5h5M14 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8zM9 13h1.5a1.5 1.5 0 0 1 0 3H9v-3zm0 0v5m4-5h2m-2 0v5m0-2.5h1.5"
          />
        </svg>
        <span class="text-sm font-semibold">PDF</span>
        <span class="text-xs text-zinc-500 dark:text-zinc-400">Print-ready</span
        >
      </button>
    </div>

    <div class="mt-4 flex justify-end">
      <Button variant="gray" onclick={onClose}>Cancel</Button>
    </div>
  </div>
</div>
