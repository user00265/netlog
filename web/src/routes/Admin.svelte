<script lang="ts">
  import { api, ApiError } from "../lib/api";
  import { link } from "../lib/router.svelte";
  import { syncState } from "../lib/sync.svelte";
  import type { Role, User } from "../lib/types";
  import Button from "../lib/components/Button.svelte";
  import Pencil from "@lucide/svelte/icons/pencil";

  // Accounts are managed live only — there is no offline user list or stored
  // credentials, so listing/editing needs a reachable backend. Per the offline
  // UX, action controls stay visible but disabled when offline.
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

  // Per-user edit (identity + role).
  let editUser = $state<User | null>(null);
  let eCall = $state("");
  let eFirst = $state("");
  let eLast = $state("");
  let eEmail = $state("");
  let eRole = $state<Role>("user");
  let editErr = $state("");
  let editBusy = $state(false);

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

  function startEdit(u: User) {
    editUser = u;
    eCall = u.callsign;
    eFirst = u.firstName;
    eLast = u.lastName;
    eEmail = u.email;
    eRole = u.role;
    editErr = "";
  }

  async function saveEdit(e: SubmitEvent) {
    e.preventDefault();
    if (!editUser) return;
    editBusy = true;
    editErr = "";
    try {
      await api.updateUser(editUser.id, {
        callsign: eCall.toUpperCase().trim(),
        firstName: eFirst.trim(),
        lastName: eLast.trim(),
        email: eEmail.trim(),
        role: eRole,
      });
      editUser = null;
      await refresh();
    } catch (err) {
      editErr =
        err instanceof ApiError ? err.message : "Could not save changes.";
    } finally {
      editBusy = false;
    }
  }
</script>

<svelte:window
  onkeydown={(e) => editUser && e.key === "Escape" && (editUser = null)}
/>

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
        Only admins can add and edit operator accounts.
      </p>
    </div>
    <Button
      variant="primary"
      disabled={!connected}
      onclick={() => (showForm = !showForm)}
    >
      {showForm ? "Cancel" : "Add operator"}
    </Button>
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
          <span
            class="ml-auto hidden text-xs text-zinc-500 sm:inline dark:text-zinc-400"
            >{u.email}</span
          >
          <button
            class="nl-icon-btn ml-auto sm:ml-0"
            title="Edit operator"
            aria-label={`Edit ${u.callsign}`}
            onclick={() => startEdit(u)}
          >
            <Pencil class="h-4 w-4" />
          </button>
        </div>
      {/each}
    </div>
  {/if}
</div>

{#if editUser}
  <div class="fixed inset-0 z-30 flex items-center justify-center p-4">
    <button
      class="absolute inset-0 bg-black/50"
      aria-label="Close"
      onclick={() => (editUser = null)}
    ></button>
    <div
      class="nl-card relative z-10 w-full max-w-md p-5"
      role="dialog"
      aria-modal="true"
      aria-labelledby="edit-title"
    >
      <h2 id="edit-title" class="mb-3 text-lg font-semibold">Edit operator</h2>
      <form onsubmit={saveEdit} class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div class="sm:col-span-2">
          <label class="nl-label" for="e-call">Callsign</label>
          <input
            id="e-call"
            bind:value={eCall}
            class="nl-input font-mono uppercase"
            required
          />
        </div>
        <div>
          <label class="nl-label" for="e-first">First name</label>
          <input id="e-first" bind:value={eFirst} class="nl-input" required />
        </div>
        <div>
          <label class="nl-label" for="e-last">Last name</label>
          <input id="e-last" bind:value={eLast} class="nl-input" required />
        </div>
        <div>
          <label class="nl-label" for="e-email">Email</label>
          <input
            id="e-email"
            type="email"
            bind:value={eEmail}
            class="nl-input"
            required
          />
        </div>
        <div>
          <label class="nl-label" for="e-role">Role</label>
          <select id="e-role" bind:value={eRole} class="nl-input">
            <option value="user">User</option>
            <option value="admin">Admin</option>
          </select>
        </div>
        {#if editErr}
          <p class="text-sm text-accent-600 sm:col-span-2 dark:text-accent-500">
            {editErr}
          </p>
        {/if}
        <div class="flex justify-end gap-2 sm:col-span-2">
          <Button variant="gray" onclick={() => (editUser = null)}
            >Cancel</Button
          >
          <Button type="submit" variant="green" disabled={editBusy}>
            {editBusy ? "Saving…" : "Save"}
          </Button>
        </div>
      </form>
    </div>
  </div>
{/if}
