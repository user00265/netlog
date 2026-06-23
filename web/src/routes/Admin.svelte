<script lang="ts">
  import { api, ApiError } from "../lib/api";
  import { link } from "../lib/router.svelte";
  import { syncState } from "../lib/sync.svelte";
  import type { User } from "../lib/types";
  import Button from "../lib/components/Button.svelte";

  // Accounts are managed live only — there is no offline user list or stored
  // credentials, so this whole page requires a reachable backend.
  const connected = $derived(syncState.online && syncState.reachable);

  let users = $state<User[]>([]);
  let error = $state("");
  let busy = $state(false);
  let showForm = $state(false);

  let callsign = $state("");
  let firstName = $state("");
  let lastName = $state("");
  let email = $state("");
  let password = $state("");

  async function refresh() {
    if (!connected) return;
    try {
      users = await api.listUsers();
    } catch (err) {
      error =
        err instanceof ApiError ? err.message : "Could not load operators.";
    }
  }

  // Load (and reload when connectivity returns).
  $effect(() => {
    if (connected) void refresh();
  });

  async function create(e: SubmitEvent) {
    e.preventDefault();
    busy = true;
    error = "";
    try {
      await api.createUser({
        callsign: callsign.toUpperCase().trim(),
        firstName: firstName.trim(),
        lastName: lastName.trim(),
        email: email.trim(),
        password,
      });
      callsign = firstName = lastName = email = password = "";
      showForm = false;
      await refresh();
    } catch (err) {
      error =
        err instanceof ApiError ? err.message : "Could not create operator.";
    } finally {
      busy = false;
    }
  }
</script>

<div class="mx-auto max-w-3xl px-4 py-6">
  <a
    href="/"
    use:link
    class="mb-3 inline-block text-sm text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300"
    >← All nets</a
  >
  <div class="mb-5 flex items-center justify-between gap-3">
    <div>
      <h1 class="text-xl font-bold">Operators</h1>
      <p class="text-sm text-zinc-500 dark:text-zinc-400">
        Only admins can add operator accounts.
      </p>
    </div>
    {#if connected}
      <Button variant="primary" onclick={() => (showForm = !showForm)}>
        {showForm ? "Cancel" : "Add operator"}
      </Button>
    {/if}
  </div>

  {#if !connected}
    <div class="nl-card p-8 text-center">
      <p class="text-zinc-600 dark:text-zinc-300">
        Account management is offline.
      </p>
      <p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
        Operators and credentials are only available with a live connection to
        the server.
      </p>
    </div>
  {/if}

  {#if connected && showForm}
    <form
      onsubmit={create}
      class="nl-card mb-5 grid grid-cols-1 gap-3 p-4 sm:grid-cols-2"
    >
      <div class="sm:col-span-2">
        <label class="nl-label" for="a-call">Callsign</label>
        <input
          id="a-call"
          bind:value={callsign}
          class="nl-input font-mono uppercase"
          required
        />
      </div>
      <div>
        <label class="nl-label" for="a-first">First name</label>
        <input id="a-first" bind:value={firstName} class="nl-input" required />
      </div>
      <div>
        <label class="nl-label" for="a-last">Last name</label>
        <input id="a-last" bind:value={lastName} class="nl-input" required />
      </div>
      <div>
        <label class="nl-label" for="a-email">Email</label>
        <input
          id="a-email"
          type="email"
          bind:value={email}
          class="nl-input"
          required
        />
      </div>
      <div>
        <label class="nl-label" for="a-pass">Initial password</label>
        <input
          id="a-pass"
          type="password"
          bind:value={password}
          class="nl-input"
          minlength="8"
          required
        />
      </div>
      <div class="sm:col-span-2">
        <Button type="submit" variant="green" disabled={busy}
          >{busy ? "Creating…" : "Create operator"}</Button
        >
      </div>
    </form>
  {/if}

  {#if error}
    <p class="mb-3 text-sm text-accent-600 dark:text-accent-500">{error}</p>
  {/if}

  {#if connected}
    <div class="nl-card divide-y divide-zinc-200 dark:divide-zinc-800">
      {#each users as u (u.id)}
        <div class="flex items-center gap-3 px-4 py-3">
          <span class="nl-call">{u.callsign}</span>
          <span class="text-sm text-zinc-600 dark:text-zinc-300"
            >{u.firstName} {u.lastName}</span
          >
          {#if u.role === "admin"}
            <span class="nl-tag nl-tag-blue">Admin</span>
          {/if}
          <span class="ml-auto text-xs text-zinc-500 dark:text-zinc-400"
            >{u.email}</span
          >
        </div>
      {/each}
    </div>
  {/if}
</div>
