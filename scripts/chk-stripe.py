#!/usr/bin/env python3
"""Valida chave Stripe — saldo, conta, payouts e transferências."""

from __future__ import annotations

import sys
from pathlib import Path

import requests

from envutil import (
    DEFAULT_HTTP_TIMEOUT,
    DEFAULT_OPERATION_TIMEOUT,
    env_arg_parser,
    format_network_error,
    is_batch_mode,
    is_interactive,
    load_env_keys,
    log_step,
    main_missing,
    pick_key,
    print_summary,
    prompt_optional,
    prompt_required,
    run_with_timeout,
    select_from_list,
)

STRIPE_API = "https://api.stripe.com/v1"


def stripe_key(env: dict[str, str]) -> str:
    key = pick_key(env, ["STRIPE_SECRET", "STRIPE_SECRET_KEY", "STRIPE_APIKEY"])
    if not key:
        main_missing("STRIPE_SECRET")
    return key


def key_mode(secret: str) -> str:
    if secret.startswith("sk_live"):
        return "LIVE"
    if secret.startswith("sk_test"):
        return "test"
    if secret.startswith("rk_live"):
        return "restricted LIVE"
    if secret.startswith("rk_test"):
        return "restricted test"
    return "?"


def fmt_money(amount_cents: int, currency: str) -> str:
    return f"{amount_cents / 100:.2f} {currency.upper()}"


def stripe_request(secret: str, method: str, path: str, *, params: dict | None = None, data: dict | None = None) -> dict:
    url = path if path.startswith("http") else f"{STRIPE_API}{path if path.startswith('/') else '/' + path}"
    r = requests.request(
        method,
        url,
        auth=(secret, ""),
        params=params,
        data=data,
        timeout=DEFAULT_HTTP_TIMEOUT,
    )
    if r.status_code == 401:
        raise PermissionError("Chave inválida (401)")
    if r.status_code >= 400:
        try:
            err = r.json().get("error", {})
            msg = err.get("message") or r.text[:300]
        except Exception:
            msg = r.text[:300]
        raise RuntimeError(f"HTTP {r.status_code}: {msg}")
    return r.json()


def stripe_call(secret: str, method: str, path: str, **kwargs) -> dict:
    return run_with_timeout(
        lambda: stripe_request(secret, method, path, **kwargs),
        DEFAULT_OPERATION_TIMEOUT,
        f"Stripe {path}",
    )


def parse_balance(data: dict) -> list[dict]:
    rows: list[dict] = []
    for bucket in ("available", "pending"):
        for item in data.get(bucket, []):
            rows.append(
                {
                    "bucket": bucket,
                    "amount": int(item.get("amount", 0)),
                    "currency": str(item.get("currency", "?")).lower(),
                }
            )
    return rows


def format_balance(data: dict) -> str:
    rows = parse_balance(data)
    if not rows:
        return "OK — conta activa (saldo zero)"
    return "\n".join(f"  {r['bucket']}: {fmt_money(r['amount'], r['currency'])}" for r in rows)


def fetch_balance(secret: str) -> tuple[bool, str]:
    try:
        data = stripe_call(secret, "GET", "/balance")
    except PermissionError as exc:
        return False, str(exc)
    except Exception as exc:
        return False, str(exc)
    return True, format_balance(data)


def show_account(secret: str) -> None:
    log_step("A consultar conta…")
    acc = stripe_call(secret, "GET", "/account")
    print(f"  ID:       {acc.get('id')}", flush=True)
    print(f"  Email:    {acc.get('email') or '—'}", flush=True)
    print(f"  País:     {acc.get('country') or '—'}", flush=True)
    print(f"  Tipo:     {acc.get('type') or 'standard'}", flush=True)
    print(f"  Payouts:  {acc.get('payouts_enabled')}", flush=True)
    print(f"  Cobranças:{acc.get('charges_enabled')}", flush=True)


def show_charges(secret: str) -> None:
    log_step("A listar cobranças…")
    data = stripe_call(secret, "GET", "/charges", params={"limit": 10})
    items = data.get("data", [])
    if not items:
        print("  Nenhuma cobrança recente.", flush=True)
        return
    for ch in items:
        amt = fmt_money(int(ch.get("amount", 0)), str(ch.get("currency", "?")))
        status = ch.get("status", "?")
        created = ch.get("created", "?")
        print(f"  {ch.get('id')} — {amt} — {status} — ts {created}", flush=True)


