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
  hasCredentials: boolean;
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
};

export type ScanProgressDTO = {
  domainsScanned: number;
  vulnsFound: number;
  domainsNew: number;
  domainsPending: number;
  running: boolean;
};

const S = "main.App";

export const api = {
  searchFindings: (query: string, confidence: string, limit: number) =>
    Call.ByName(`${S}.SearchFindings`, query, confidence, limit) as Promise<FindingDTO[]>,
  getFinding: (id: number) => Call.ByName(`${S}.GetFinding`, id) as Promise<FindingDetailDTO>,
  listScripts: () => Call.ByName(`${S}.ListScripts`) as Promise<ScriptDTO[]>,
  compatibleScripts: (findingId: number) =>
    Call.ByName(`${S}.CompatibleScripts`, findingId) as Promise<ScriptDTO[]>,
  runScript: (scriptId: string, findingId: number) =>
    Call.ByName(`${S}.RunScript`, scriptId, findingId) as Promise<void>,
  cancelScript: () => Call.ByName(`${S}.CancelScript`) as Promise<void>,
  terminalInput: (data: string) => Call.ByName(`${S}.TerminalInput`, data) as Promise<void>,
  terminalResize: (cols: number, rows: number) => Call.ByName(`${S}.TerminalResize`, cols, rows) as Promise<void>,
  startScan: (opts: ScanOptsDTO) => Call.ByName(`${S}.StartScan`, opts) as Promise<void>,
  cancelScan: () => Call.ByName(`${S}.CancelScan`) as Promise<void>
};

export { Events };
