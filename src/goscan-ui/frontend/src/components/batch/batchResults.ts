export type BatchResultRow = {
  id: string;
  findingId: number;
  domain: string;
  scriptId: string;
  scriptLabel: string;
  status: string;
  summary: string;
  exitCode: number;
  checkIndex: number;
};

export function batchProgressToRow(p: {
  findingId: number;
  domain: string;
  scriptId: string;
  scriptLabel: string;
  status: string;
  summary: string;
  exitCode: number;
  checkIndex: number;
}): BatchResultRow {
  return {
    id: `${p.findingId}-${p.scriptId}-${p.checkIndex}`,
    findingId: p.findingId,
    domain: p.domain,
    scriptId: p.scriptId,
    scriptLabel: p.scriptLabel,
    status: p.status,
    summary: p.summary,
    exitCode: p.exitCode,
    checkIndex: p.checkIndex
  };
}

export function upsertBatchResult(rows: BatchResultRow[], row: BatchResultRow): BatchResultRow[] {
  const idx = rows.findIndex((r) => r.id === row.id);
  if (idx < 0) return [...rows, row];
  const next = [...rows];
  next[idx] = row;
  return next;
}
