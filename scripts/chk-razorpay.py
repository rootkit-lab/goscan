#!/usr/bin/env python3
"""Valida credenciais Razorpay."""

from __future__ import annotations

import sys
from pathlib import Path

import requests

from envutil import env_arg_parser, is_interactive, load_env_keys, main_missing, pick_key


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
        timeout=30,
    )
    if r.status_code == 401:
        return False, "Credenciais inválidas (401)"
    if r.status_code != 200:
        return False, f"HTTP {r.status_code}: {r.text[:200]}"
    count = r.json().get("count", 0)
    return True, f"API OK — pagamentos visíveis: {count}"


def run_interactive(env: dict[str, str]) -> int:
    key_id, secret = razorpay_config(env)
    print(f"Razorpay — key …{key_id[-6:]}")
    ok, msg = validate(key_id, secret)
    print(msg)
    return 0 if ok else 1


def main() -> None:
    args = env_arg_parser("Razorpay checker").parse_args()
    env = load_env_keys(Path(args.env))
    if is_interactive():
        sys.exit(run_interactive(env))
    ok, msg = validate(*razorpay_config(env))
    print(msg)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
