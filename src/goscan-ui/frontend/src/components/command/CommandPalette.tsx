import { useEffect, useRef } from "react";
import type { FindingDTO } from "@/lib/api";
import { WORKBENCH_VIEWS, type WorkbenchView } from "@/lib/workbenchView";

type Props = {
  open: boolean;
  onClose: () => void;
  query: string;
  onQueryChange: (q: string) => void;
  findings: FindingDTO[];
  onSelectFinding: (id: number) => void;
  onRunScript?: () => void;
  onStartScan?: () => void;
  onTestAllFinding?: () => void;
  onTestAllFiltered?: () => void;
  onViewChange?: (view: WorkbenchView) => void;
};

export function CommandPalette({
  open,
  onClose,
  query,
  onQueryChange,
  findings,
  onSelectFinding,
  onRunScript,
  onStartScan,
  onTestAllFinding,
  onTestAllFiltered,
  onViewChange
}: Props) {
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (open) {
      inputRef.current?.focus();
    }
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  if (!open) return null;

  const filtered = findings.slice(0, 12);

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center bg-black/50 pt-[15vh]" onClick={onClose}>
      <div
        className="w-full max-w-lg border border-gs-border bg-gs-surface shadow-none"
        onClick={(e) => e.stopPropagation()}
      >
        <input
          ref={inputRef}
          className="w-full border-0 border-b border-gs-border bg-gs-surface-raised px-3 py-2 text-gs-fg outline-none"
          placeholder="Pesquisar findings ou comando…"
          value={query}
          onChange={(e) => onQueryChange(e.target.value)}
        />
        <ul className="max-h-64 overflow-auto py-1">
          <li className="px-3 py-1 text-[10px] uppercase tracking-wide text-gs-muted">Views</li>
          {WORKBENCH_VIEWS.map((v) => (
            <li key={v.id}>
              <button
                type="button"
                className="w-full px-3 py-1.5 text-left text-[12px] hover:bg-gs-hover"
                onClick={() => {
                  onViewChange?.(v.id);
                  onClose();
                }}
              >
                View: {v.label}
                {v.shortcut ? <span className="ml-2 text-gs-muted">Alt+{v.shortcut}</span> : null}
              </button>
            </li>
          ))}
          <li className="mt-1 px-3 py-1 text-[10px] uppercase tracking-wide text-gs-muted">Acções</li>
          <li>
            <button type="button" className="w-full px-3 py-1.5 text-left text-[12px] hover:bg-gs-hover" onClick={() => { onRunScript?.(); onClose(); }}>
              Run script
            </button>
          </li>
          <li>
            <button type="button" className="w-full px-3 py-1.5 text-left text-[12px] hover:bg-gs-hover" onClick={() => { onStartScan?.(); onClose(); }}>
              Start scan
            </button>
          </li>
          <li>
            <button type="button" className="w-full px-3 py-1.5 text-left text-[12px] hover:bg-gs-hover" onClick={() => { onTestAllFinding?.(); onClose(); }}>
              Test all checkers — finding actual
            </button>
          </li>
          <li>
            <button type="button" className="w-full px-3 py-1.5 text-left text-[12px] hover:bg-gs-hover" onClick={() => { onTestAllFiltered?.(); onClose(); }}>
              Test all checkers — filtro actual
            </button>
          </li>
          {filtered.length > 0 && (
            <li className="mt-1 px-3 py-1 text-[10px] uppercase tracking-wide text-gs-muted">Findings</li>
          )}
          {filtered.map((f) => (
            <li key={f.id}>
              <button
                type="button"
                className="w-full px-3 py-1.5 text-left hover:bg-gs-hover"
                onClick={() => {
                  onSelectFinding(f.id);
                  onClose();
                }}
              >
                <span className="block truncate text-gs-fg">{f.domain}</span>
                <span className="block truncate text-[11px] text-gs-muted">{f.path}</span>
              </button>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
