import type { ReactNode } from "react";
import { ActivityBar } from "@/components/layout/ActivityBar";
import type { WorkbenchView } from "@/lib/workbenchView";

type Props = {
  view: WorkbenchView;
  onViewChange: (view: WorkbenchView) => void;
  batchActive?: boolean;
  main: ReactNode;
  terminal?: ReactNode | null;
  statusBar: ReactNode;
};

export function WorkbenchLayout({ view, onViewChange, batchActive, main, terminal, statusBar }: Props) {
  return (
    <div className="flex h-screen flex-col bg-gs-bg text-gs-fg">
      <div className="flex min-h-0 flex-1">
        <ActivityBar view={view} onViewChange={onViewChange} batchActive={batchActive} />
        <div className="flex min-h-0 min-w-0 flex-1 flex-col">
          <div className="flex min-h-0 min-w-0 flex-1 flex-col [&>*]:min-h-0 [&>*]:min-w-0 [&>*]:flex-1">{main}</div>
          {terminal ?? null}
        </div>
      </div>
      {statusBar}
    </div>
  );
}
