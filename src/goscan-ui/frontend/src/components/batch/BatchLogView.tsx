import { Copy, Trash2 } from "lucide-react";
import { useEffect, useRef } from "react";

type Props = {
  lines: string[];
  onClear: () => void;
  emptyHint?: string;
  embedded?: boolean;
};

export function BatchLogView({ lines, onClear, emptyHint, embedded }: Props) {
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const el = scrollRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [lines]);

  return (
    <div
      className={
        embedded
          ? "flex min-h-0 flex-1 flex-col"
          : "flex min-h-0 flex-1 flex-col rounded-lg border border-gs-border bg-gs-surface"
      }
    >
      {!embedded && (
        <div className="flex shrink-0 items-center justify-between border-b border-gs-border px-3 py-2">
          <span className="text-[11px] font-semibold uppercase tracking-wide text-gs-accent">Log</span>
          <LogActions lines={lines} onClear={onClear} />
        </div>
      )}
      {embedded && (
        <div className="flex shrink-0 justify-end px-1 pb-1">
          <LogActions lines={lines} onClear={onClear} />
        </div>
      )}
      <div
        ref={scrollRef}
        className="min-h-0 flex-1 overflow-auto p-3 font-mono text-[11px] leading-relaxed text-gs-fg"
      >
        {lines.length === 0 ? (
          <span className="text-gs-muted">
            {emptyHint ?? "Linhas do batch aparecem aqui quando iniciar uma operação."}
          </span>
        ) : (
          lines.map((line, i) => (
            <div key={i} className="whitespace-pre-wrap break-all">
              {line}
            </div>
          ))
        )}
      </div>
    </div>
  );
}

function LogActions({ lines, onClear }: { lines: string[]; onClear: () => void }) {
  return (
    <span className="flex gap-1">
      <button
        type="button"
        className="rounded p-1 text-gs-muted hover:bg-gs-hover hover:text-gs-fg"
        onClick={onClear}
        title="Limpar log"
        disabled={lines.length === 0}
      >
        <Trash2 className="h-3.5 w-3.5" />
      </button>
      <button
        type="button"
        className="rounded p-1 text-gs-muted hover:bg-gs-hover hover:text-gs-fg"
        onClick={() => void navigator.clipboard.writeText(lines.join("\n"))}
        title="Copiar log"
        disabled={lines.length === 0}
      >
        <Copy className="h-3.5 w-3.5" />
      </button>
    </span>
  );
}
