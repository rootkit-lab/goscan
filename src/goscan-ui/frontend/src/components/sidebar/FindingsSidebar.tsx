import { Key, Search } from "lucide-react";
import type { FindingDTO } from "@/lib/api";

const CONFIDENCE_OPTIONS = [
  { value: "", label: "All" },
  { value: "HIGH", label: "HIGH" },
  { value: "MEDIUM", label: "MEDIUM" }
];

type Props = {
  query: string;
  onQueryChange: (q: string) => void;
  confidence: string;
  onConfidenceChange: (c: string) => void;
  findings: FindingDTO[];
  selectedId: number | null;
  onSelect: (id: number) => void;
};

export function FindingsSidebar({
  query,
  onQueryChange,
  confidence,
  onConfidenceChange,
  findings,
  selectedId,
  onSelect
}: Props) {
  return (
    <div className="flex h-full flex-col">
      <div className="border-b border-vscode-border px-3 py-2 text-[11px] font-semibold uppercase tracking-wide text-vscode-muted">
        Explorer
      </div>
      <div className="space-y-2 border-b border-vscode-border p-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-2 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-vscode-muted" />
          <input
            className="vscode-input pl-7"
            placeholder="Search findings (Ctrl+K)"
            value={query}
            onChange={(e) => onQueryChange(e.target.value)}
          />
        </div>
        <div className="flex gap-1">
          {CONFIDENCE_OPTIONS.map((opt) => (
            <button
              key={opt.value || "all"}
              type="button"
              onClick={() => onConfidenceChange(opt.value)}
              className={`flex-1 border px-1 py-0.5 text-[11px] ${
                confidence === opt.value
                  ? "border-vscode-accent bg-vscode-selection text-vscode-fg"
                  : "border-vscode-border bg-transparent text-vscode-muted hover:bg-vscode-hover"
              }`}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>
      <div className="min-h-0 flex-1 overflow-auto py-1">
        {findings.length === 0 && <p className="px-3 py-4 text-vscode-muted">No results</p>}
        {findings.map((f) => (
          <button
            key={f.id}
            type="button"
            onClick={() => onSelect(f.id)}
            className={`flex w-full items-start gap-2 px-2 py-1 text-left hover:bg-vscode-hover ${
              selectedId === f.id ? "bg-vscode-list-active" : ""
            }`}
          >
            {f.hasCredentials ? (
              <Key className="mt-0.5 h-3.5 w-3.5 shrink-0 text-vscode-warning" />
            ) : (
              <span className="mt-0.5 inline-block h-3.5 w-3.5 shrink-0" />
            )}
            <span className="min-w-0 flex-1">
              <span className="block truncate text-vscode-fg">{f.domain}</span>
              <span className="block truncate text-[11px] text-vscode-muted">
                {f.path} · {f.confidence}
              </span>
            </span>
          </button>
        ))}
      </div>
    </div>
  );
}
