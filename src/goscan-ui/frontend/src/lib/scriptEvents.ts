export type ScriptEventPayload = {
  findingId?: number;
  chunk?: string;
  scriptId?: string;
  label?: string;
  python?: string;
  exitCode?: number;
  payload?: unknown;
};

export function unwrapEventData(ev: unknown): unknown {
  return (ev as { data?: unknown }).data ?? ev;
}

export function parseScriptEvent(raw: unknown): ScriptEventPayload | null {
  if (raw === null || raw === undefined) return null;
  if (typeof raw === "string") return { chunk: raw };
  if (typeof raw === "object") return raw as ScriptEventPayload;
  return { chunk: String(raw) };
}

export function sameFindingId(eventId: unknown, findingId: number): boolean {
  if (eventId === undefined || eventId === null) return false;
  return Number(eventId) === findingId;
}

export function eventForFinding(raw: unknown, findingId: number): ScriptEventPayload | null {
  const p = parseScriptEvent(raw);
  if (!p || !sameFindingId(p.findingId, findingId)) return null;
  return p;
}

export function eventChunk(p: ScriptEventPayload): string {
  if (p.chunk !== undefined) return p.chunk;
  if (p.payload !== undefined) return String(p.payload);
  return "";
}
