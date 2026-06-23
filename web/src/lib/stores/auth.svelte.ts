// Authentication state, shared app-wide.
import {
  api,
  ApiError,
  type AccountInput,
  type Credentials,
  type ProfileInput,
} from "../api";
import { resetLocal } from "../db";
import type { Bootstrap, User } from "../types";

// The last authenticated user is cached so the app can render while the backend
// is unreachable. It carries no secret — the session id stays in an HttpOnly
// cookie; this is only the profile the UI needs to draw itself.
//
// The cache is honored for OFFLINE_GRACE_MS since the last successful online
// check. That window mirrors the backend session lifetime (auth.SessionTTL = 14
// days) so an offline operator stays signed in for as long as their session
// could plausibly still be valid, then is asked to log in again.
const USER_KEY = "netlog.user";
const OFFLINE_GRACE_MS = 14 * 24 * 60 * 60 * 1000;

interface CachedUser {
  user: User;
  savedAt: number;
}

function cacheUser(user: User): void {
  if (typeof localStorage === "undefined") return;
  const entry: CachedUser = { user, savedAt: Date.now() };
  localStorage.setItem(USER_KEY, JSON.stringify(entry));
}

function clearCachedUser(): void {
  if (typeof localStorage !== "undefined") localStorage.removeItem(USER_KEY);
}

// cachedUser returns the cached user if one was stored within the grace window,
// else null (clearing a stale entry).
function cachedUser(): User | null {
  if (typeof localStorage === "undefined") return null;
  const raw = localStorage.getItem(USER_KEY);
  if (!raw) return null;
  try {
    const entry = JSON.parse(raw) as CachedUser;
    if (
      !entry?.user ||
      typeof entry.savedAt !== "number" ||
      Date.now() - entry.savedAt > OFFLINE_GRACE_MS
    ) {
      clearCachedUser();
      return null;
    }
    return entry.user;
  } catch {
    clearCachedUser();
    return null;
  }
}

class AuthStore {
  user = $state<User | null>(null);
  bootstrap = $state<Bootstrap | null>(null);
  ready = $state(false);

  get isAdmin(): boolean {
    return this.user?.role === "admin";
  }

  // load resolves the current session and bootstrap state on startup.
  async load(): Promise<void> {
    try {
      this.bootstrap = await api.bootstrap();
    } catch {
      this.bootstrap = {
        needsFirstAdmin: false,
        oidcEnabled: false,
        version: "dev",
        commit: "unknown",
      };
    }
    try {
      this.user = await api.me();
      cacheUser(this.user);
    } catch (err) {
      // Only a definitive 401 means "not signed in" — we reached the backend and
      // it rejected the session. Everything else (a network error while offline,
      // or a 502/503/504 from a reverse proxy whose backend is down, or any other
      // 5xx/timeout) means we simply couldn't confirm the session, so we keep the
      // operator signed in from cache. Logging them out on a 502 would defeat the
      // entire point of an offline-first app.
      if (err instanceof ApiError && err.status === 401) {
        this.user = null;
        clearCachedUser();
      } else {
        this.user = cachedUser();
      }
    }
    this.ready = true;
  }

  async login(creds: Credentials): Promise<void> {
    this.user = await api.login(creds);
    cacheUser(this.user);
  }

  async register(input: AccountInput): Promise<void> {
    this.user = await api.register(input);
    cacheUser(this.user);
    if (this.bootstrap) this.bootstrap.needsFirstAdmin = false;
  }

  async updateProfile(input: ProfileInput): Promise<void> {
    this.user = await api.updateProfile(input);
    cacheUser(this.user);
  }

  async changePassword(current: string, next: string): Promise<void> {
    await api.changePassword(current, next);
  }

  async logout(): Promise<void> {
    try {
      await api.logout();
    } finally {
      this.user = null;
      clearCachedUser();
      await resetLocal();
    }
  }
}

export const auth = new AuthStore();