def show_payouts(secret: str) -> None:
    log_step("A listar payouts…")
    data = stripe_call(secret, "GET", "/payouts", params={"limit": 10})
    items = data.get("data", [])
    if not items:
        print("  Nenhum payout recente.", flush=True)
        return
    for p in items:
        amt = fmt_money(int(p.get("amount", 0)), str(p.get("currency", "?")))
        print(f"  {p.get('id')} — {amt} — {p.get('status')} — chegada {p.get('arrival_date')}", flush=True)


def list_bank_accounts(secret: str) -> list[dict]:
    log_step("A listar contas bancárias…")
    data = stripe_call(secret, "GET", "/account/external_accounts", params={"object": "bank_account", "limit": 10})
    return data.get("data", [])


def show_bank_accounts(secret: str) -> None:
    banks = list_bank_accounts(secret)
    if not banks:
        print("  Nenhuma conta bancária ligada (ou sem permissão).", flush=True)
        return
    for b in banks:
        last4 = b.get("last4", "????")
        bank = b.get("bank_name") or b.get("account_holder_name") or "?"
        cur = b.get("currency", "?")
        default = " (default)" if b.get("default_for_currency") else ""
        print(f"  {b.get('id')} — {bank} ****{last4} — {cur}{default}", flush=True)


def list_connected_accounts(secret: str) -> list[dict]:
    log_step("A listar contas Connect…")
    data = stripe_call(secret, "GET", "/accounts", params={"limit": 20})
    return data.get("data", [])


def show_connected_accounts(secret: str) -> None:
    accounts = list_connected_accounts(secret)
    if not accounts:
        print("  Nenhuma conta Connect (ou chave não é de plataforma).", flush=True)
        return
    for acc in accounts:
        email = acc.get("email") or "—"
        print(f"  {acc.get('id')} — {email} — payouts={acc.get('payouts_enabled')}", flush=True)


def pick_available_currency(secret: str) -> tuple[int, str] | None:
    data = stripe_call(secret, "GET", "/balance")
    available = [r for r in parse_balance(data) if r["bucket"] == "available" and r["amount"] > 0]
    if not available:
        print("Saldo disponível zero — não há fundos para mover.", flush=True)
        return None
    if len(available) == 1:
        row = available[0]
        return row["amount"], row["currency"]
    labels = [f"{fmt_money(r['amount'], r['currency'])} disponível" for r in available]
    picked = select_from_list("Escolha a moeda", labels)
    if not picked:
        return None
    idx = labels.index(picked)
    row = available[idx]
    return row["amount"], row["currency"]


def parse_amount(raw: str, currency: str, max_cents: int) -> int | None:
    raw = raw.strip().replace(",", ".")
    try:
        value = int(round(float(raw) * 100))
    except ValueError:
        print("Valor inválido — use decimal, ex: 10.50", flush=True)
        return None
    if value <= 0:
        print("Valor deve ser positivo.", flush=True)
        return None
    if value > max_cents:
        print(f"Máximo disponível: {fmt_money(max_cents, currency)}", flush=True)
        return None
    return value


def confirm_live_action(mode: str, action: str) -> bool:
    if mode == "LIVE":
        print(f"\n⚠  Chave LIVE — {action} move dinheiro real.", flush=True)
    confirm = prompt_required(f"Escreva SIM para confirmar {action}").strip().upper()
    return confirm == "SIM"


def create_payout(secret: str, mode: str) -> None:
    picked = pick_available_currency(secret)
    if not picked:
        return
    max_cents, currency = picked
    print(f"\nSaldo disponível: {fmt_money(max_cents, currency)}", flush=True)
    show_bank_accounts(secret)

    raw = prompt_optional(f"Valor payout em {currency.upper()} (decimal, ex: 10.50)", "")
    if not raw:
        return
    amount = parse_amount(raw, currency, max_cents)
    if amount is None:
        return

    if not confirm_live_action(mode, f"payout de {fmt_money(amount, currency)} para o banco"):
        print("Cancelado.", flush=True)
        return

    log_step("A criar payout…")
    payout = stripe_call(
        secret,
        "POST",
        "/payouts",
        data={"amount": str(amount), "currency": currency},
    )
    print(f"OK — payout {payout.get('id')} — {fmt_money(amount, currency)} — status {payout.get('status')}", flush=True)


