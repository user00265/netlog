<script lang="ts">
  import { auth } from "./lib/stores/auth.svelte";
  import { router } from "./lib/router.svelte";
  import { startSync, stopSync, syncState } from "./lib/sync.svelte";
  import { checkForUpdate } from "./lib/pwa.svelte";
  import TopBar from "./lib/components/TopBar.svelte";
  import Footer from "./lib/components/Footer.svelte";
  import UpdateBanner from "./lib/components/UpdateBanner.svelte";
  import Login from "./routes/Login.svelte";
  import Register from "./routes/Register.svelte";
  import NetList from "./routes/NetList.svelte";
  import NetView from "./routes/NetView.svelte";
  import Admin from "./routes/Admin.svelte";
  import Settings from "./routes/Settings.svelte";

  // Resolve session + bootstrap on first load.
  $effect(() => {
    void auth.load();
  });

  // Start syncing (and the SSE stream) while authenticated; stop on logout so
  // the event stream doesn't keep retrying without a session.
  let syncing = false;
  $effect(() => {
    if (auth.user && !syncing) {
      syncing = true;
      void startSync();
    } else if (!auth.user && syncing) {
      syncing = false;
      stopSync();
    }
  });

  // When the backend becomes reachable again (reconnect, or initial load), check
  // for a newer SPA build so a deploy is picked up promptly.
  let wasReachable = false;
  $effect(() => {
    if (syncState.reachable && !wasReachable) {
      wasReachable = true;
      void checkForUpdate();
    } else if (!syncState.reachable) {
      wasReachable = false;
    }
  });

  const route = $derived(router.matched);
</script>

<UpdateBanner />

{#if !auth.ready}
  <div
    class="flex min-h-dvh items-center justify-center text-zinc-400"
    role="status"
  >
    <span class="h-2 w-2 animate-pulse rounded-full bg-accent-500"></span>
    <span class="sr-only">Loading NetLog…</span>
  </div>
{:else if !auth.user}
  {#if auth.bootstrap?.needsFirstAdmin}
    <Register />
  {:else}
    <Login />
  {/if}
{:else}
  <div class="flex min-h-dvh flex-col">
    <TopBar />
    <main class="flex-1">
      {#if route.name === "net" && route.id}
        {#key route.id}
          <NetView id={route.id} />
        {/key}
      {:else if route.name === "admin" && auth.isAdmin}
        <Admin />
      {:else if route.name === "settings"}
        <Settings />
      {:else}
        <NetList />
      {/if}
    </main>
    <Footer />
  </div>
{/if}
