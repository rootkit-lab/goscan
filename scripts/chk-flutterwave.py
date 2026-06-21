#!/usr/bin/env python3
"""Valida chave Flutterwave."""

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


def flutterwave_secret(env: dict[str, str]) -> str:
    key = pick_key(env, ["FLW_SECRET_KEY", "FLW_SECRET", "FLUTTERWAVE_SECRET_KEY"])
    if not key:
        main_missing("FLW_SECRET_KEY")
    return key


def fetch_balance(secret: str) -> tuple[bool, str]:
    r = requests.get(
        "https://api.flutterwave.com/v3/balances",
        headers={"Authorization": f"Bearer {secret}"},
        timeout=DEFAULT_HTTP_TIMEOUT,
    )
    if r.status_code == 401:
        return False, "Chave inválida (401)"
    if r.status_code != 200:
        return False, f"HTTP {r.status_code}: {r.text[:200]}"
    rows = r.json().get("data", [])
    if not rows:
        return True, "API OK — sem saldos"
    parts = []
    for row in rows[:3]:
        parts.append(f"{row.get('currency', '?')} {row.get('available_balance', 0)}")
    return True, "Saldo: " + ", ".join(parts)


def run_batch(env: dict[str, str]) -> int:
    secret = flutterwave_secret(env)
    log_step("Flutterwave…")
    try:
        ok, msg = run_with_timeout(lambda: fetch_balance(secret), DEFAULT_OPERATION_TIMEOUT, "Flutterwave")
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1
    if not ok:
        print(msg, flush=True)
        return 1
    print_summary(msg)
    return 0


def main() -> None:
    args = env_arg_parser("Flutterwave checker").parse_args()
    env = load_env_keys(Path(args.env))
    if is_batch_mode(args):
        sys.exit(run_batch(env))
    if is_interactive():
        print(f"Flutterwave — …{flutterwave_secret(env)[-6:]}", flush=True)
    sys.exit(run_batch(env))


if __name__ == "__main__":
    main()
