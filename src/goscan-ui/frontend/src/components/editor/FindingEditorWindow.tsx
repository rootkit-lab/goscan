import { ArrowLeft, Key } from "lucide-react";
import { FindingContextPanel } from "@/components/actions/FindingContextPanel";
import { CursorEditor } from "@/components/editor/CursorEditor";
import type { FindingDetailDTO, ScriptCheckerStatusDTO } from "@/lib/api";

type Props = {
  detail: FindingDetailDTO;
  standalone?: boolean;
  onBackToList?: () => void;
  onFocusMain?: () => void;
  scripts: ScriptCheckerStatusDTO[];
  selectedScript: string;
  onScriptChange: (id: string) => void;
  onRunScript: (scriptId?: string) => void;
  onCancelScript?: () => void;
  terminalActive?: boolean;
  runningScriptId?: string;
};

export function FindingEditorWindow({
  detail,
  standalone,
  onBackToList,
  onFocusMain,
  scripts,
  selectedScript,
  onScriptChange,
  onRunScript,
  onCancelScript,
  terminalActive,
  runningScriptId
}: Props) {
  const showNav = standalone ? onFocusMain : onBackToList;

  return (
    <div className="flex min-h-0 flex-1 flex-col bg-gs-bg">
      <header className="flex h-9 shrink-0 items-center gap-2 border-b border-gs-border bg-gs-surface px-2">
        {showNav && (
          <>
            <button
              type="button"
              onClick={standalone ? onFocusMain : onBackToList}
              className="flex items-center gap-1 rounded-sm px-2 py-1 text-[12px] text-gs-muted transition-colors hover:bg-gs-hover hover:text-gs-fg"
              title={standalone ? "Focar janela principal" : "Voltar à lista (Alt+1)"}
            >
              <ArrowLeft className="h-3.5 w-3.5" />
              {standalone ? "Principal" : "Lista"}
            </button>
            <div className="h-4 w-px bg-gs-border" />
          </>
        )}
        <span className="truncate text-[12px] text-gs-fg">
          {detail.domain}
          <span className="text-gs-muted">{detail.path}</span>
        </span>
        {detail.hasCredentials && (
          <span className="inline-flex items-center gap-1 rounded bg-gs-warning/10 px-1.5 py-0.5 text-[10px] text-gs-warning">
            <Key className="h-3 w-3" />
            credenciais
          </span>
        )}
        <span className="ml-auto text-[11px] text-gs-muted">
          {standalone ? "Janela .env" : "Editor"} · read-only
        </span>
      </header>

      <div className="flex min-h-0 flex-1">
        <main className="min-w-0 flex-1">
          <CursorEditor detail={detail} />
        </main>
        <aside className="flex w-[300px] shrink-0 flex-col border-l border-gs-border bg-gs-surface">
          <FindingContextPanel
            scripts={scripts}
            selectedScript={selectedScript}
            onScriptChange={onScriptChange}
            onRunScript={onRunScript}
            onCancelScript={onCancelScript}
            terminalActive={terminalActive}
            runningScriptId={runningScriptId}
          />
        </aside>
      </div>
    </div>
  );
}
