<script lang="ts">
  import { auth } from "../lib/stores/auth.svelte";
  import { link } from "../lib/router.svelte";
  import { ApiError } from "../lib/api";
  import Button from "../lib/components/Button.svelte";
  import TimezoneSelect from "../lib/components/TimezoneSelect.svelte";

  // Profile form, seeded from the current user.
  const u = auth.user!;
  let callsign = $state(u.callsign);
  let firstName = $state(u.firstName);
  let lastName = $state(u.lastName);
  let email = $state(u.email);
  let timezone = $state(u.timezone || "");
  let timeFormat = $state<"24h" | "12h">(u.timeFormat);
  let profileError = $state("");
  let profileSaved = $state(false);
  let savingProfile = $state(false);

  async function saveProfile(e: SubmitEvent) {
    e.preventDefault();
    savingProfile = true;
    profileError = "";
    profileSaved = false;
    try {
      await auth.updateProfile({
        callsign: callsign.toUpperCase().trim(),
        firstName: firstName.trim(),
        lastName: lastName.trim(),
        email: email.trim(),
        timezone,
        timeFormat,
      });
      profileSaved = true;
    } catch (err) {
      profileError =
        err instanceof ApiError ? err.message : "Could not save profile.";
    } finally {
      savingProfile = false;
    }
  }

  // Password form.
  let currentPassword = $state("");
  let newPassword = $state("");
  let confirmPassword = $state("");
  let pwError = $state("");
  let pwSaved = $state(false);
  let savingPw = $state(false);

  async function savePassword(e: SubmitEvent) {
    e.preventDefault();
    pwError = "";
    pwSaved = false;
    if (newPassword !== confirmPassword) {
      pwError = "New passwords don't match.";
      return;
    }
    savingPw = true;
    try {
      await auth.changePassword(currentPassword, newPassword);
      pwSaved = true;
      currentPassword = newPassword = confirmPassword = "";
    } catch (err) {
      pwError =
        err instanceof ApiError ? err.message : "Could not change password.";
    } finally {
      savingPw = false;
    }
  }
</script>

<div class="mx-auto max-w-2xl px-4 py-6">
  <a
    href="/"
    use:link
    class="mb-3 inline-block text-sm text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300"
    >← All nets</a
  >
  <h1 class="mb-5 text-xl font-bold">Settings</h1>

  <!-- Profile -->
  <form onsubmit={saveProfile} class="nl-card mb-4 p-4">
    <h2
      class="mb-3 text-sm font-semibold uppercase tracking-wide text-zinc-500"
    >
      Profile
    </h2>
    <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
      <div class="sm:col-span-2">
        <label class="nl-label" for="s-call">Callsign</label>
        <input
          id="s-call"
          bind:value={callsign}
          class="nl-input font-mono uppercase"
          required
        />
      </div>
      <div>
        <label class="nl-label" for="s-first">First name</label>
        <input id="s-first" bind:value={firstName} class="nl-input" required />
      </div>
      <div>
        <label class="nl-label" for="s-last">Last name</label>
        <input id="s-last" bind:value={lastName} class="nl-input" required />
      </div>
      <div class="sm:col-span-2">
        <label class="nl-label" for="s-email">Email</label>
        <input
          id="s-email"
          type="email"
          bind:value={email}
          class="nl-input"
          required
        />
      </div>
      <div>
        <label class="nl-label" for="s-tz">Timezone</label>
        <TimezoneSelect id="s-tz" bind:value={timezone} />
      </div>
      <div>
        <label class="nl-label" for="s-tf">Clock</label>
        <select id="s-tf" bind:value={timeFormat} class="nl-input">
          <option value="24h">24-hour</option>
          <option value="12h">12-hour</option>
        </select>
      </div>
    </div>

    {#if profileError}
      <p class="mt-3 text-sm text-accent-600 dark:text-accent-500">
        {profileError}
      </p>
    {/if}
    <div class="mt-4 flex items-center gap-3">
      <Button type="submit" variant="green" disabled={savingProfile}>
        {savingProfile ? "Saving…" : "Save profile"}
      </Button>
      {#if profileSaved}<span
          class="text-sm text-emerald-600 dark:text-emerald-500">Saved.</span
        >{/if}
    </div>
  </form>

  <!-- Password -->
  <form onsubmit={savePassword} class="nl-card mb-4 p-4">
    <h2
      class="mb-3 text-sm font-semibold uppercase tracking-wide text-zinc-500"
    >
      Change password
    </h2>
    <div class="grid grid-cols-1 gap-3">
      <div>
        <label class="nl-label" for="s-cur">Current password</label>
        <input
          id="s-cur"
          type="password"
          bind:value={currentPassword}
          class="nl-input"
          autocomplete="current-password"
          required
        />
      </div>
      <div>
        <label class="nl-label" for="s-new">New password</label>
        <input
          id="s-new"
          type="password"
          bind:value={newPassword}
          class="nl-input"
          autocomplete="new-password"
          minlength="8"
          required
        />
      </div>
      <div>
        <label class="nl-label" for="s-conf">Confirm new password</label>
        <input
          id="s-conf"
          type="password"
          bind:value={confirmPassword}
          class="nl-input"
          autocomplete="new-password"
          minlength="8"
          required
        />
      </div>
    </div>
    {#if pwError}
      <p class="mt-3 text-sm text-accent-600 dark:text-accent-500">{pwError}</p>
    {/if}
    <div class="mt-4 flex items-center gap-3">
      <Button type="submit" variant="green" disabled={savingPw}>
        {savingPw ? "Saving…" : "Change password"}
      </Button>
      {#if pwSaved}<span class="text-sm text-emerald-600 dark:text-emerald-500"
          >Password changed.</span
        >{/if}
    </div>
  </form>

  <!-- Single sign-on -->
  {#if auth.bootstrap?.oidcEnabled}
    <section class="nl-card p-4">
      <h2
        class="mb-1 text-sm font-semibold uppercase tracking-wide text-zinc-500"
      >
        Single sign-on
      </h2>
      <p class="mb-3 text-sm text-zinc-500 dark:text-zinc-400">
        Link your identity provider to this account for one-click sign-in.
      </p>
      <a href="/api/auth/oidc/start" class="nl-btn nl-btn-blue"
        >Link single sign-on</a
      >
    </section>
  {/if}
</div>
