export type WorkbenchView = "findings" | "batch" | "settings";

export const WORKBENCH_VIEWS: { id: WorkbenchView; label: string; shortcut?: string }[] = [
  { id: "findings", label: "Lista", shortcut: "1" },
  { id: "batch", label: "Batch", shortcut: "2" },
  { id: "settings", label: "Settings", shortcut: "3" }
];

export function workbenchViewLabel(view: WorkbenchView): string {
  return WORKBENCH_VIEWS.find((v) => v.id === view)?.label ?? view;
}

export function parseEditorWindowParams(): number | null {
  const params = new URLSearchParams(window.location.search);
  if (params.get("window") !== "editor") return null;
  const id = Number(params.get("findingId"));
  return Number.isFinite(id) && id > 0 ? id : null;
}
