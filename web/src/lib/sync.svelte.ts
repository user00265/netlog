// The offline-first engine. Mutations write Dexie + an outbox entry first, then a
// background flusher pushes the outbox to the backend and reconciles with the
// authoritative server records. On load/reconnect we sync down changes.
import { api, ApiError } from "./api";
import { db, getLastSync, setLastSync } from "./db";
import { newId } from "./id";
import { normalizeCallsign } from "./callsign";
import { auth } from "./stores/auth.svelte";
import type { CheckIn, Net, SyncChange } from "./types";

// syncState is reactive UI status for the footer indicators.
//   online    — the browser reports a network connection (navigator.onLine)
//   reachable — the last request to OUR backend actually completed (a server
//               can be unreachable even when the device is "online")
//   syncing   — a flush is in progress
//   pending   — number of unsynced outbox entries
//   error     — last sync error, surfaced in the sync-icon tooltip
export const syncState = $state({
  online: typeof navigator !== "undefined" ? navigator.onLine : true,
  reachable: true,
  syncing: false,
  pending: 0,
  error: "",
});

function now(): string {
  return new Date().toISOString();
}

async function refreshPending(): Promise<void> {
  syncState.pending = await db.outbox.count();
}

// syncDown pulls server changes since the last sync and merges them into Dexie,
// applying tombstones (server is authoritative).
export async function syncDown(): Promise<void> {
  if (!syncState.online) return;
  const since = await getLastSync();
  try {
    const res = await api.pull(since);

    // The API serializes empty result sets as null, so guard before iterating.
    await db.transaction("rw", db.nets, db.checkins, async () => {
      for (const n of res.nets ?? []) await mergeNet(n);
      for (const c of res.checkins ?? []) await mergeCheckin(c);
    });
    await setLastSync(res.serverTime);
    syncState.reachable = true;
  } catch (err) {
    // An HTTP response (ApiError) still means we reached the server; only a
    // thrown network error means it's unreachable.
    syncState.reachable = err instanceof ApiError;
  }
}

async function mergeNet(n: Net): Promise<void> {
  if (n.deletedAt) {
    await db.nets.delete(n.id);
    return;
  }
  // Sync push results omit the denormalized NCS callsign; keep what we have so
  // the list doesn't flicker until the next full pull repopulates it.
  if (!n.ncsCallsign) {
    const existing = await db.nets.get(n.id);
    if (existing?.ncsCallsign) n.ncsCallsign = existing.ncsCallsign;
  }
  await db.nets.put(n);
}

async function mergeCheckin(c: CheckIn): Promise<void> {
  if (c.deletedAt) await db.checkins.delete(c.id);
  else await db.checkins.put(c);
}

// flushOutbox sends pending changes to the backend and reconciles results.
export async function flushOutbox(): Promise<void> {
  if (!syncState.online || syncState.syncing) return;
  const entries = await db.outbox.orderBy("createdAt").toArray();
  if (entries.length === 0) return;

  syncState.syncing = true;
  syncState.error = "";
  try {
    const changes: SyncChange[] = entries.map((e) => ({
      entity: e.entity,
      op: e.op,
      id: e.entityId,
      data: e.payload,
    }));
    const res = await api.push(changes);
    syncState.reachable = true; // the push completed, so the server is reachable

    let sawConflict = false;
    let conflictMessage = "";
    const results = res.results ?? [];
    await db.transaction("rw", db.nets, db.checkins, db.outbox, async () => {
      for (let i = 0; i < results.length; i++) {
        const result = results[i];
        const entry = entries[i];
        if (entry.localId === undefined) continue;

        if (result.status === "applied") {
          if (result.net) await mergeNet(result.net);
          if (result.checkin) await mergeCheckin(result.checkin);
          await db.outbox.delete(entry.localId);
        } else if (result.status === "conflict") {
          // The server permanently rejected this change; drop it and re-pull the
          // authoritative state to undo our optimistic local edit.
          sawConflict = true;
          if (!conflictMessage && result.message)
            conflictMessage = result.message;
          await db.outbox.delete(entry.localId);
        } else {
          // Transient server error: keep the entry so the next flush retries it
          // rather than silently losing the change.
          await db.outbox.update(entry.localId, {
            attempts: (entry.attempts ?? 0) + 1,
          });
        }
      }
    });

    // A rejected change means our optimistic local state is wrong; pull the
    // authoritative state to correct it.
    if (sawConflict) {
      syncState.error = conflictMessage
        ? `Change rejected: ${conflictMessage}`
        : "Some changes were rejected by the server.";
      await syncDown();
    }
  } catch (err) {
    if (err instanceof ApiError) {
      syncState.reachable = true; // server responded, just with an error
      syncState.error = err.message;
    } else {
      syncState.reachable = false; // network failure — couldn't reach the server
      syncState.error = "Can't reach the server; will retry.";
    }
  } finally {
    syncState.syncing = false;
    await refreshPending();
  }
}

