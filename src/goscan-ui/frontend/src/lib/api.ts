import { Call, Events } from "@wailsio/runtime";

export type FindingDTO = {
  id: number;
  domain: string;
  path: string;
  url: string;
  confidence: string;
  filePath: string;
  scanRunId: string;
  foundAt: string;
  openedAt?: string;
  modifiedAt?: string;
  hasCredentials: boolean;
  isNew?: boolean;
};

export type FindingsStatsDTO = {
  total: number;
  unopened: number;
};

export type FindingDetailDTO = FindingDTO & {
  content: string;
  absPath: string;
};

export type ScriptDTO = {
  id: string;
  label: string;
  envKeys: string[];
  interactive?: boolean;
};

export type ScanOptsDTO = {
  dir?: string;
  threads?: number;
  pathWorkers?: number;
  fast?: boolean;
  rescan?: boolean;
  timeoutSec?: number;
  targets?: string[];
  deployRemote?: boolean;
};

export type ScanWorkerProgressDTO = {
  workerId: string;
  workerName: string;
  domainsScanned: number;
  vulnsFound: number;
  domainsTotal: number;
  status: string;
  error?: string;
  running: boolean;
  phasePercent?: number;
  phaseLabel?: string;
};

export type ScanProgressDTO = {
  domainsScanned: number;
  vulnsFound: number;
  domainsNew: number;
  domainsPending: number;
  wave?: number;
  waveBatchSize?: number;
  waveScanned?: number;
  sessionScanned?: number;
  running: boolean;
};

export type CheckerResultDTO = {
  findingId: number;
  scriptId: string;
  scriptLabel: string;
  status: "ok" | "fail" | "skip";
  exitCode: number;
  summary: string;
  testedAt: string;
};

export type ScriptCheckerStatusDTO = {
  scriptId: string;
  label: string;
  status: "ok" | "fail" | "skip" | "pending" | "running";
  summary: string;
  testedAt: string;
  exitCode: number;
  logPath?: string;
};

export type FindingCheckerOverviewDTO = {
  findingId: number;
  scripts: ScriptCheckerStatusDTO[];
};

export type BatchCheckOptsDTO = {
  findingId?: number;
  query?: string;
  confidence?: string;
  unopenedOnly?: boolean;
  scriptId?: string;
  quick?: boolean;
  untestedOnly?: boolean;
  forceRecheck?: boolean;
  limit?: number;
  threads?: number;
};

export type BatchProgressDTO = {
  findingIndex: number;
  findingTotal: number;
  findingId: number;
  domain: string;
  scriptIndex: number;
  scriptTotal: number;
  scriptId: string;
  scriptLabel: string;
  status: string;
  summary: string;
  exitCode: number;
  line: string;
  running: boolean;
  checkIndex: number;
  checkTotal: number;
  okCount: number;
  failCount: number;
  skipCount: number;
  threads: number;
};

export type BatchDoneDTO = {
  ok: number;
  fail: number;
  skip: number;
  total: number;
  secs: number;
  logDir?: string;
};

export type SettingsDTO = {
  mode: string;
  dataDir: string;
  scanDir: string;
  appRoot: string;
  defaultProdDataDir: string;
  pointsToDevRepo: boolean;
  needsSetup: boolean;
  version: string;
  pythonPath: string;
  pythonPathEffective: string;
  notifyEnvFound: boolean;
  notifyScriptOk: boolean;
  soundEnvFound: boolean;
  soundScriptOk: boolean;
  workers: RemoteWorkerDTO[];
  deployRepoUrl: string;
  deployRepoRef: string;
  deployRepoMethod: string;
  deployRepoHasToken: boolean;
  hubEnabled: boolean;
};

export type RemoteWorkerDTO = {
  id: string;
  name: string;
  host: string;
  port: number;
  user: string;
  authType: string;
  keyPath: string;
  execMode: string;
  apiPort: number;
  enabled: boolean;
  hasPassword: boolean;
  remoteVersion?: string;
};

export type RemoteWorkerSaveDTO = {
  id: string;
  name: string;
  host: string;
  port: number;
  user: string;
  authType: string;
  password: string;
  keyPath: string;
  keyPassphrase: string;
  execMode: string;
  apiPort: number;
  apiToken: string;
  enabled: boolean;
};

export type RemoteWorkerTestResultDTO = {
  ok: boolean;
  remoteVersion: string;
  error?: string;
};

export type AlertTestResultDTO = {
  soundOk: boolean;
  notifyOk: boolean;
  notifyAvailable: boolean;
  error?: string;
};

