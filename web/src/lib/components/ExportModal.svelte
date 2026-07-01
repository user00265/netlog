<script lang="ts">
  import { db } from "../db";
  import { auth } from "../stores/auth.svelte";
  import { exportNetCSV, exportNetPDF } from "../export";
  import type { CallsignData } from "../types";
  import Button from "./Button.svelte";
  import FileSpreadsheet from "@lucide/svelte/icons/file-spreadsheet";
  import FileText from "@lucide/svelte/icons/file-text";

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
        <FileSpreadsheet class="h-7 w-7 text-emerald-600" strokeWidth={1.6} />
        <span class="text-sm font-semibold">CSV</span>
        <span class="text-xs text-zinc-500 dark:text-zinc-400">Spreadsheet</span
        >
      </button>
      <button
        class="nl-card flex flex-col items-center gap-2 p-4 transition hover:border-zinc-400 dark:hover:border-zinc-600"
        onclick={pdf}
      >
        <FileText class="h-7 w-7 text-accent-600" strokeWidth={1.6} />
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
