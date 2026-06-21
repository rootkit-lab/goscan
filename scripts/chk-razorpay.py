#!/usr/bin/env python3
"""Valida credenciais Razorpay."""

from __future__ import annotations

import sys
from pathlib import Path

import requests

from envutil import (
    DEFAULT_HTTP_TIMEOUT,
    DEFAULT_OPERATION_TIMEOUT,
    env_arg_parser,
    format_network_error,
    is_interactive,
    load_env_keys,
    log_step,
    main_missing,
    pick_key,
    run_with_timeout,
)


def razorpay_config(env: dict[str, str]) -> tuple[str, str]:
    key_id = pick_key(env, ["RAZORPAY_KEY_ID", "RAZORPAY_KEY"])
    secret = pick_key(env, ["RAZORPAY_KEY_SECRET", "RAZORPAY_SECRET"])
    if not key_id or not secret:
        main_missing("RAZORPAY_KEY_ID + RAZORPAY_KEY_SECRET")
    return key_id, secret


def validate(key_id: str, secret: str) -> tuple[bool, str]:
    r = requests.get(
        "https://api.razorpay.com/v1/payments",
        params={"count": 1},
        auth=(key_id, secret),
        timeout=DEFAULT_HTTP_TIMEOUT,
    )
    if r.status_code == 401:
        return False, "Credenciais inválidas (401)"
    if r.status_code != 200:
        return False, f"HTTP {r.status_code}: {r.text[:200]}"
    count = r.json().get("count", 0)
    return True, f"API OK — pagamentos visíveis: {count}"


def run_interactive(env: dict[str, str]) -> int:
    key_id, secret = razorpay_config(env)
    print(f"Razorpay — key …{key_id[-6:]}", flush=True)
    log_step("A validar credenciais…")
    try:
        ok, msg = run_with_timeout(
            lambda: validate(key_id, secret),
            DEFAULT_OPERATION_TIMEOUT,
            "Razorpay",
        )
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1
    print(msg, flush=True)
    return 0 if ok else 1


def main() -> None:
    args = env_arg_parser("Razorpay checker").parse_args()
    env = load_env_keys(Path(args.env))
    if is_interactive():
        sys.exit(run_interactive(env))
    try:
        ok, msg = run_with_timeout(
            lambda: validate(*razorpay_config(env)),
            DEFAULT_OPERATION_TIMEOUT,
            "Razorpay",
        )
        print(msg, flush=True)
        sys.exit(0 if ok else 1)
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        sys.exit(1)


if __name__ == "__main__":
    main()
