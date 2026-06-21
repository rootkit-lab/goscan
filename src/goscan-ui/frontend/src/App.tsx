import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ScanPanel } from "@/components/actions/ScanPanel";
import { batchProgressToRow, upsertBatchResult, type BatchResultRow } from "@/components/batch/batchResults";
import { BatchPanel } from "@/components/batch/BatchPanel";
import { CommandPalette } from "@/components/command/CommandPalette";
import { StatusBar } from "@/components/layout/StatusBar";
import { WorkbenchLayout } from "@/components/layout/WorkbenchLayout";
import { SettingsView } from "@/components/settings/SettingsView";
import type { DraftWorker } from "@/components/settings/RemoteWorkersSection";
import { FindingsSidebar } from "@/components/sidebar/FindingsSidebar";
import { BottomPanel, FINDINGS_BOTTOM_TABS, type BottomTab } from "@/components/terminal/BottomPanel";
import {
  api,
  Events,
  type BatchProgressDTO,
  type CheckerResultDTO,
  type FindingsStatsDTO,
  type ScanProgressDTO,
  type ScanWorkerProgressDTO,
  type ScriptCheckerStatusDTO,
  type SettingsDTO
} from "@/lib/api";
import {
  alertEnvFound,
  alertPrefsFromSettings,
  alertScriptOk,
  type AlertPrefs
} from "@/lib/alerts";
import { type CheckerResultFilter, findingMatchesCheckerFilter } from "@/lib/checkerFilters";
import { type WorkbenchView } from "@/lib/workbenchView";

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
  const [workbenchView, setWorkbenchView] = useState<WorkbenchView>("findings");
  const [query, setQuery] = useState("");
  const debouncedQuery = useDebounce(query, 150);
  const [confidence, setConfidence] = useState("");
  const [unopenedOnly, setUnopenedOnly] = useState(false);
  const [checkerFilter, setCheckerFilter] = useState<CheckerResultFilter>("");
  const [findingsStats, setFindingsStats] = useState<FindingsStatsDTO>({ total: 0, unopened: 0 });
  const [findings, setFindings] = useState<Awaited<ReturnType<typeof api.searchFindings>>>([]);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [checkerOverview, setCheckerOverview] = useState<Record<number, ScriptCheckerStatusDTO[]>>({});
  const [runningScript, setRunningScript] = useState<{ findingId: number; scriptId: string } | undefined>();
  const [scanProgress, setScanProgress] = useState<ScanProgressDTO | null>(null);
  const [workerProgress, setWorkerProgress] = useState<ScanWorkerProgressDTO[]>([]);
  const [scanLogLines, setScanLogLines] = useState<string[]>([]);
  const [scanLogFilter, setScanLogFilter] = useState("all");
  const [scanOpts, setScanOpts] = useState({
    threads: 50,
    fast: false,
    rescan: false,
    timeoutSec: 8,
    targets: [] as string[],
    deployRemote: false
  });
  const [error, setError] = useState("");
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [bottomTab, setBottomTab] = useState<BottomTab>("scan-log");
  const [batchLogLines, setBatchLogLines] = useState<string[]>([]);
  const [batchRunning, setBatchRunning] = useState(false);
  const [batchProgress, setBatchProgress] = useState<BatchProgressDTO | null>(null);
  const [batchResults, setBatchResults] = useState<BatchResultRow[]>([]);
  const [batchLogDir, setBatchLogDir] = useState("");
  const [batchThreads, setBatchThreads] = useState(4);
  const [batchUntestedOnly, setBatchUntestedOnly] = useState(true);
  const [batchForceRecheck, setBatchForceRecheck] = useState(false);
  const [settings, setSettings] = useState<SettingsDTO | null>(null);
  const [draftDataDir, setDraftDataDir] = useState("");
  const [draftScanDir, setDraftScanDir] = useState("");
  const [draftPythonPath, setDraftPythonPath] = useState("");
  const [draftNotifyEnvFound, setDraftNotifyEnvFound] = useState(true);
  const [draftNotifyScriptOk, setDraftNotifyScriptOk] = useState(true);
  const [draftSoundEnvFound, setDraftSoundEnvFound] = useState(false);
  const [draftSoundScriptOk, setDraftSoundScriptOk] = useState(true);
  const [draftWorkers, setDraftWorkers] = useState<DraftWorker[]>([]);
  const [draftDeployRepoUrl, setDraftDeployRepoUrl] = useState("");
  const [draftDeployRepoRef, setDraftDeployRepoRef] = useState("");
  const [draftDeployRepoToken, setDraftDeployRepoToken] = useState("");
  const [draftDeployRepoMethod, setDraftDeployRepoMethod] = useState("git");
  const [draftHubEnabled, setDraftHubEnabled] = useState(true);
  const [settingsSaving, setSettingsSaving] = useState(false);

  const selectedIdRef = useRef(selectedId);
  selectedIdRef.current = selectedId;
  const alertPrefsRef = useRef<AlertPrefs>(alertPrefsFromSettings({}));

  const appendBatchLog = useCallback((line: string) => {
    setBatchLogLines((prev) => [...prev, line]);
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
      setDraftPythonPath(s.pythonPath);
      setDraftNotifyEnvFound(s.notifyEnvFound);
      setDraftNotifyScriptOk(s.notifyScriptOk);
      setDraftSoundEnvFound(s.soundEnvFound);
      setDraftSoundScriptOk(s.soundScriptOk);
      setDraftDeployRepoUrl(s.deployRepoUrl ?? "");
      setDraftDeployRepoRef(s.deployRepoRef ?? "");
      setDraftDeployRepoMethod(s.deployRepoMethod || "git");
      setDraftDeployRepoToken("");
      setDraftHubEnabled(s.hubEnabled ?? true);
      setDraftWorkers(
        (s.workers ?? []).map((w) => ({
          id: w.id,
          name: w.name,
          host: w.host,
          port: w.port || 22,
          user: w.user,
          authType: w.authType || "password",
          password: "",
          keyPath: w.keyPath,
          keyPassphrase: "",
          execMode: w.execMode || "ssh",
          apiPort: w.apiPort || 9090,
          apiToken: "",
          enabled: w.enabled
        }))
      );
      alertPrefsRef.current = alertPrefsFromSettings(s);
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
      const list = await api.searchFindings(debouncedQuery, confidence, unopenedOnly, 500);
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
    const onKey = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        setPaletteOpen(true);
      }
      if (e.altKey && !e.ctrlKey && !e.metaKey) {
        const views: WorkbenchView[] = ["findings", "batch", "settings"];
        const idx = Number(e.key) - 1;
        if (idx >= 0 && idx < views.length) {
          e.preventDefault();
          setWorkbenchView(views[idx]);
        }
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  const openFinding = async (id: number) => {
    setSelectedId(id);
    const wasNew = findings.find((f) => f.id === id)?.isNew;
    if (wasNew) {
      setFindingsStats((prev) => ({ ...prev, unopened: Math.max(0, prev.unopened - 1) }));
      setFindings((prev) =>
        unopenedOnly
          ? prev.filter((f) => f.id !== id)
          : prev.map((f) =>
              f.id === id ? { ...f, isNew: false, openedAt: new Date().toISOString() } : f
            )
      );
    }
    try {
      await api.openEditorWindow(id);
      setError("");
    } catch (e) {
      setError(String(e));
    }
  };

  const applyCheckerUpdate = useCallback((dto: CheckerResultDTO) => {
    setCheckerOverview((prev) => ({
      ...prev,
      [dto.findingId]: mergeCheckerUpdate(prev[dto.findingId] ?? [], dto)
    }));
    setRunningScript(undefined);
  }, []);

  useEffect(() => {
    const data = (ev: unknown) => (ev as { data?: unknown }).data ?? ev;

    const offProgress = Events.On("scan:progress", (ev) => {
      const p = data(ev) as ScanProgressDTO;
      setScanProgress(p);
      if (!p.running) {
        setWorkerProgress((prev) => prev.map((w) => ({ ...w, running: false, status: w.running ? "cancelled" : w.status })));
        void loadFindings();
      }
    });
    const offWorkerProgress = Events.On("scan:worker-progress", (ev) => {
      const wp = data(ev) as ScanWorkerProgressDTO;
      setWorkerProgress((prev) => {
        const idx = prev.findIndex((x) => x.workerId === wp.workerId);
        if (idx < 0) return [...prev, wp];
        const next = [...prev];
        next[idx] = wp;
        return next;
      });
    });
    const offScanOutput = Events.On("scan:output", (ev) => {
      const line = String(data(ev));
      setScanLogLines((prev) => [...prev.slice(-2000), line]);
    });
    const offFound = Events.On("scan:found", (ev) => {
      const found = data(ev) as { domain?: string; path?: string; isNew?: boolean };
      if (found.isNew !== false) {
        alertEnvFound(alertPrefsRef.current, found.domain ?? "", found.path ?? "");
      }
      void loadFindings();
    });
    const offFindingsRefresh = Events.On("scan:findings-refresh", () => {
      void loadFindings();
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
    });
    const offCheckerUpdated = Events.On("checker:updated", (ev) => {
      const dto = data(ev) as CheckerResultDTO;
      applyCheckerUpdate(dto);
      if (dto.status === "ok") {
        const domain = findings.find((f) => f.id === dto.findingId)?.domain ?? `#${dto.findingId}`;
        alertScriptOk(alertPrefsRef.current, domain, dto.scriptLabel, dto.summary);
      }
      void refreshOverview([dto.findingId]);
    });

    const offBatchOutput = Events.On("batch:output", (ev) => {
      appendBatchLog(String(data(ev)));
    });
    const offBatchProgress = Events.On("batch:progress", (ev) => {
      const p = data(ev) as BatchProgressDTO;
      setBatchRunning(p.running);
      if (p.running) {
        setBatchProgress(p);
        if (p.findingId > 0 && p.scriptId) {
          setBatchResults((prev) =>
            upsertBatchResult(
              prev,
              batchProgressToRow({
                findingId: p.findingId,
                domain: p.domain,
                scriptId: p.scriptId,
                scriptLabel: p.scriptLabel,
                status: p.status,
                summary: p.summary,
                exitCode: p.exitCode,
                checkIndex: p.checkIndex
              })
            )
          );
        }
      } else {
        setBatchProgress((prev) => (prev ? { ...prev, running: false } : null));
      }
    });
    const offBatchDone = Events.On("batch:done", (ev) => {
      setBatchRunning(false);
      void refreshOverview(findings.map((f) => f.id));
      const d = data(ev) as { ok: number; fail: number; skip: number; total: number; secs: number; logDir?: string };
      if (d.logDir) setBatchLogDir(d.logDir);
      appendBatchLog(`Batch concluído — OK ${d.ok} · FAIL ${d.fail} · SKIP ${d.skip} · ${d.secs}s`);
      setBatchProgress((prev) =>
        prev
          ? {
              ...prev,
              running: false,
              okCount: d.ok,
              failCount: d.fail,
              skipCount: d.skip,
              checkIndex: d.total || prev.checkIndex,
              checkTotal: d.total || prev.checkTotal
            }
          : null
      );
    });

    return () => {
      offProgress();
      offWorkerProgress();
      offScanOutput();
      offFound();
      offFindingsRefresh();
      offCheckerRunning();
      offCheckerUpdated();
      offBatchOutput();
      offBatchProgress();
      offBatchDone();
    };
  }, [appendBatchLog, loadFindings, applyCheckerUpdate, refreshOverview, findings]);

  const startScan = async () => {
    try {
      setWorkerProgress([]);
      setScanLogLines([]);
      setScanProgress({ domainsScanned: 0, vulnsFound: 0, domainsNew: 0, domainsPending: 0, running: true });
      setBottomTab("scan-log");
      let targets = scanOpts.targets;
      if (targets.length === 0) {
        targets = ["local", ...(settings?.workers ?? []).filter((w) => w.enabled).map((w) => w.id)];
      }
      await api.startScan({
        ...scanOpts,
        targets,
        dir: draftScanDir || settings?.scanDir || undefined
      });
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

  const pickPython = async () => {
    try {
      const picked = await api.pickPythonExecutable(draftPythonPath || settings?.pythonPathEffective || "");
      setDraftPythonPath(picked);
    } catch (e) {
      const msg = String(e);
      if (!msg.includes("cancel") && !msg.includes("nenhum")) setError(msg);
    }
  };

  const saveSettings = async () => {
    setSettingsSaving(true);
    setError("");
    try {
      await api.saveSettings({
        dataDir: draftDataDir,
        scanDir: draftScanDir,
        pythonPath: draftPythonPath,
        notifyEnvFound: draftNotifyEnvFound,
        notifyScriptOk: draftNotifyScriptOk,
        soundEnvFound: draftSoundEnvFound,
        soundScriptOk: draftSoundScriptOk,
        deployRepoUrl: draftDeployRepoUrl,
        deployRepoRef: draftDeployRepoRef,
        deployRepoToken: draftDeployRepoToken,
        deployRepoMethod: draftDeployRepoMethod,
        hubEnabled: draftHubEnabled,
        workers: draftWorkers.map(({ testResult: _t, testing: _x, ...w }) => w)
      });
      await loadSettings();
      await loadFindings();
      await loadStats();
    } catch (e) {
      setError(String(e));
    } finally {
      setSettingsSaving(false);
    }
  };

  const pickKeyForWorker = async (index: number) => {
    try {
      const current = draftWorkers[index]?.keyPath ?? "";
      const picked = await api.pickKeyFile("Chave SSH (PEM ou PPK)", current);
      setDraftWorkers((prev) => prev.map((w, i) => (i === index ? { ...w, keyPath: picked } : w)));
    } catch (e) {
      const msg = String(e);
      if (!msg.includes("cancel") && !msg.includes("nenhum")) setError(msg);
    }
  };

  const testWorker = async (index: number) => {
    const worker = draftWorkers[index];
    if (!worker) return;
    setDraftWorkers((prev) => prev.map((w, i) => (i === index ? { ...w, testing: true, testResult: undefined } : w)));
    try {
      const result = await api.testRemoteWorker(worker);
      setDraftWorkers((prev) =>
        prev.map((w, i) => (i === index ? { ...w, testing: false, testResult: result } : w))
      );
    } catch (e) {
      setDraftWorkers((prev) =>
        prev.map((w, i) =>
          i === index ? { ...w, testing: false, testResult: { ok: false, remoteVersion: "", error: String(e) } } : w
        )
      );
    }
  };

  const startBatch = async (opts: { findingOnly?: boolean; quick?: boolean; threads?: number }) => {
    setWorkbenchView("batch");
    setError("");
    setBatchLogDir("");
    setBatchLogLines([]);
    setBatchResults([]);
    setBatchProgress(null);
    const threads = opts.threads ?? 1;
    try {
      await api.startBatchCheck({
        findingId: opts.findingOnly && selectedId ? selectedId : 0,
        query: debouncedQuery,
        confidence,
        unopenedOnly: opts.findingOnly ? false : unopenedOnly,
        quick: opts.quick ?? false,
        untestedOnly: batchUntestedOnly && !batchForceRecheck,
        forceRecheck: batchForceRecheck,
        limit: opts.findingOnly ? 1 : 500,
        threads
      });
    } catch (e) {
      setError(String(e));
    }
  };

  // Progresso agregado vem do backend (fila central SQLite); workers só mostram slice da onda.

  const batchLabel = batchProgress
    ? `${batchProgress.domain} · ${batchProgress.scriptLabel} (${batchProgress.scriptIndex}/${batchProgress.scriptTotal})`
    : undefined;

  const scanStats = scanProgress
    ? scanProgress.running
      ? `${scanProgress.domainsScanned.toLocaleString()} sessão · ${scanProgress.domainsPending.toLocaleString()} fila · ${scanProgress.vulnsFound} vulns`
      : workerProgress.length > 1
        ? `${scanProgress.domainsScanned} dom · ${scanProgress.vulnsFound} vulns · ${workerProgress.filter((w) => w.running).length} workers`
        : `${scanProgress.domainsScanned} dom · ${scanProgress.vulnsFound} vulns`
    : undefined;

  const panelRunningId = runningScript?.scriptId;
  const scanRunning = !!scanProgress?.running;
  const selectedFinding = selectedId ? findings.find((f) => f.id === selectedId) : undefined;

  const listProps = {
    query,
    onQueryChange: setQuery,
    confidence,
    onConfidenceChange: setConfidence,
    unopenedOnly,
    onUnopenedOnlyChange: setUnopenedOnly,
    unopenedCount: findingsStats.unopened,
    checkerFilter,
    onCheckerFilterChange: setCheckerFilter,
    findings: displayedFindings,
    findingIdsForCounts: findings.map((f) => f.id),
    selectedId,
    onSelect: setSelectedId,
    onOpen: (id: number) => void openFinding(id),
    checkerOverview,
    runningScript
  };

  const mainContent = (() => {
    if (workbenchView === "batch") {
      return (
        <div className="flex min-h-0 min-w-0 flex-1 flex-col">
          <BatchPanel
          batchThreads={batchThreads}
          onBatchThreadsChange={setBatchThreads}
          batchUntestedOnly={batchUntestedOnly}
          onBatchUntestedOnlyChange={(v) => {
            setBatchUntestedOnly(v);
            if (v) setBatchForceRecheck(false);
          }}
          batchForceRecheck={batchForceRecheck}
          onBatchForceRecheckChange={(v) => {
            setBatchForceRecheck(v);
            if (v) setBatchUntestedOnly(false);
          }}
          onTestAllFinding={() => void startBatch({ findingOnly: true })}
          onTestAllFiltered={() => void startBatch({ findingOnly: false })}
          onTestAllQuick={() => void startBatch({ findingOnly: false, quick: true })}
          onTestAllEnvs={() => void startBatch({ findingOnly: false, threads: batchThreads })}
          onCancelBatch={() => void api.cancelBatchCheck()}
          batchRunning={batchRunning}
          batchProgress={batchProgress}
          batchResults={batchResults}
          batchLogDir={batchLogDir}
          onOpenBatchLogs={() => void api.openBatchLogDir(batchLogDir)}
          batchLogLines={batchLogLines}
          onClearBatchLog={() => setBatchLogLines([])}
          onOpenFinding={(id) => void openFinding(id)}
          filterSummary={{
            count: displayedFindings.length,
            confidence,
            query: debouncedQuery,
            unopenedOnly,
            checkerFilter
          }}
          hasSelectedFinding={!!selectedId}
          runningScript={!!panelRunningId}
          />
        </div>
      );
    }

    if (workbenchView === "settings") {
      return (
        <div className="flex min-h-0 min-w-0 flex-1 flex-col">
          <SettingsView
            settings={settings}
            draftDataDir={draftDataDir}
            draftScanDir={draftScanDir}
            draftPythonPath={draftPythonPath}
            draftNotifyEnvFound={draftNotifyEnvFound}
            draftNotifyScriptOk={draftNotifyScriptOk}
            draftSoundEnvFound={draftSoundEnvFound}
            draftSoundScriptOk={draftSoundScriptOk}
            draftHubEnabled={draftHubEnabled}
            draftWorkers={draftWorkers}
            draftDeployRepoUrl={draftDeployRepoUrl}
            draftDeployRepoRef={draftDeployRepoRef}
            draftDeployRepoToken={draftDeployRepoToken}
            draftDeployRepoMethod={draftDeployRepoMethod}
            deployRepoHasToken={settings?.deployRepoHasToken ?? false}
            onDraftWorkersChange={setDraftWorkers}
            onDeployRepoChange={(patch) => {
              if (patch.url !== undefined) setDraftDeployRepoUrl(patch.url);
              if (patch.ref !== undefined) setDraftDeployRepoRef(patch.ref);
              if (patch.method !== undefined) setDraftDeployRepoMethod(patch.method);
              if (patch.token !== undefined) setDraftDeployRepoToken(patch.token);
            }}
            onDraftHubEnabledChange={setDraftHubEnabled}
            onPickKey={(i) => void pickKeyForWorker(i)}
            onTestWorker={(i) => void testWorker(i)}
            onDraftDataDirChange={setDraftDataDir}
            onDraftScanDirChange={setDraftScanDir}
            onDraftPythonPathChange={setDraftPythonPath}
            onDraftNotifyEnvFoundChange={setDraftNotifyEnvFound}
            onDraftNotifyScriptOkChange={setDraftNotifyScriptOk}
            onDraftSoundEnvFoundChange={setDraftSoundEnvFound}
            onDraftSoundScriptOkChange={setDraftSoundScriptOk}
            onPickDataDir={() => void pickDataDir()}
            onPickScanDir={() => void pickScanDir()}
            onPickPython={() => void pickPython()}
            onSave={() => void saveSettings()}
            onOpenDataDir={() => void api.openDataDirectory()}
            onOpenScanDir={() => void api.openScanDirectory()}
            saving={settingsSaving}
          />
        </div>
      );
    }

    if (workbenchView === "findings") {
      return (
        <div className="flex min-h-0 min-w-0 flex-1">
          <div className="flex min-w-0 flex-1 flex-col">
            <FindingsSidebar {...listProps} />
          </div>
          <aside className="w-[300px] shrink-0">
            <ScanPanel
              scanOpts={scanOpts}
              onScanOptsChange={setScanOpts}
              onStartScan={() => void startScan()}
              onCancelScan={() => void api.cancelScan()}
              scanRunning={scanRunning}
              scanProgress={scanProgress}
              workerProgress={workerProgress}
              workers={settings?.workers ?? []}
              scanDir={draftScanDir || settings?.scanDir}
            />
          </aside>
        </div>
      );
    }

    return null;
  })();

  const showBottomPanel =
    workbenchView === "findings" && (scanRunning || scanLogLines.length > 0 || batchRunning);
  const bottomTabs: BottomTab[] = batchRunning ? FINDINGS_BOTTOM_TABS : ["scan-log"];

  return (
    <>
      <WorkbenchLayout
        view={workbenchView}
        onViewChange={setWorkbenchView}
        batchActive={batchRunning}
        main={mainContent}
        terminal={
          showBottomPanel ? (
            <BottomPanel
              tabs={bottomTabs}
              tab={bottomTab}
              onTabChange={setBottomTab}
              outputLines={[]}
              batchLogLines={batchLogLines}
              scanLogLines={scanLogLines}
              scanLogFilter={scanLogFilter}
              onScanLogFilterChange={setScanLogFilter}
              scanWorkers={workerProgress}
              scanRunning={scanRunning}
              onCancelScan={() => void api.cancelScan()}
              onRestartScan={() => void startScan()}
              onClearOutput={() => {}}
              onClearBatchLog={() => setBatchLogLines([])}
              onClearScanLog={() => setScanLogLines([])}
              batchLogDir={batchLogDir}
              onOpenBatchLogs={() => void api.openBatchLogDir(batchLogDir)}
              defaultHeight={280}
            />
          ) : null
        }
        statusBar={
          <StatusBar
            findingLabel={
              selectedFinding ? `${selectedFinding.domain}${selectedFinding.path}` : undefined
            }
            findingsCount={displayedFindings.length}
            unopenedCount={findingsStats.unopened}
            unopenedFilter={unopenedOnly}
            checkerFilter={checkerFilter}
            scanRunning={scanRunning || batchRunning}
            scanStats={batchRunning || batchProgress ? batchLabel : scanStats}
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
        onSelectFinding={(id) => void openFinding(id)}
        onStartScan={() => void startScan()}
        onTestAllFinding={() => void startBatch({ findingOnly: true })}
        onTestAllFiltered={() => void startBatch({ findingOnly: false })}
        onViewChange={setWorkbenchView}
      />
    </>
  );
}