export type SettingsSaveDTO = {
  dataDir: string;
  scanDir: string;
  pythonPath: string;
  notifyEnvFound: boolean;
  notifyScriptOk: boolean;
  soundEnvFound: boolean;
  soundScriptOk: boolean;
  workers: RemoteWorkerSaveDTO[];
  deployRepoUrl: string;
  deployRepoRef: string;
  deployRepoToken: string;
  deployRepoMethod: string;
  hubEnabled: boolean;
};

const S = "main.App";

export const api = {
  searchFindings: (query: string, confidence: string, unopenedOnly: boolean, limit: number) =>
    Call.ByName(`${S}.SearchFindings`, query, confidence, unopenedOnly, limit) as Promise<FindingDTO[]>,
  findingsStats: () => Call.ByName(`${S}.FindingsStats`) as Promise<FindingsStatsDTO>,
  getFinding: (id: number) => Call.ByName(`${S}.GetFinding`, id) as Promise<FindingDetailDTO>,
  listScripts: () => Call.ByName(`${S}.ListScripts`) as Promise<ScriptDTO[]>,
  compatibleScripts: (findingId: number) =>
    Call.ByName(`${S}.CompatibleScripts`, findingId) as Promise<ScriptDTO[]>,
  checkerOverview: (findingIds: number[]) =>
    Call.ByName(`${S}.CheckerOverview`, findingIds) as Promise<FindingCheckerOverviewDTO[]>,
  listCheckerResults: (findingId: number) =>
    Call.ByName(`${S}.ListCheckerResults`, findingId) as Promise<CheckerResultDTO[]>,
  runScript: (scriptId: string, findingId: number) =>
    Call.ByName(`${S}.RunScript`, scriptId, findingId) as Promise<void>,
  cancelScript: () => Call.ByName(`${S}.CancelScript`) as Promise<void>,
  terminalInput: (data: string) => Call.ByName(`${S}.TerminalInput`, data) as Promise<void>,
  terminalResize: (cols: number, rows: number) => Call.ByName(`${S}.TerminalResize`, cols, rows) as Promise<void>,
  startScan: (opts: ScanOptsDTO) => Call.ByName(`${S}.StartScan`, opts) as Promise<void>,
  cancelScan: () => Call.ByName(`${S}.CancelScan`) as Promise<void>,
  startBatchCheck: (opts: BatchCheckOptsDTO) => Call.ByName(`${S}.StartBatchCheck`, opts) as Promise<void>,
  cancelBatchCheck: () => Call.ByName(`${S}.CancelBatchCheck`) as Promise<void>,
  openBatchLogDir: (dir?: string) => Call.ByName(`${S}.OpenBatchLogDir`, dir ?? "") as Promise<void>,
  getSettings: () => Call.ByName(`${S}.GetSettings`) as Promise<SettingsDTO>,
  pickDirectory: (title: string, current: string) =>
    Call.ByName(`${S}.PickDirectory`, title, current) as Promise<string>,
  pickPythonExecutable: (current: string) =>
    Call.ByName(`${S}.PickPythonExecutable`, current) as Promise<string>,
  pickKeyFile: (title: string, current: string) =>
    Call.ByName(`${S}.PickKeyFile`, title, current) as Promise<string>,
  testRemoteWorker: (worker: RemoteWorkerSaveDTO) =>
    Call.ByName(`${S}.TestRemoteWorker`, worker) as Promise<RemoteWorkerTestResultDTO>,
  saveSettings: (opts: SettingsSaveDTO) => Call.ByName(`${S}.SaveSettings`, opts) as Promise<void>,
  openDataDirectory: () => Call.ByName(`${S}.OpenDataDirectory`) as Promise<void>,
  openScanDirectory: () => Call.ByName(`${S}.OpenScanDirectory`) as Promise<void>,
  openEditorWindow: (findingId: number) =>
    Call.ByName(`${S}.OpenEditorWindow`, findingId) as Promise<void>,
  focusMainWindow: () => Call.ByName(`${S}.FocusMainWindow`) as Promise<void>,
  notifyAvailable: () => Call.ByName(`${S}.NotifyAvailable`) as Promise<boolean>,
  desktopNotify: (title: string, body: string) =>
    Call.ByName(`${S}.DesktopNotify`, title, body) as Promise<boolean>,
  playAlertSound: (kind: string) => Call.ByName(`${S}.PlayAlertSound`, kind) as Promise<void>,
  testEnvAlert: (notify: boolean, sound: boolean) =>
    Call.ByName(`${S}.TestEnvAlert`, notify, sound) as Promise<AlertTestResultDTO>
};

export { Events };
