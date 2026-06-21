import { FileCode2, List } from "lucide-react";
import { ViewShell } from "@/components/layout/ViewShell";

type Props = {
  onOpenList: () => void;
};

export function EditorEmptyState({ onOpenList }: Props) {
  return (
    <ViewShell centered>
      <div className="max-w-md text-center">
        <div className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-lg border border-gs-border bg-gs-surface">
          <FileCode2 className="h-7 w-7 text-gs-accent" />
        </div>
        <h2 className="mb-2 text-[15px] font-medium text-gs-fg">Nenhum .env aberto</h2>
        <p className="mb-4 text-[13px] leading-relaxed text-gs-muted">
          A Lista serve para explorar e filtrar findings. Ao seleccionar um .env, abre-se aqui um editor dedicado —
          como no Cursor — só para inspeccionar o ficheiro e correr checkers.
        </p>
        <button type="button" className="gs-btn gs-btn-primary inline-flex items-center gap-2" onClick={onOpenList}>
          <List className="h-4 w-4" />
          Ir para Lista
        </button>
      </div>
    </ViewShell>
  );
}
