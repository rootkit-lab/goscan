import Editor from "@monaco-editor/react";
import type { FindingDetailDTO } from "@/lib/api";

type Props = {
  detail: FindingDetailDTO | null;
};

export function EditorArea({ detail }: Props) {
  if (!detail) {
    return (
      <div className="flex h-full flex-col">
        <div className="flex h-[35px] items-center border-b border-vscode-border bg-vscode-tab-inactive px-3 text-vscode-muted">
          Welcome
        </div>
        <div className="flex flex-1 items-center justify-center text-vscode-muted">Select a finding from the sidebar</div>
      </div>
    );
  }

  const tabLabel = `${detail.domain} · ${detail.path}`;

  return (
    <div className="flex h-full flex-col">
      <div className="flex h-[35px] shrink-0 items-end border-b border-vscode-border bg-vscode-tab-inactive">
        <div className="flex h-full items-center border-r border-vscode-border bg-vscode-tab-active px-3 text-vscode-fg">
          <span className="max-w-[420px] truncate text-[13px]">{tabLabel}</span>
        </div>
      </div>
      <div className="flex h-[22px] shrink-0 items-center gap-1 border-b border-vscode-border px-3 text-[11px] text-vscode-muted">
        <span>{detail.domain}</span>
        <span>›</span>
        <span>{detail.path}</span>
      </div>
      <div className="min-h-0 flex-1">
        <Editor
          height="100%"
          defaultLanguage="plaintext"
          theme="vs-dark"
          value={detail.content}
          options={{
            minimap: { enabled: false },
            fontSize: 13,
            fontFamily: "var(--vscode-font-mono)",
            readOnly: true,
            scrollBeyondLastLine: false,
            padding: { top: 8 },
            lineNumbers: "on",
            renderLineHighlight: "line"
          }}
        />
      </div>
    </div>
  );
}
