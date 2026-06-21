#!/usr/bin/env python3
"""Valida chave Paystack."""

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
    run_with_timeout,
)


def paystack_secret(env: dict[str, str]) -> str:
    key = pick_key(env, ["PAYSTACK_SECRET_KEY", "PAYSTACK_SECRET", "PAYSTACK_SK"])
    if not key:
        main_missing("PAYSTACK_SECRET_KEY")
    return key


def fetch_balance(secret: str) -> tuple[bool, str]:
    r = requests.get(
        "https://api.paystack.co/balance",
        headers={"Authorization": f"Bearer {secret}"},
        timeout=DEFAULT_HTTP_TIMEOUT,
    )
    if r.status_code == 401:
        return False, "Chave inválida (401)"
    if r.status_code != 200:
        return False, f"HTTP {r.status_code}: {r.text[:200]}"
    data = r.json().get("data", [])
    if not data:
        return True, "API OK — saldo zero ou vazio"
    parts = []
    for row in data[:3]:
        parts.append(f"{row.get('currency', '?').upper()} {row.get('balance', 0) / 100:.2f}")
    return True, "Saldo: " + ", ".join(parts)


def run_batch(env: dict[str, str]) -> int:
    secret = paystack_secret(env)
    log_step("Paystack…")
    try:
        ok, msg = run_with_timeout(lambda: fetch_balance(secret), DEFAULT_OPERATION_TIMEOUT, "Paystack")
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1
    if not ok:
        print(msg, flush=True)
        return 1
    print_summary(msg)
    return 0


def main() -> None:
    args = env_arg_parser("Paystack checker").parse_args()
    env = load_env_keys(Path(args.env))
    if is_batch_mode(args):
        sys.exit(run_batch(env))
    if is_interactive():
        print(f"Paystack — …{paystack_secret(env)[-6:]}", flush=True)
    sys.exit(run_batch(env))


if __name__ == "__main__":
    main()
