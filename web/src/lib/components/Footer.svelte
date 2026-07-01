<script lang="ts">
  import { auth } from "../stores/auth.svelte";
  import SyncBadge from "./SyncBadge.svelte";
  import Heart from "@lucide/svelte/icons/heart";

  const version = $derived(auth.bootstrap?.version ?? "dev");
  const commit = $derived(auth.bootstrap?.commit ?? "unknown");
  const build = __FRONTEND_BUILD__;
</script>

<footer class="mt-auto border-t border-zinc-200 dark:border-zinc-800">
  <div
    class="mx-auto flex max-w-5xl flex-col items-center gap-3 px-4 py-5 text-center text-xs text-zinc-500 sm:flex-row sm:justify-between sm:text-left dark:text-zinc-400"
  >
    <div class="space-y-0.5">
      <p class="font-medium text-zinc-600 dark:text-zinc-300">
        NetLog — a ham radio net log for NCS
      </p>
      <p class="flex items-center justify-center gap-1.5 sm:justify-start">
        Made with
        <Heart
          class="h-3.5 w-3.5 text-accent-500"
          fill="currentColor"
          aria-label="love"
        />
        in the
        <span
          class="fi fi-us h-3 w-4 rounded-[1px] ring-1 ring-black/10"
          aria-label="United States"
        ></span>
      </p>
      <p class="font-mono text-[0.7rem] text-zinc-400 dark:text-zinc-500">
        Version {version} (commit: {commit} / build: {build})
      </p>
    </div>

    <SyncBadge />
  </div>
</footer>
