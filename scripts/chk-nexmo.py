#!/usr/bin/env python3
"""Valida credenciais Nexmo / Vonage."""

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


def nexmo_config(env: dict[str, str]) -> tuple[str, str]:
    key = pick_key(env, ["NEXMO_KEY", "VONAGE_API_KEY", "VONAGE_KEY"])
    secret = pick_key(env, ["NEXMO_SECRET", "VONAGE_API_SECRET", "VONAGE_SECRET"])
    if not key or not secret:
        main_missing("NEXMO_KEY + NEXMO_SECRET")
    return key, secret


def fetch_balance(key: str, secret: str) -> tuple[bool, str]:
    r = requests.get(
        "https://rest.nexmo.com/account/get-balance",
        params={"api_key": key, "api_secret": secret},
        timeout=DEFAULT_HTTP_TIMEOUT,
    )
    if r.status_code != 200:
        return False, f"HTTP {r.status_code}: {r.text[:200]}"
    try:
        balance = float(r.text.strip())
    except ValueError:
        return False, f"Resposta inesperada: {r.text[:120]}"
    return True, f"Saldo EUR {balance:.2f}"


def run_batch(env: dict[str, str]) -> int:
    key, secret = nexmo_config(env)
    log_step("Nexmo/Vonage…")
    try:
        ok, msg = run_with_timeout(lambda: fetch_balance(key, secret), DEFAULT_OPERATION_TIMEOUT, "Nexmo")
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1
    if not ok:
        print(msg, flush=True)
        return 1
    print_summary(msg)
    return 0


def main() -> None:
    args = env_arg_parser("Nexmo/Vonage checker").parse_args()
    env = load_env_keys(Path(args.env))
    if is_batch_mode(args):
        sys.exit(run_batch(env))
    if is_interactive():
        print(f"Nexmo — key …{nexmo_config(env)[0][-4:]}", flush=True)
    sys.exit(run_batch(env))


if __name__ == "__main__":
    main()
