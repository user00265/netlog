<script lang="ts">
  import { auth } from "../lib/stores/auth.svelte";
  import { navigate } from "../lib/router.svelte";
  import { ApiError } from "../lib/api";
  import Button from "../lib/components/Button.svelte";
  import ThemeToggle from "../lib/components/ThemeToggle.svelte";
  import TimezoneSelect from "../lib/components/TimezoneSelect.svelte";
  import { browserTimezone } from "../lib/datetime";
  import { syncState } from "../lib/sync.svelte";

  // Account creation is live-only — credentials are never stored offline.
  const online = $derived(syncState.online);

  let callsign = $state("");
  let firstName = $state("");
  let lastName = $state("");
  let email = $state("");
  let password = $state("");
  let timezone = $state(browserTimezone());
  let timeFormat = $state<"24h" | "12h">("24h");
  let error = $state("");
  let busy = $state(false);

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    busy = true;
    error = "";
    try {
      await auth.register({
        callsign: callsign.toUpperCase().trim(),
        firstName: firstName.trim(),
        lastName: lastName.trim(),
        email: email.trim(),
        password,
        timezone,
        timeFormat,
      });
      navigate("/");
    } catch (err) {
      error =
        err instanceof ApiError ? err.message : "Could not create the account.";
    } finally {
      busy = false;
    }
  }
</script>

<div class="flex min-h-dvh items-center justify-center p-4">
  <div class="absolute right-4 top-4"><ThemeToggle /></div>
  <div class="w-full max-w-md">
    <div class="mb-6 flex items-center justify-center gap-2">
      <span class="flex items-center gap-[3px]" aria-hidden="true">
        <span class="h-2.5 w-2.5 rounded-full bg-accent-500"></span>
        <span class="h-2.5 w-6 rounded-full bg-accent-500"></span>
      </span>
      <span class="text-2xl font-bold tracking-tight">NetLog</span>
    </div>

    <form onsubmit={submit} class="nl-card p-6">
      <h1 class="text-lg font-semibold">Create the admin account</h1>
      <p class="mb-4 text-sm text-zinc-500 dark:text-zinc-400">
        This is the first account, so it becomes the administrator. After this,
        only the admin can add operators.
      </p>

      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div class="sm:col-span-2">
          <label class="nl-label" for="r-call">Callsign</label>
          <input
            id="r-call"
            bind:value={callsign}
            class="nl-input font-mono uppercase tracking-wide"
            placeholder="W1AW"
            autocapitalize="characters"
            spellcheck="false"
            required
          />
        </div>
        <div>
          <label class="nl-label" for="r-first">First name</label>
          <input
            id="r-first"
            bind:value={firstName}
            class="nl-input"
            required
          />
        </div>
        <div>
          <label class="nl-label" for="r-last">Last name</label>
          <input id="r-last" bind:value={lastName} class="nl-input" required />
        </div>
        <div class="sm:col-span-2">
          <label class="nl-label" for="r-email">Email</label>
          <input
            id="r-email"
            type="email"
            bind:value={email}
            class="nl-input"
            required
          />
        </div>
        <div class="sm:col-span-2">
          <label class="nl-label" for="r-pass">Password</label>
          <input
            id="r-pass"
            type="password"
            bind:value={password}
            class="nl-input"
            autocomplete="new-password"
            minlength="8"
            required
          />
        </div>
        <div>
          <label class="nl-label" for="r-tz">Timezone</label>
          <TimezoneSelect id="r-tz" bind:value={timezone} />
        </div>
        <div>
          <label class="nl-label" for="r-tf">Clock</label>
          <select id="r-tf" bind:value={timeFormat} class="nl-input">
            <option value="24h">24-hour</option>
            <option value="12h">12-hour</option>
          </select>
        </div>
      </div>
      <p class="mt-2 text-xs text-zinc-400">
        Times display in UTC with your local time in parentheses.
      </p>

      {#if error}
        <p class="mt-3 text-sm text-accent-600 dark:text-accent-500">{error}</p>
      {/if}
      {#if !online}
        <p class="mt-3 text-sm text-amber-600 dark:text-amber-400">
          You must be online to create an account.
        </p>
      {/if}

      <Button
        type="submit"
        variant="primary"
        disabled={busy || !online}
        class="mt-4 w-full"
      >
        {busy ? "Creating…" : "Create admin account"}
      </Button>
    </form>
  </div>
</div>
