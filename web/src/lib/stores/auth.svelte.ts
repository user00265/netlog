// Authentication state, shared app-wide.
import {
  api,
  type AccountInput,
  type Credentials,
  type ProfileInput,
} from "../api";
import { resetLocal } from "../db";
import type { Bootstrap, User } from "../types";

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
    } catch {
      this.user = null;
    }
    this.ready = true;
  }

  async login(creds: Credentials): Promise<void> {
    this.user = await api.login(creds);
  }

  async register(input: AccountInput): Promise<void> {
    this.user = await api.register(input);
    if (this.bootstrap) this.bootstrap.needsFirstAdmin = false;
  }

  async updateProfile(input: ProfileInput): Promise<void> {
    this.user = await api.updateProfile(input);
  }

  async changePassword(current: string, next: string): Promise<void> {
    await api.changePassword(current, next);
  }

  async logout(): Promise<void> {
    try {
      await api.logout();
    } finally {
      this.user = null;
      await resetLocal();
    }
  }
}

export const auth = new AuthStore();
