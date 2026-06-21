import { ScriptList } from "@/components/actions/ScriptList";
import { CollapsibleSection } from "@/components/layout/CollapsibleSection";
import { ZoneHeader } from "@/components/layout/ZoneHeader";
import type { ScriptCheckerStatusDTO } from "@/lib/api";

type Props = {
  scripts: ScriptCheckerStatusDTO[];
  selectedScript: string;
  onScriptChange: (id: string) => void;
  onRunScript: (scriptId?: string) => void;
  onCancelScript?: () => void;
  terminalActive?: boolean;
  runningScriptId?: string;
};

export function FindingContextPanel({
  scripts,
  selectedScript,
  onScriptChange,
  onRunScript,
  onCancelScript,
  terminalActive,
  runningScriptId
}: Props) {
  return (
    <div className="flex h-full flex-col overflow-auto bg-gs-surface">
      <ZoneHeader>Operar</ZoneHeader>

      <CollapsibleSection title="Scripts" defaultOpen>
        <ScriptList
          scripts={scripts}
          selectedScript={selectedScript}
          onScriptChange={onScriptChange}
          onRunScript={onRunScript}
          onCancelScript={onCancelScript}
          terminalActive={terminalActive}
          runningScriptId={runningScriptId}
        />
      </CollapsibleSection>
    </div>
  );
}
