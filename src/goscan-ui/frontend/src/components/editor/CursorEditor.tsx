import Editor from "@monaco-editor/react";
import { FileCode2 } from "lucide-react";
import type { FindingDetailDTO } from "@/lib/api";

type Props = {
  detail: FindingDetailDTO;
};

export function CursorEditor({ detail }: Props) {
  const tabLabel = `${detail.domain}${detail.path}`;

  return (
    <div className="flex h-full flex-col bg-gs-bg">
      <div className="flex h-[35px] shrink-0 items-end border-b border-gs-border bg-gs-tab-inactive">
        <div className="group flex h-full max-w-[min(100%,520px)] items-center gap-2 border-r border-gs-border bg-gs-tab-active px-3 text-gs-fg">
          <FileCode2 className="h-3.5 w-3.5 shrink-0 text-gs-accent" />
          <span className="truncate text-[13px]">{tabLabel}</span>
        </div>
      </div>
      <div className="flex h-[22px] shrink-0 items-center gap-1.5 border-b border-gs-border bg-gs-bg px-3 text-[11px] text-gs-muted">
        <span className="truncate">{detail.domain}</span>
        <span className="text-gs-dim">›</span>
        <span className="truncate">{detail.path.replace(/^\//, "")}</span>
        {detail.confidence && (
          <>
            <span className="text-gs-dim">·</span>
            <span className="rounded bg-gs-surface-raised px-1.5 py-px text-[10px] uppercase tracking-wide text-gs-fg">
              {detail.confidence}
            </span>
          </>
        )}
      </div>
      <div className="min-h-0 flex-1">
        <Editor
          height="100%"
          defaultLanguage="plaintext"
          theme="vs-dark"
          value={detail.content}
          options={{
            minimap: { enabled: true, scale: 1, maxColumn: 80 },
            fontSize: 13,
            lineHeight: 20,
            fontFamily: "var(--gs-font-mono)",
            readOnly: true,
            scrollBeyondLastLine: false,
            padding: { top: 12, bottom: 12 },
            lineNumbers: "on",
            renderLineHighlight: "all",
            smoothScrolling: true,
            cursorBlinking: "smooth",
            bracketPairColorization: { enabled: true },
            guides: { indentation: true }
          }}
        />
      </div>
    </div>
  );
}
