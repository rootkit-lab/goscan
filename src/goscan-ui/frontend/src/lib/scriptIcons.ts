import type { LucideIcon } from "lucide-react";
import {
  Bot,
  Cloud,
  CreditCard,
  Database,
  Flame,
  HardDrive,
  Mail,
  MessageSquare,
  Radio,
  Server,
  ShoppingBag,
  Sparkles
} from "lucide-react";

export type CheckerStatus = "ok" | "fail" | "skip" | "pending" | "running";

const ICONS: Record<string, LucideIcon> = {
  "chk-smtp": Mail,
  "chk-sendgrid": Mail,
  "chk-mailgun": Mail,
  "chk-mysql": Database,
  "chk-postgres": Database,
  "chk-mongodb": Database,
  "chk-redis": HardDrive,
  "chk-memcached": HardDrive,
  "chk-aws": Cloud,
  "chk-pusher": Radio,
  "chk-stripe": CreditCard,
  "chk-paystack": CreditCard,
  "chk-paypal": CreditCard,
  "chk-razorpay": CreditCard,
  "chk-paddle": CreditCard,
  "chk-flutterwave": CreditCard,
  "chk-twilio": MessageSquare,
  "chk-nexmo": MessageSquare,
  "chk-telegram": MessageSquare,
  "chk-fcm": Flame,
  "chk-firebase": Flame,
  "chk-hubspot": Server,
  "chk-shopify": ShoppingBag,
  "chk-supabase": Server,
  "chk-sentry": Server,
  "chk-openai": Bot,
  "chk-gemini": Bot,
  "chk-groq": Bot,
  "chk-claud": Bot,
  "chk-xai": Bot,
  "chk-mistral": Bot,
  "chk-deepseek": Bot,
  "chk-perplexity": Bot,
  "chk-openrouter": Bot,
  "chk-together": Bot,
  "chk-cohere": Bot,
  "chk-replicate": Sparkles,
  "chk-huggingface": Sparkles
};

export function scriptIcon(scriptId: string): LucideIcon {
  return ICONS[scriptId] ?? Server;
}

export function scriptShortLabel(label: string): string {
  if (label.length <= 8) return label;
  return label.slice(0, 7) + "…";
}

export function statusTitle(status: CheckerStatus, summary?: string): string {
  const base = { ok: "OK", fail: "Falhou", skip: "Skip", pending: "Por testar", running: "A correr…" }[status];
  return summary ? `${base}: ${summary}` : base;
}
