// Thin fetch wrappers for the JSON API. All requests include credentials so the
// session cookie travels with them.
import type {
  Bootstrap,
  CallsignData,
  CheckIn,
  Net,
  NetWithMeta,
  PullResponse,
  PushResponse,
  SyncChange,
  User,
} from "./types";

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
  }
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
): Promise<T> {
  const res = await fetch(path, {
    method,
    credentials: "include",
    headers:
      body !== undefined ? { "Content-Type": "application/json" } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) {
    let message = res.statusText;
    try {
      const data = await res.json();
      if (data?.error) message = data.error;
    } catch {
      // non-JSON error body; keep status text
    }
    throw new ApiError(res.status, message);
  }
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}

export interface Credentials {
  callsign: string;
  password: string;
}

export interface AccountInput {
  callsign: string;
  firstName: string;
  lastName: string;
  email: string;
  password: string;
  timezone?: string;
  timeFormat?: "24h" | "12h";
}

export interface ProfileInput {
  callsign: string;
  firstName: string;
  lastName: string;
  email: string;
  timezone: string;
  timeFormat: "24h" | "12h";
}

export interface AdminUserInput {
  callsign: string;
  firstName: string;
  lastName: string;
  email: string;
  role: "admin" | "user";
}

export const api = {
  bootstrap: () => request<Bootstrap>("GET", "/api/bootstrap"),
  register: (input: AccountInput) =>
    request<User>("POST", "/api/register", input),
  login: (creds: Credentials) => request<User>("POST", "/api/login", creds),
  logout: () => request<void>("POST", "/api/logout"),
  me: () => request<User>("GET", "/api/me"),
  updateProfile: (input: ProfileInput) =>
    request<User>("PATCH", "/api/account/profile", input),
  changePassword: (currentPassword: string, newPassword: string) =>
    request<void>("POST", "/api/account/password", {
      currentPassword,
      newPassword,
    }),

  listUsers: () => request<User[]>("GET", "/api/admin/users"),
  createUser: (input: AccountInput) =>
    request<User>("POST", "/api/admin/users", input),
  updateUser: (id: string, input: AdminUserInput) =>
    request<User>("PATCH", `/api/admin/users/${encodeURIComponent(id)}`, input),

  listNets: () => request<NetWithMeta[]>("GET", "/api/nets"),
  getNet: (id: string) =>
    request<Net & { ncsCallsign: string; checkins: CheckIn[] }>(
      "GET",
      `/api/nets/${id}`,
    ),
  reassignNcs: (netId: string, callsign: string) =>
    request<Net>("POST", `/api/nets/${encodeURIComponent(netId)}/ncs`, {
      callsign,
    }),

  getCallsign: (call: string) =>
    request<CallsignData>("GET", `/api/callsign/${encodeURIComponent(call)}`),
  refreshCallsign: (call: string) =>
    request<CallsignData>(
      "POST",
      `/api/callsign/${encodeURIComponent(call)}/refresh`,
    ),

  pull: (since: string) =>
    request<PullResponse>(
      "GET",
      `/api/sync?since=${encodeURIComponent(since)}`,
    ),
  push: (changes: SyncChange[]) =>
    request<PushResponse>("POST", "/api/sync", { changes }),
};
