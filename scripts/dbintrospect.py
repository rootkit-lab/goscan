#!/usr/bin/env python3
"""Introspecção partilhada para checkers DB — linhas, tabelas, nomes sensíveis."""

from __future__ import annotations

import re
from dataclasses import dataclass, field

from envutil import fmt_bytes

# Padrões por categoria (substring match, case-insensitive)
SENSITIVE_CATEGORIES: dict[str, tuple[str, ...]] = {
    "auth": ("user", "users", "login", "logins", "password", "passwd", "credential", "credentials", "oauth", "session", "sessions", "token", "tokens", "api_key", "apikey", "secret", "secrets", "auth", "account", "accounts"),
    "payment": ("payment", "payments", "pay", "payout", "transaction", "transactions", "invoice", "invoices", "billing", "wallet", "wallets", "stripe", "paypal", "order", "orders", "cart", "checkout"),
    "pii": ("customer", "customers", "client", "clients", "member", "members", "employee", "employees", "patient", "patients", "contact", "contacts", "profile", "profiles", "personal", "pii", "email", "emails", "phone", "phones"),
    "financial": ("bank", "banks", "card", "cards", "credit", "debit", "balance", "salary", "payroll", "tax", "cpf", "cnpj", "ssn", "iban", "pix"),
    "admin": ("admin", "admins", "administrator", "root", "superuser", "staff", "moderator"),
}

SENSITIVE_DB_EXTRA = ("prod", "production", "live", "payment", "pay", "crm", "erp", "billing", "customer")


@dataclass
class SensitiveHit:
    kind: str  # db | table | collection
    parent: str
    name: str
    rows: int
    tags: list[str] = field(default_factory=list)

    def label(self) -> str:
        if self.parent and self.parent != self.name:
            return f"{self.parent}.{self.name}"
        return self.name

    def short(self, max_rows: bool = True) -> str:
        base = self.label()
        if max_rows and self.rows > 0:
            return f"{base}({self.rows:,})"
        return base


def _tokenize(name: str) -> list[str]:
    parts = re.split(r"[\s._\-/]+", name.lower())
    return [p for p in parts if p]


def classify_sensitive(name: str, *, is_db: bool = False) -> list[str]:
    tags: list[str] = []
    lower = name.lower()
    tokens = set(_tokenize(name))
    for tag, words in SENSITIVE_CATEGORIES.items():
        for w in words:
            if w in tokens or w in lower:
                tags.append(tag)
                break
    if is_db:
        for w in SENSITIVE_DB_EXTRA:
            if w in tokens or w in lower:
                if "db" not in tags:
                    tags.append("db")
                break
    return sorted(set(tags))


def is_sensitive(name: str, *, is_db: bool = False) -> bool:
    return bool(classify_sensitive(name, is_db=is_db))


def format_sensitive_section(hits: list[SensitiveHit], limit: int = 12) -> list[str]:
    if not hits:
        return []
    lines = ["Sensível (por nome):"]
    for hit in hits[:limit]:
        tag_str = ",".join(hit.tags) if hit.tags else "?"
        rows = f" ~{hit.rows:,} rows" if hit.rows > 0 else ""
        lines.append(f"  ⚠ [{tag_str}] {hit.label()}{rows}")
    if len(hits) > limit:
        lines.append(f"  … +{len(hits) - limit} mais")
    return lines


def build_summary(
    db_count: int,
    total_rows: int,
    total_size: int | None,
    hits: list[SensitiveHit],
    *,
    unit: str = "rows",
) -> str:
    parts = [f"{db_count} DBs"]
    if total_size is not None and total_size > 0:
        parts.append(f"~{fmt_bytes(total_size)}")
    if total_rows > 0:
        parts.append(f"~{total_rows:,} {unit}")
    if hits:
        top = ", ".join(h.short() for h in hits[:5])
        parts.append(f"sensível: {top}")
    return " · ".join(parts)


def merge_hits(*groups: list[SensitiveHit]) -> list[SensitiveHit]:
    out: list[SensitiveHit] = []
    seen: set[str] = set()
    for group in groups:
        for h in group:
            key = f"{h.kind}:{h.parent}:{h.name}"
            if key in seen:
                continue
            seen.add(key)
            out.append(h)
    out.sort(key=lambda h: (-h.rows, h.label()))
    return out
