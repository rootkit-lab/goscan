import type { ScriptCheckerStatusDTO } from "@/lib/api";

export type CheckerResultFilter =
  | ""
  | "any_ok"
  | "email_ok"
  | "db_ok"
  | "cache_ok"
  | "llm_ok"
  | "pay_ok"
  | "cloud_ok"
  | "msg_ok"
  | "crm_ok";

type FilterDef = {
  value: CheckerResultFilter;
  label: string;
  title: string;
};

export const CHECKER_FILTER_OPTIONS: FilterDef[] = [
  { value: "any_ok", label: "Válidos", title: "Qualquer checker OK" },
  { value: "email_ok", label: "Email ✓", title: "SMTP, SendGrid ou Mailgun OK" },
  { value: "db_ok", label: "DB ✓", title: "MySQL, Postgres ou MongoDB OK" },
  { value: "cache_ok", label: "Cache ✓", title: "Redis ou Memcached OK" },
  { value: "llm_ok", label: "LLM ✓", title: "Qualquer API de IA OK" },
  { value: "pay_ok", label: "Pay ✓", title: "Stripe, PayPal, Razorpay, Paystack, Paddle ou Flutterwave OK" },
  { value: "cloud_ok", label: "Cloud ✓", title: "AWS, Firebase, Supabase ou Sentry OK" },
  { value: "msg_ok", label: "Msg ✓", title: "Twilio, Nexmo, Telegram, FCM ou Pusher OK" },
  { value: "crm_ok", label: "CRM ✓", title: "HubSpot ou Shopify OK" }
];

const CHECKER_GROUPS: Record<Exclude<CheckerResultFilter, "">, readonly string[]> = {
  any_ok: [],
  email_ok: ["chk-smtp", "chk-sendgrid", "chk-mailgun"],
  db_ok: ["chk-mysql", "chk-postgres", "chk-mongodb"],
  cache_ok: ["chk-redis", "chk-memcached"],
  llm_ok: [
    "chk-openai",
    "chk-gemini",
    "chk-groq",
    "chk-claud",
    "chk-xai",
    "chk-mistral",
    "chk-deepseek",
    "chk-perplexity",
    "chk-openrouter",
    "chk-together",
    "chk-cohere",
    "chk-replicate",
    "chk-huggingface"
  ],
  pay_ok: [
    "chk-stripe",
    "chk-paypal",
    "chk-razorpay",
    "chk-paystack",
    "chk-paddle",
    "chk-flutterwave"
  ],
  cloud_ok: ["chk-aws", "chk-firebase", "chk-supabase", "chk-sentry"],
  msg_ok: ["chk-twilio", "chk-nexmo", "chk-telegram", "chk-fcm", "chk-pusher"],
  crm_ok: ["chk-hubspot", "chk-shopify"]
};

const GROUP_LOOKUP = new Map<string, CheckerResultFilter>();
for (const [filter, ids] of Object.entries(CHECKER_GROUPS) as [CheckerResultFilter, readonly string[]][]) {
  if (filter === "any_ok") continue;
  for (const id of ids) {
    GROUP_LOOKUP.set(id, filter);
  }
}

export function checkerFilterLabel(filter: CheckerResultFilter): string {
  return CHECKER_FILTER_OPTIONS.find((o) => o.value === filter)?.label ?? filter;
}

export function scriptsForFilter(filter: CheckerResultFilter): Set<string> | null {
  if (!filter || filter === "any_ok") return null;
  return new Set(CHECKER_GROUPS[filter]);
}

export function findingMatchesCheckerFilter(
  findingId: number,
  filter: CheckerResultFilter,
  overview: Record<number, ScriptCheckerStatusDTO[]>
): boolean {
  if (!filter) return true;
  const scripts = overview[findingId] ?? [];
  if (filter === "any_ok") return scripts.some((s) => s.status === "ok");
  const group = scriptsForFilter(filter);
  if (!group) return true;
  return scripts.some((s) => group.has(s.scriptId) && s.status === "ok");
}

export function countCheckerFilter(
  findingIds: number[],
  filter: CheckerResultFilter,
  overview: Record<number, ScriptCheckerStatusDTO[]>
): number {
  if (!filter) return 0;
  return findingIds.filter((id) => findingMatchesCheckerFilter(id, filter, overview)).length;
}

export function visibleCheckerScripts(
  scripts: ScriptCheckerStatusDTO[],
  filter: CheckerResultFilter
): ScriptCheckerStatusDTO[] {
  if (!filter) return scripts;
  const okOnly = scripts.filter((s) => s.status === "ok");
  const group = scriptsForFilter(filter);
  if (filter === "any_ok" || !group) return okOnly;
  return okOnly.filter((s) => group.has(s.scriptId));
}

export function checkerGroupForScript(scriptId: string): CheckerResultFilter | null {
  return GROUP_LOOKUP.get(scriptId) ?? null;
}
