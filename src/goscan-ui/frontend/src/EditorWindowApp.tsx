import { useCallback, useEffect, useRef, useState } from "react";
import { FindingEditorWindow } from "@/components/editor/FindingEditorWindow";
import { StatusBar } from "@/components/layout/StatusBar";
import { BottomPanel, EDITOR_BOTTOM_TABS, type BottomTab } from "@/components/terminal/BottomPanel";
import {
  api,
  Events,
  type CheckerResultDTO,
  type FindingDetailDTO,
  type ScriptCheckerStatusDTO
} from "@/lib/api";
import { eventChunk, eventForFinding, sameFindingId, unwrapEventData } from "@/lib/scriptEvents";

function mergeCheckerUpdate(
  list: ScriptCheckerStatusDTO[],
  dto: CheckerResultDTO
): ScriptCheckerStatusDTO[] {
  const idx = list.findIndex((s) => s.scriptId === dto.scriptId);
  const row: ScriptCheckerStatusDTO = {
    scriptId: dto.scriptId,
    label: dto.scriptLabel,
    status: dto.status,
    summary: dto.summary,
    testedAt: dto.testedAt,
    exitCode: dto.exitCode
  };
  if (idx < 0) return [...list, row];
  const next = [...list];
  next[idx] = row;
  return next;
}

type Props = {
  findingId: number;
};

export function EditorWindowApp({ findingId }: Props) {
  const [detail, setDetail] = useState<FindingDetailDTO | null>(null);
  const [scripts, setScripts] = useState<ScriptCheckerStatusDTO[]>([]);
  const [selectedScript, setSelectedScript] = useState("");
  const [runningScript, setRunningScript] = useState<{ findingId: number; scriptId: string } | undefined>();
  const [error, setError] = useState("");
  const [bottomTab, setBottomTab] = useState<BottomTab>("results");
  const [logActive, setLogActive] = useState(false);
  const [logLines, setLogLines] = useState<string[]>([]);

  const findingIdRef = useRef(findingId);
  findingIdRef.current = findingId;
  const scriptsRef = useRef(scripts);
  scriptsRef.current = scripts;

  const appendLog = useCallback((line: string) => {
    setLogLines((prev) => [...prev, line]);
  }, []);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const d = await api.getFinding(findingId);
        if (cancelled) return;
        setDetail(d);
        const [overview] = await api.checkerOverview([findingId]);
        const list = overview?.scripts ?? [];
        setScripts(list);
        setSelectedScript(list[0]?.scriptId ?? "");
        setError("");
      } catch (e) {
        if (!cancelled) setError(String(e));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [findingId]);

  useEffect(() => {
    const forThis = (ev: unknown) => eventForFinding(unwrapEventData(ev), findingIdRef.current);

    const pushChunk = (ev: unknown) => {
      const p = forThis(ev);
      if (!p) return;
      const chunk = eventChunk(p);
      if (!chunk) return;
      chunk.split("\n").forEach((line) => {
        if (line.length > 0) appendLog(line);
      });
    };

    const offStdout = Events.On("script:stdout", pushChunk);
    const offStderr = Events.On("script:stderr", pushChunk);
    const offTermData = Events.On("terminal:data", pushChunk);

    const offExit = (ev: unknown) => {
      const p = forThis(ev);
      if (!p) return;
      const sid = p.scriptId ?? "script";
      appendLog(`--- fim ${sid}: código ${p.exitCode ?? 0} ---`);
      setLogActive(false);
      setRunningScript(undefined);
      setBottomTab("results");
    };

    const offScriptExit = Events.On("script:exit", offExit);
    const offTermExit = Events.On("terminal:exit", offExit);

    const offCheckerRunning = Events.On("checker:running", (ev) => {
      const p = unwrapEventData(ev) as { findingId: number; scriptId: string };
      if (!sameFindingId(p.findingId, findingIdRef.current)) return;
      const label = scriptsRef.current.find((s) => s.scriptId === p.scriptId)?.label ?? p.scriptId;
      setBottomTab("terminal");
      setLogActive(true);
      setLogLines([`${label} — a correr…`]);
      setRunningScript({ findingId: Number(p.findingId), scriptId: p.scriptId });
      setScripts((prev) =>
        prev.map((s) => (s.scriptId === p.scriptId ? { ...s, status: "running" as const } : s))
      );
    });

    const offCheckerUpdated = Events.On("checker:updated", (ev) => {
      const dto = unwrapEventData(ev) as CheckerResultDTO;
      if (!sameFindingId(dto.findingId, findingIdRef.current)) return;
      setScripts((prev) => mergeCheckerUpdate(prev, dto));
      setRunningScript(undefined);
      setLogActive(false);
    });

    return () => {
      offStdout();
      offStderr();
      offTermData();
      offScriptExit();
      offTermExit();
      offCheckerRunning();
      offCheckerUpdated();
    };
  }, [appendLog]);

  const runScript = useCallback(
    async (scriptId?: string) => {
      const sid = scriptId ?? selectedScript;
      if (!sid) return;
      setSelectedScript(sid);
      try {
        setBottomTab("terminal");
        await api.runScript(sid, findingId);
      } catch (e) {
        setError(String(e));
        setRunningScript(undefined);
        setLogActive(false);
      }
    },
    [findingId, selectedScript]
  );

  const panelRunningId = runningScript?.findingId === findingId ? runningScript.scriptId : undefined;

  if (!detail) {
    return (
      <div className="flex h-screen items-center justify-center bg-gs-bg text-gs-muted">
        {error || "A carregar .env…"}
      </div>
    );
  }

  return (
    <div className="flex h-screen flex-col bg-gs-bg text-gs-fg">
      <FindingEditorWindow
        standalone
        detail={detail}
        onFocusMain={() => void api.focusMainWindow()}
        scripts={scripts}
        selectedScript={selectedScript}
        onScriptChange={setSelectedScript}
        onRunScript={(id) => void runScript(id)}
        onCancelScript={() => void api.cancelScript()}
        terminalActive={logActive}
        runningScriptId={panelRunningId}
      />
      <BottomPanel
        tabs={EDITOR_BOTTOM_TABS}
        tab={bottomTab}
        onTabChange={setBottomTab}
        outputLines={[]}
        batchLogLines={[]}
        onClearOutput={() => {}}
        onClearBatchLog={() => {}}
        logLines={logLines}
        onClearLog={() => setLogLines([])}
        terminalActive={logActive}
        resultsScripts={scripts}
        resultsFindingLabel={`${detail.domain}${detail.path}`}
        runningScriptId={panelRunningId}
      />
      <StatusBar findingLabel={`${detail.domain}${detail.path}`} findingsCount={1} error={error} />
    </div>
  );
}
