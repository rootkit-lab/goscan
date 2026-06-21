import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ActionPanel } from "@/components/actions/ActionPanel";
import { CommandPalette } from "@/components/command/CommandPalette";
import { EditorArea } from "@/components/editor/EditorArea";
import { StatusBar } from "@/components/layout/StatusBar";
import { WorkbenchLayout } from "@/components/layout/WorkbenchLayout";
import { FindingsSidebar } from "@/components/sidebar/FindingsSidebar";
import { BottomPanel, type BottomTab } from "@/components/terminal/BottomPanel";
import { useInteractiveTerminal } from "@/hooks/useInteractiveTerminal";
import {
  api,
  Events,
  type BatchProgressDTO,
  type CheckerResultDTO,
  type FindingDetailDTO,
  type FindingsStatsDTO,
  type ScanProgressDTO,
  type ScriptCheckerStatusDTO,
  type SettingsDTO
} from "@/lib/api";
import { type CheckerResultFilter, findingMatchesCheckerFilter } from "@/lib/checkerFilters";

function useDebounce<T>(value: T, ms: number): T {
  const [v, setV] = useState(value);
  useEffect(() => {
    const t = setTimeout(() => setV(value), ms);
    return () => clearTimeout(t);
  }, [value, ms]);
  return v;
}

function overviewToMap(items: Awaited<ReturnType<typeof api.checkerOverview>>) {
  const map: Record<number, ScriptCheckerStatusDTO[]> = {};
  for (const o of items) map[o.findingId] = o.scripts;
  return map;
}

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

