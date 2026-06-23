<script lang="ts">
  import { auth } from "../lib/stores/auth.svelte";
  import { navigate } from "../lib/router.svelte";
  import { ApiError } from "../lib/api";
  import Button from "../lib/components/Button.svelte";
  import ThemeToggle from "../lib/components/ThemeToggle.svelte";

  let callsign = $state("");
  let password = $state("");
  let error = $state("");
  let busy = $state(false);

  // Surface OIDC redirect errors.
  const params = new URLSearchParams(location.search);
  const oidcError = params.get("error");
  if (oidcError === "nouser")
    error = "No NetLog account is linked to that identity.";
  else if (oidcError === "oidc") error = "Single sign-on failed. Try again.";

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    busy = true;
    error = "";
    try {
      await auth.login({ callsign: callsign.toUpperCase().trim(), password });
      navigate("/");
    } catch (err) {
      error = err instanceof ApiError ? err.message : "Sign in failed.";
    } finally {
      busy = false;
    }
  }
</script>

<div class="flex min-h-dvh items-center justify-center p-4">
  <div class="absolute right-4 top-4"><ThemeToggle /></div>
  <div class="w-full max-w-sm">
    <div class="mb-6 flex items-center justify-center gap-2">
      <span class="flex items-center gap-[3px]" aria-hidden="true">
        <span class="h-2.5 w-2.5 rounded-full bg-accent-500"></span>
        <span class="h-2.5 w-6 rounded-full bg-accent-500"></span>
      </span>
      <span class="text-2xl font-bold tracking-tight">NetLog</span>
    </div>

    <form onsubmit={submit} class="nl-card p-6">
      <h1 class="mb-1 text-lg font-semibold">Sign in</h1>
      <p class="mb-4 text-sm text-zinc-500 dark:text-zinc-400">
        Net Control logging for directed nets.
      </p>

      <label class="nl-label" for="callsign">Callsign</label>
      <input
        id="callsign"
        bind:value={callsign}
        class="nl-input mb-3 font-mono uppercase tracking-wide"
        placeholder="W1AW"
        autocomplete="username"
        autocapitalize="characters"
        spellcheck="false"
        required
      />

      <label class="nl-label" for="password">Password</label>
      <input
        id="password"
        type="password"
        bind:value={password}
        class="nl-input mb-4"
        autocomplete="current-password"
        required
      />

      {#if error}
        <p class="mb-3 text-sm text-accent-600 dark:text-accent-500">{error}</p>
      {/if}

      <Button type="submit" variant="primary" disabled={busy} class="w-full">
        {busy ? "Signing in…" : "Sign in"}
      </Button>

      {#if auth.bootstrap?.oidcEnabled}
        <a
          href="/api/auth/oidc/start"
          class="nl-btn nl-btn-blue mt-3 w-full"
          data-sveltekit-reload>Sign in with single sign-on</a
        >
      {/if}
    </form>
  </div>
</div>