async function enqueue(
  entity: "net" | "checkin",
  op: "put" | "delete",
  entityId: string,
  payload: Record<string, unknown>,
): Promise<void> {
  await db.outbox.add({
    entity,
    op,
    entityId,
    payload,
    createdAt: now(),
    attempts: 0,
  });
  await refreshPending();
  void flushOutbox();
}

// ---- Mutations (optimistic local write + outbox) --------------------------

export async function createNet(
  name: string,
  netDate: string,
): Promise<string> {
  const id = newId();
  const ts = now();
  const net: Net = {
    id,
    name,
    netDate,
    // Unassigned until someone opens it (claims NCS). Anyone may open/clean an
    // unassigned net, so it's manageable on create.
    ncsUserId: "",
    status: "pending",
    startAt: null,
    endAt: null,
    notes: "",
    createdAt: ts,
    updatedAt: ts,
    ncsCallsign: "",
    canManage: true,
  };
  await db.nets.put(net);
  await enqueue("net", "put", id, { name, netDate });
  return id;
}

// deleteNet soft-deletes a net (admins and any controller). Offline-capable: the
// local row is removed optimistically and the tombstone syncs via the outbox.
export async function deleteNet(net: Net): Promise<void> {
  await db.nets.delete(net.id);
  await enqueue("net", "delete", net.id, {});
}

// reassignNcs hands a net's NCS role to another operator by callsign. Online-only
// (the server resolves the callsign to an account), so callers must gate it on
// connectivity. The returned net carries the new NCS + controller access.
export async function reassignNcs(
  netId: string,
  callsign: string,
): Promise<void> {
  const net = await api.reassignNcs(netId, normalizeCallsign(callsign));
  await mergeNet(net);
}

// setNetNotes saves the NCS notes for a net (auto-save path).
export async function setNetNotes(net: Net, notes: string): Promise<void> {
  await db.nets.update(net.id, { notes, updatedAt: now() });
  await enqueue("net", "put", net.id, { notes });
}

export async function setNetStatus(
  net: Net,
  status: "open" | "closed",
): Promise<void> {
  const patch: Partial<Net> = { status, updatedAt: now() };
  if (status === "open" && !net.startAt) patch.startAt = now();
  if (status === "closed" && !net.endAt) patch.endAt = now();
  // Opening an unassigned net claims NCS for the current operator. Apply it
  // optimistically (the server does the same) so the NCS shows immediately.
  if (status === "open" && !net.ncsUserId && auth.user) {
    patch.ncsUserId = auth.user.id;
    patch.ncsCallsign = auth.user.callsign;
    patch.canManage = true;
  }
  await db.nets.update(net.id, patch);
  await enqueue("net", "put", net.id, { status });
}

export interface CheckInInput {
  callsign: string;
  nickname: string;
  hasTraffic: boolean;
  shortTime: boolean;
  notes?: string;
}

export async function addCheckIn(
  netId: string,
  input: CheckInInput,
): Promise<void> {
  const id = newId();
  const ts = now();
  const maxSeq = await db.checkins.where("netId").equals(netId).count();
  const checkin: CheckIn = {
    id,
    netId,
    callsign: normalizeCallsign(input.callsign),
    nickname: input.nickname,
    hasTraffic: input.hasTraffic,
    shortTime: input.shortTime,
    notes: input.notes ?? "",
    seq: maxSeq + 1,
    checkedInAt: ts,
    createdAt: ts,
    updatedAt: ts,
  };
  await db.checkins.put(checkin);
  await enqueue("checkin", "put", id, {
    netId,
    callsign: checkin.callsign,
    nickname: checkin.nickname,
    hasTraffic: checkin.hasTraffic,
    shortTime: checkin.shortTime,
    notes: checkin.notes,
  });
  void loadCallsign(checkin.callsign, true);
}

export async function editCheckIn(checkin: CheckIn): Promise<void> {
  const updated = {
    ...checkin,
    callsign: normalizeCallsign(checkin.callsign),
    updatedAt: now(),
  };
  await db.checkins.put(updated);
  await enqueue("checkin", "put", checkin.id, {
    netId: updated.netId,
    callsign: updated.callsign,
    nickname: updated.nickname,
    hasTraffic: updated.hasTraffic,
    shortTime: updated.shortTime,
    notes: updated.notes,
  });
}

export async function removeCheckIn(checkin: CheckIn): Promise<void> {
  await db.checkins.delete(checkin.id);
  await enqueue("checkin", "delete", checkin.id, {});
}

// ---- Callsign cache -------------------------------------------------------

