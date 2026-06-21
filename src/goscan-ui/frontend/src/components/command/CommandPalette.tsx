import { useEffect, useRef } from "react";
import type { FindingDTO } from "@/lib/api";

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
  onTestAllFiltered
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
        className="w-full max-w-lg border border-vscode-border bg-vscode-sidebar shadow-none"
        onClick={(e) => e.stopPropagation()}
      >
        <input
          ref={inputRef}
          className="w-full border-0 border-b border-vscode-border bg-vscode-input px-3 py-2 text-vscode-fg outline-none"
          placeholder="Type to search findings or command…"
          value={query}
          onChange={(e) => onQueryChange(e.target.value)}
        />
        <ul className="max-h-64 overflow-auto py-1">
          <li>
            <button type="button" className="w-full px-3 py-1.5 text-left text-[12px] hover:bg-vscode-hover" onClick={() => { onRunScript?.(); onClose(); }}>
              Run script
            </button>
          </li>
          <li>
            <button type="button" className="w-full px-3 py-1.5 text-left text-[12px] hover:bg-vscode-hover" onClick={() => { onStartScan?.(); onClose(); }}>
              Start scan
            </button>
          </li>
          <li>
            <button type="button" className="w-full px-3 py-1.5 text-left text-[12px] hover:bg-vscode-hover" onClick={() => { onTestAllFinding?.(); onClose(); }}>
              Test all checkers — finding actual
            </button>
          </li>
          <li>
            <button type="button" className="w-full px-3 py-1.5 text-left text-[12px] hover:bg-vscode-hover" onClick={() => { onTestAllFiltered?.(); onClose(); }}>
              Test all checkers — filtro actual
            </button>
          </li>
          {filtered.map((f) => (
            <li key={f.id}>
              <button
                type="button"
                className="w-full px-3 py-1.5 text-left hover:bg-vscode-hover"
                onClick={() => {
                  onSelectFinding(f.id);
                  onClose();
                }}
              >
                <span className="block truncate text-vscode-fg">{f.domain}</span>
                <span className="block truncate text-[11px] text-vscode-muted">{f.path}</span>
              </button>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