export function App() {
  const [query, setQuery] = useState("");
  const debouncedQuery = useDebounce(query, 150);
  const [confidence, setConfidence] = useState("");
  const [unopenedOnly, setUnopenedOnly] = useState(false);
  const [checkerFilter, setCheckerFilter] = useState<CheckerResultFilter>("");
  const [findingsStats, setFindingsStats] = useState<FindingsStatsDTO>({ total: 0, unopened: 0 });
  const [findings, setFindings] = useState<Awaited<ReturnType<typeof api.searchFindings>>>([]);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [detail, setDetail] = useState<FindingDetailDTO | null>(null);
  const [scripts, setScripts] = useState<ScriptCheckerStatusDTO[]>([]);
  const [selectedScript, setSelectedScript] = useState("");
  const [checkerOverview, setCheckerOverview] = useState<Record<number, ScriptCheckerStatusDTO[]>>({});
  const [runningScript, setRunningScript] = useState<{ findingId: number; scriptId: string } | undefined>();
  const [scanProgress, setScanProgress] = useState<ScanProgressDTO | null>(null);
  const [scanOpts, setScanOpts] = useState({ threads: 50, fast: false, rescan: false, timeoutSec: 8 });
  const [error, setError] = useState("");
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [bottomTab, setBottomTab] = useState<BottomTab>("output");
  const [outputLines, setOutputLines] = useState<string[]>([]);
  const [terminalActive, setTerminalActive] = useState(false);
  const [batchRunning, setBatchRunning] = useState(false);
  const [batchProgress, setBatchProgress] = useState<BatchProgressDTO | null>(null);
  const [batchLogDir, setBatchLogDir] = useState("");
  const [batchThreads, setBatchThreads] = useState(4);
  const [settings, setSettings] = useState<SettingsDTO | null>(null);
  const [draftDataDir, setDraftDataDir] = useState("");
  const [draftScanDir, setDraftScanDir] = useState("");
  const [settingsSaving, setSettingsSaving] = useState(false);

  const selectedIdRef = useRef(selectedId);
  selectedIdRef.current = selectedId;

  const termRef = useRef<HTMLDivElement>(null);
  const { write, reset, focus, fitTerminal } = useInteractiveTerminal(termRef, {
    enabled: true,
    onData: (data) => void api.terminalInput(data),
    onResize: (cols, rows) => {
      if (bottomTab === "terminal") void api.terminalResize(cols, rows);
    }
  });

  const appendOutput = useCallback((line: string) => {
    setOutputLines((prev) => [...prev, line]);
  }, []);

  const refreshOverview = useCallback(async (ids: number[]) => {
    if (ids.length === 0) return;
    try {
      const items = await api.checkerOverview(ids);
      setCheckerOverview((prev) => ({ ...prev, ...overviewToMap(items) }));
    } catch {
      /* ignore batch errors */
    }
  }, []);

  const loadSettings = useCallback(async () => {
    try {
      const s = await api.getSettings();
      setSettings(s);
      setDraftDataDir(s.dataDir);
      setDraftScanDir(s.scanDir);
    } catch (e) {
      setError(String(e));
    }
  }, []);

  const loadStats = useCallback(async () => {
    try {
      setFindingsStats(await api.findingsStats());
    } catch {
      /* ignore */
    }
  }, []);

  const loadFindings = useCallback(async () => {
    try {
      const list = await api.searchFindings(debouncedQuery, confidence, unopenedOnly, 200);
      setFindings(list);
      setError("");
      void refreshOverview(list.map((f) => f.id));
      void loadStats();
    } catch (e) {
      setError(String(e));
    }
  }, [debouncedQuery, confidence, unopenedOnly, refreshOverview, loadStats]);

  const displayedFindings = useMemo(
    () =>
      checkerFilter
        ? findings.filter((f) => findingMatchesCheckerFilter(f.id, checkerFilter, checkerOverview))
        : findings,
    [findings, checkerFilter, checkerOverview]
  );

  useEffect(() => {
    void loadSettings();
  }, [loadSettings]);

  useEffect(() => {
    void loadFindings();
  }, [loadFindings]);

  useEffect(() => {
    void loadStats();
  }, [loadStats]);

  useEffect(() => {
    if (bottomTab === "terminal") {
      const t = setTimeout(() => fitTerminal(), 100);
      return () => clearTimeout(t);
    }
  }, [bottomTab, fitTerminal]);

  useEffect(() => {
    const mq = window.matchMedia("(max-width: 1023px)");
    const apply = () => setSidebarCollapsed(mq.matches);
    apply();
    mq.addEventListener("change", apply);
    return () => mq.removeEventListener("change", apply);
  }, []);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        setPaletteOpen(true);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  const selectFinding = async (id: number) => {
    setSelectedId(id);
    const wasNew = findings.find((f) => f.id === id)?.isNew;
    if (wasNew) {
      setFindingsStats((prev) => ({ ...prev, unopened: Math.max(0, prev.unopened - 1) }));
      setFindings((prev) =>
        unopenedOnly ? prev.filter((f) => f.id !== id) : prev.map((f) => (f.id === id ? { ...f, isNew: false } : f))
      );
    }
    try {
      const d = await api.getFinding(id);
      setDetail(d);
      const [overview] = await api.checkerOverview([id]);
      const list = overview?.scripts ?? [];
      setScripts(list);
      setCheckerOverview((prev) => ({ ...prev, [id]: list }));
      setSelectedScript(list[0]?.scriptId ?? "");
    } catch (e) {
      setError(String(e));
    }
  };

  const applyCheckerUpdate = useCallback((dto: CheckerResultDTO) => {
    setCheckerOverview((prev) => ({
      ...prev,
      [dto.findingId]: mergeCheckerUpdate(prev[dto.findingId] ?? [], dto)
    }));
    if (selectedIdRef.current === dto.findingId) {
      setScripts((prev) => mergeCheckerUpdate(prev, dto));
    }
    setRunningScript(undefined);
  }, []);

  useEffect(() => {
    const data = (ev: unknown) => (ev as { data?: unknown }).data ?? ev;

    const offOutput = Events.On("scan:output", (ev) => appendOutput(String(data(ev))));
    const offProgress = Events.On("scan:progress", (ev) => {
      const p = data(ev) as ScanProgressDTO;
      setScanProgress(p);
      if (p.running && p.domainsScanned > 0 && p.domainsScanned % 25 === 0) {
        appendOutput(`… ${p.domainsScanned} domínios · ${p.vulnsFound} vulns`);
      }
    });
    const offFound = Events.On("scan:found", () => void loadFindings());

    const offTermStart = Events.On("terminal:start", (ev) => {
      const p = data(ev) as { scriptId: string; label?: string; python?: string };
      setBottomTab("terminal");
      setTerminalActive(true);
      reset();
      const pyLine = p.python ? `\r\n\x1b[90m${p.python}\x1b[0m` : "";
      write(`\r\n\x1b[1m${p.label ?? p.scriptId}\x1b[0m${pyLine}\r\n`);
      setTimeout(() => {
        focus();
        fitTerminal();
      }, 50);
    });
    const offTermData = Events.On("terminal:data", (ev) => write(String(data(ev))));
    const offTermExit = Events.On("terminal:exit", (ev) => {
      const p = data(ev) as { exitCode: number; scriptId: string };
      write(`\r\n--- exit ${p.scriptId}: code ${p.exitCode} ---\r\n`);
      setTerminalActive(false);
      setRunningScript(undefined);
    });

    const offCheckerRunning = Events.On("checker:running", (ev) => {
      const p = data(ev) as { findingId: number; scriptId: string };
      setRunningScript({ findingId: p.findingId, scriptId: p.scriptId });
      const markRunning = (list: ScriptCheckerStatusDTO[]) =>
        list.map((s) => (s.scriptId === p.scriptId ? { ...s, status: "running" as const } : s));
      setCheckerOverview((prev) => ({
        ...prev,
        [p.findingId]: markRunning(prev[p.findingId] ?? [])
      }));
      if (selectedIdRef.current === p.findingId) {
        setScripts((prev) => markRunning(prev));
      }
    });
    const offCheckerUpdated = Events.On("checker:updated", (ev) => {
      const dto = data(ev) as CheckerResultDTO;
      applyCheckerUpdate(dto);
      void refreshOverview([dto.findingId]);
    });

    const offBatchOutput = Events.On("batch:output", (ev) => {
      setBottomTab("output");
      appendOutput(String(data(ev)));
    });
    const offBatchProgress = Events.On("batch:progress", (ev) => {
      const p = data(ev) as BatchProgressDTO;
      setBatchRunning(p.running);
      setBatchProgress(p.running ? p : null);
    });
    const offBatchDone = Events.On("batch:done", (ev) => {
      setBatchRunning(false);
      setBatchProgress(null);
      void refreshOverview(findings.map((f) => f.id));
      const d = data(ev) as { ok: number; fail: number; skip: number; total: number; secs: number; logDir?: string };
      if (d.logDir) setBatchLogDir(d.logDir);
      appendOutput(`Batch concluído — OK ${d.ok} · FAIL ${d.fail} · SKIP ${d.skip} · ${d.secs}s`);
    });

    return () => {
      offOutput();
      offProgress();
      offFound();
      offTermStart();
      offTermData();
      offTermExit();
      offCheckerRunning();
      offCheckerUpdated();
      offBatchOutput();
      offBatchProgress();
      offBatchDone();
    };
  }, [appendOutput, loadFindings, write, reset, focus, fitTerminal, applyCheckerUpdate, refreshOverview, findings]);

  const runScript = async (scriptId?: string) => {
    const sid = scriptId ?? selectedScript;
    if (!selectedId || !sid) return;
    setSelectedScript(sid);
    const row = scripts.find((s) => s.scriptId === sid);
    try {
      if (row?.status !== "running") {
        setBottomTab("terminal");
      }
      await api.runScript(sid, selectedId);
    } catch (e) {
      setError(String(e));
      setRunningScript(undefined);
    }
  };

  const startScan = async () => {
    setBottomTab("output");
    try {
      await api.startScan({ ...scanOpts, dir: draftScanDir || settings?.scanDir || undefined });
    } catch (e) {
      setError(String(e));
    }
  };

  const pickDataDir = async () => {
    try {
      const picked = await api.pickDirectory("Pasta de dados (DB, findings)", draftDataDir);
      setDraftDataDir(picked);
    } catch (e) {
      const msg = String(e);
      if (!msg.includes("cancel") && !msg.includes("nenhuma")) setError(msg);
    }
  };

  const pickScanDir = async () => {
    try {
      const picked = await api.pickDirectory("Pasta com listas de domínios", draftScanDir || draftDataDir);
      setDraftScanDir(picked);
    } catch (e) {
      const msg = String(e);
      if (!msg.includes("cancel") && !msg.includes("nenhuma")) setError(msg);
    }
  };

  const saveSettings = async () => {
    setSettingsSaving(true);
    setError("");
    try {
      await api.saveSettings(draftDataDir, draftScanDir);
      await loadSettings();
      await loadFindings();
      await loadStats();
    } catch (e) {
      setError(String(e));
    } finally {
      setSettingsSaving(false);
    }
  };

  const startBatch = async (opts: { findingOnly?: boolean; quick?: boolean; threads?: number }) => {
    setBottomTab("output");
    setError("");
    setBatchLogDir("");
    const threads = opts.threads ?? 1;
    try {
      await api.startBatchCheck({
        findingId: opts.findingOnly && selectedId ? selectedId : 0,
        query: debouncedQuery,
        confidence,
        unopenedOnly: opts.findingOnly ? false : unopenedOnly,
        quick: opts.quick ?? false,
        limit: opts.findingOnly ? 1 : 500,
        threads
      });
    } catch (e) {
      setError(String(e));
    }
  };

  const batchLabel = batchProgress
    ? `${batchProgress.domain} · ${batchProgress.scriptLabel} (${batchProgress.scriptIndex}/${batchProgress.scriptTotal})`
    : undefined;

  const scanStats = scanProgress
    ? `${scanProgress.domainsScanned} dom · ${scanProgress.vulnsFound} vulns`
    : undefined;

  const panelRunningId = runningScript?.findingId === selectedId ? runningScript.scriptId : undefined;

  return (
    <>
      <WorkbenchLayout
        sidebarCollapsed={sidebarCollapsed}
        sidebar={
          <FindingsSidebar
            query={query}
            onQueryChange={setQuery}
            confidence={confidence}
            onConfidenceChange={setConfidence}
            unopenedOnly={unopenedOnly}
            onUnopenedOnlyChange={setUnopenedOnly}
            unopenedCount={findingsStats.unopened}
            checkerFilter={checkerFilter}
            onCheckerFilterChange={setCheckerFilter}
            findings={displayedFindings}
            findingIdsForCounts={findings.map((f) => f.id)}
            selectedId={selectedId}
            onSelect={(id) => void selectFinding(id)}
            checkerOverview={checkerOverview}
            runningScript={runningScript}
          />
        }
        editor={<EditorArea detail={detail} />}
        actions={
          <ActionPanel
            scripts={scripts}
            selectedScript={selectedScript}
            onScriptChange={setSelectedScript}
            onRunScript={(id) => void runScript(id)}
            scanOpts={scanOpts}
            onScanOptsChange={setScanOpts}
            onStartScan={() => void startScan()}
            onCancelScan={() => void api.cancelScan()}
            onCancelScript={() => void api.cancelScript()}
            onTestAllFinding={() => void startBatch({ findingOnly: true })}
            onTestAllFiltered={() => void startBatch({ findingOnly: false })}
            onTestAllQuick={() => void startBatch({ findingOnly: false, quick: true })}
            onTestAllEnvs={() => void startBatch({ findingOnly: false, threads: batchThreads })}
            onCancelBatch={() => void api.cancelBatchCheck()}
            batchRunning={batchRunning}
            batchProgress={batchProgress}
            batchLabel={batchLabel}
            batchThreads={batchThreads}
            onBatchThreadsChange={setBatchThreads}
            terminalActive={terminalActive}
            runningScriptId={panelRunningId}
            settings={settings}
            draftDataDir={draftDataDir}
            draftScanDir={draftScanDir}
            onDraftDataDirChange={setDraftDataDir}
            onDraftScanDirChange={setDraftScanDir}
            onPickDataDir={() => void pickDataDir()}
            onPickScanDir={() => void pickScanDir()}
            onSaveSettings={() => void saveSettings()}
            onOpenDataDir={() => void api.openDataDirectory()}
            onOpenScanDir={() => void api.openScanDirectory()}
            settingsSaving={settingsSaving}
          />
        }
        terminal={
          <BottomPanel
            tab={bottomTab}
            onTabChange={setBottomTab}
            outputLines={outputLines}
            onClearOutput={() => setOutputLines([])}
            termRef={termRef}
            onClearTerminal={reset}
            terminalActive={terminalActive}
            batchRunning={batchRunning}
            batchProgress={batchProgress}
            batchLogDir={batchLogDir}
            onOpenBatchLogs={() => void api.openBatchLogDir(batchLogDir)}
          />
        }
        statusBar={
          <StatusBar
            findingLabel={detail ? `${detail.domain}${detail.path}` : undefined}
            findingsCount={displayedFindings.length}
            unopenedCount={findingsStats.unopened}
            unopenedFilter={unopenedOnly}
            checkerFilter={checkerFilter}
            scanRunning={scanProgress?.running || batchRunning}
            scanStats={batchRunning ? batchLabel : scanStats}
            error={error}
            mode={settings?.mode}
            dataDir={settings?.dataDir}
          />
        }
      />
      <CommandPalette
        open={paletteOpen}
        onClose={() => setPaletteOpen(false)}
        query={query}
        onQueryChange={setQuery}
        findings={findings}
        onSelectFinding={(id) => void selectFinding(id)}
        onRunScript={() => void runScript()}
        onStartScan={() => void startScan()}
        onTestAllFinding={() => void startBatch({ findingOnly: true })}
        onTestAllFiltered={() => void startBatch({ findingOnly: false })}
      />
    </>
  );
}