// attemptedCallsigns is a session-scoped guard so a callsign the server can't
// resolve isn't re-fetched on every reactive update. Without it, a reactive
// caller (e.g. the net view's effect) re-issues a request for each unresolved
// callsign whenever any other callsign is enriched — a request storm.
const attemptedCallsigns = new Set<string>();

// loadCallsign fetches cached callbook/DXCC data and stores it locally. Because
// server-side enrichment runs asynchronously after a check-in, it retries once.
export async function loadCallsign(call: string, retry = false): Promise<void> {
  if (!syncState.online || attemptedCallsigns.has(call)) return;
  attemptedCallsigns.add(call);
  try {
    const data = await api.getCallsign(call);
    await db.callsigns.put(data);
  } catch (err) {
    if (retry && err instanceof ApiError && err.status === 404) {
      // Enrichment may not be ready yet; allow exactly one delayed retry.
      attemptedCallsigns.delete(call);
      setTimeout(() => void loadCallsign(call, false), 1800);
    }
    // Otherwise leave it marked attempted (negative cache) — the manual refresh
    // button still forces a fresh lookup via refreshCallsign.
  }
}

export async function refreshCallsign(call: string): Promise<void> {
  const data = await api.refreshCallsign(call);
  await db.callsigns.put(data);
  attemptedCallsigns.add(call);
}

// reloadCallsign forces a fresh fetch (used when the backend signals via SSE that
// a callsign's enrichment just completed), bypassing the negative cache.
async function reloadCallsign(call: string): Promise<void> {
  attemptedCallsigns.delete(call);
  await loadCallsign(call);
}

// ---- Server-Sent Events ---------------------------------------------------
// The SSE stream doubles as the backend-reachability signal: while it's open the
// server is reachable; the server pushes "sync" when data changes and "callsign"
// when enrichment finishes.

let eventSource: EventSource | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;

// The browser only auto-reconnects an EventSource when an *established* stream
// drops mid-flight. If the connection attempt itself gets an HTTP error — e.g. a
// 502 from the reverse proxy while the backend is down/restarting — the
// EventSource goes to CLOSED for good and never retries, leaving the app stuck
// "offline" even after the backend returns. So we reconnect on a timer.
function scheduleReconnect(delay = 3000): void {
  if (reconnectTimer || !syncState.online) return;
  reconnectTimer = setTimeout(() => {
    reconnectTimer = null;
    startEvents();
  }, delay);
}

function startEvents(): void {
  if (eventSource || typeof EventSource === "undefined") return;
  const es = new EventSource("/api/events");
  eventSource = es;
  es.onopen = () => {
    syncState.reachable = true;
    // Catch up on anything that changed while we were disconnected.
    void flushOutbox();
    void syncDown();
  };
  es.onerror = () => {
    syncState.reachable = false;
    // CONNECTING means the browser is already retrying a dropped stream; CLOSED
    // means it gave up (an HTTP-error response, e.g. a 502 while the backend was
    // down), so we must tear it down and reconnect ourselves.
    if (es.readyState === EventSource.CLOSED) {
      stopEvents();
      scheduleReconnect();
    }
  };
  es.addEventListener("sync", () => void syncDown());
  es.addEventListener(
    "callsign",
    (e) => void reloadCallsign((e as MessageEvent).data),
  );
}

function stopEvents(): void {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  if (eventSource) {
    eventSource.close();
    eventSource = null;
  }
}

// ---- Lifecycle ------------------------------------------------------------

let listenersBound = false;
let flushTimer: ReturnType<typeof setInterval> | null = null;

// startSync begins syncing for an authenticated session: it binds the network
// listeners once, opens the SSE stream, starts the periodic flush, and does an
// initial catch-up. Call stopSync on logout.
export async function startSync(): Promise<void> {
  attemptedCallsigns.clear();
  await refreshPending();

  if (!listenersBound && typeof window !== "undefined") {
    listenersBound = true;
    window.addEventListener("online", () => {
      syncState.online = true;
      startEvents(); // revive the SSE stream if it was torn down while offline
      void flushOutbox();
      void syncDown();
    });
    window.addEventListener("offline", () => {
      syncState.online = false;
      syncState.reachable = false;
    });
    // The service worker pings us when Background Sync fires.
    navigator.serviceWorker?.addEventListener?.("message", (e) => {
      if (e.data?.type === "FLUSH_OUTBOX") void flushOutbox();
    });
  }

  startEvents();
  if (!flushTimer) flushTimer = setInterval(() => void flushOutbox(), 30_000);

  await flushOutbox();
  await syncDown();
}

// stopSync tears down the session's sync activity (on logout). Window listeners
// are left bound (cheap, idempotent) and re-used on the next login.
export function stopSync(): void {
  stopEvents();
  if (flushTimer) {
    clearInterval(flushTimer);
    flushTimer = null;
  }
}