def create_connect_transfer(secret: str, mode: str) -> None:
    accounts = list_connected_accounts(secret)
    if not accounts:
        print("Sem contas Connect para transferir.", flush=True)
        return

    labels = [f"{a.get('id')} — {a.get('email') or 'sem email'}" for a in accounts]
    picked = select_from_list("Conta destino (Connect)", labels)
    if not picked:
        return
    destination = accounts[labels.index(picked)]["id"]

    picked_bal = pick_available_currency(secret)
    if not picked_bal:
        return
    max_cents, currency = picked_bal
    print(f"\nSaldo plataforma: {fmt_money(max_cents, currency)}", flush=True)

    raw = prompt_optional(f"Valor transferência em {currency.upper()} (decimal, ex: 10.50)", "")
    if not raw:
        return
    amount = parse_amount(raw, currency, max_cents)
    if amount is None:
        return

    if not confirm_live_action(mode, f"transferência Connect de {fmt_money(amount, currency)} → {destination}"):
        print("Cancelado.", flush=True)
        return

    log_step("A criar transferência Connect…")
    transfer = stripe_call(
        secret,
        "POST",
        "/transfers",
        data={"amount": str(amount), "currency": currency, "destination": destination},
    )
    print(
        f"OK — transfer {transfer.get('id')} — {fmt_money(amount, currency)} → {destination} — {transfer.get('status')}",
        flush=True,
    )


MENU = [
    ("1", "Ver saldo"),
    ("2", "Dados da conta"),
    ("3", "Cobranças recentes"),
    ("4", "Payouts recentes"),
    ("5", "Contas bancárias"),
    ("6", "Contas Connect"),
    ("7", "Criar payout (saldo → banco)"),
    ("8", "Transferência Connect (plataforma → subconta)"),
    ("0", "Sair"),
]


def run_menu(secret: str) -> int:
    mode = key_mode(secret)
    print(f"\nModo: {mode}", flush=True)
    if mode == "LIVE":
        print("⚠  Operações 7 e 8 movem fundos reais.", flush=True)

    while True:
        print("\n--- Stripe ---", flush=True)
        for code, label in MENU:
            print(f"  {code}. {label}", flush=True)
        choice = prompt_optional("Opção", "").strip()
        if choice in ("0", "sair", "exit", "q"):
            return 0
        try:
            if choice == "1":
                log_step("A consultar saldo…")
                ok, msg = fetch_balance(secret)
                print(msg, flush=True)
            elif choice == "2":
                show_account(secret)
            elif choice == "3":
                show_charges(secret)
            elif choice == "4":
                show_payouts(secret)
            elif choice == "5":
                show_bank_accounts(secret)
            elif choice == "6":
                show_connected_accounts(secret)
            elif choice == "7":
                create_payout(secret, mode)
            elif choice == "8":
                create_connect_transfer(secret, mode)
            else:
                print("Opção inválida.", flush=True)
        except Exception as exc:
            print(format_network_error(exc), flush=True)
    return 0


def run_interactive(env: dict[str, str]) -> int:
    secret = stripe_key(env)
    print(f"Stripe — chave …{secret[-6:]}", flush=True)
    log_step("A validar chave…")
    try:
        ok, msg = fetch_balance(secret)
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1
    if not ok:
        print(msg, flush=True)
        return 1
    print("Chave válida.", flush=True)
    print(msg, flush=True)
    return run_menu(secret)


def run_batch(env: dict[str, str]) -> int:
    secret = stripe_key(env)
    log_step("Stripe saldo…")
    try:
        ok, msg = run_with_timeout(lambda: fetch_balance(secret), DEFAULT_OPERATION_TIMEOUT, "Stripe")
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1
    if not ok:
        print(msg, flush=True)
        return 1
    line = msg.split("\n")[0] if msg else "Stripe OK"
    print_summary(line)
    return 0


def main() -> None:
    args = env_arg_parser("Stripe checker").parse_args()
    env = load_env_keys(Path(args.env))
    if is_batch_mode(args):
        sys.exit(run_batch(env))
    if is_interactive():
        sys.exit(run_interactive(env))
    try:
        ok, msg = fetch_balance(stripe_key(env))
        print(msg, flush=True)
        sys.exit(0 if ok else 1)
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        sys.exit(1)


if __name__ == "__main__":
    main()
