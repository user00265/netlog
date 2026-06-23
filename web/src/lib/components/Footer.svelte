<script lang="ts">
  import { auth } from "../stores/auth.svelte";
  import SyncBadge from "./SyncBadge.svelte";

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
        <svg
          viewBox="0 0 24 24"
          fill="currentColor"
          class="h-3.5 w-3.5 text-accent-500"
          aria-label="love"
        >
          <path
            d="M12 21s-6.7-4.35-9.33-8.07C.9 10.27 1.7 6.6 4.7 5.6c2-.67 3.9.2 5.07 1.77L12 9.5l2.23-2.13C15.4 5.8 17.3 4.93 19.3 5.6c3 .99 3.8 4.67 2.03 7.33C18.7 16.65 12 21 12 21z"
          />
        </svg>
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
