<script lang="ts">
  import { auth } from "../stores/auth.svelte";
  import { link, navigate, router } from "../router.svelte";
  import { syncState } from "../sync.svelte";
  import ThemeToggle from "./ThemeToggle.svelte";

  // Account management requires the live backend (we never store credentials or
  // the user list offline), so the Accounts link only shows when connected.
  const connected = $derived(syncState.online && syncState.reachable);

  let menuOpen = $state(false);
  let triggerEl = $state<HTMLButtonElement | null>(null);
  let firstItemEl = $state<HTMLButtonElement | null>(null);

  async function logout() {
    menuOpen = false;
    await auth.logout();
    navigate("/login");
  }

  function go(path: string) {
    menuOpen = false;
    navigate(path);
  }

  function closeMenu() {
    menuOpen = false;
    triggerEl?.focus();
  }

  // Move focus into the menu when it opens for keyboard users.
  $effect(() => {
    if (menuOpen) firstItemEl?.focus();
  });

  const onNets = $derived(
    router.matched.name === "nets" || router.matched.name === "net",
  );
  const onAdmin = $derived(router.matched.name === "admin");

  function navClass(active: boolean): string {
    return active
      ? "text-zinc-900 dark:text-zinc-100"
      : "text-zinc-500 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-100";
  }
</script>

<svelte:window
  onkeydown={(e) => menuOpen && e.key === "Escape" && closeMenu()}
/>

<header
  class="sticky top-0 z-20 border-b border-zinc-200 bg-zinc-100/90 backdrop-blur dark:border-zinc-800 dark:bg-zinc-950/90"
>
  <div class="mx-auto flex max-w-5xl items-center gap-3 px-4 py-3 sm:gap-4">
    <a
      href="/"
      use:link
      class="flex items-center gap-2 font-bold tracking-tight"
    >
      <!-- Morse ".-" (dit-dah) in the brand red, matching the auth screens. -->
      <span class="flex items-center gap-[3px]" aria-hidden="true">
        <span class="h-2 w-2 rounded-full bg-accent-500"></span>
        <span class="h-2 w-5 rounded-full bg-accent-500"></span>
      </span>
      <span class="text-lg">NL</span>
    </a>

    <nav class="flex items-center gap-3 text-sm font-medium sm:gap-4">
      <a href="/" use:link class={navClass(onNets)}>Nets</a>
      {#if auth.isAdmin && connected}
        <a href="/admin" use:link class={navClass(onAdmin)}>Accounts</a>
      {/if}
    </nav>

    <div class="ml-auto flex items-center gap-2">
      <ThemeToggle />

      {#if auth.user}
        <div class="relative">
          <button
            bind:this={triggerEl}
            class="flex items-center gap-1 rounded-lg px-2 py-1.5 text-sm font-medium hover:bg-zinc-200 dark:hover:bg-zinc-800"
            aria-haspopup="menu"
            aria-expanded={menuOpen}
            onclick={() => (menuOpen = !menuOpen)}
          >
            <span class="nl-call">{auth.user.callsign}</span>
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              class="h-4 w-4"
            >
              <path stroke-linecap="round" d="m6 9 6 6 6-6" />
            </svg>
          </button>

          {#if menuOpen}
            <button
              class="fixed inset-0 z-10 cursor-default"
              aria-label="Close menu"
              onclick={closeMenu}
            ></button>
            <div
              class="nl-card absolute right-0 z-20 mt-2 w-44 overflow-hidden p-1 text-sm shadow-lg"
              role="menu"
            >
              <button
                bind:this={firstItemEl}
                class="block w-full rounded px-3 py-2 text-left hover:bg-zinc-100 dark:hover:bg-zinc-800"
                role="menuitem"
                onclick={() => go("/settings")}>Settings</button
              >
              <button
                class="block w-full rounded px-3 py-2 text-left text-accent-600 hover:bg-zinc-100 dark:text-accent-500 dark:hover:bg-zinc-800"
                role="menuitem"
                onclick={logout}>Log out</button
              >
            </div>
          {/if}
        </div>
      {/if}
    </div>
  </div>
</header>
