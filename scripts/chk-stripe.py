#!/usr/bin/env python3
"""Valida chave Stripe e mostra saldo."""

from __future__ import annotations

import sys
from pathlib import Path

import requests

from envutil import env_arg_parser, is_interactive, load_env_keys, main_missing, pick_key


def stripe_key(env: dict[str, str]) -> str:
    key = pick_key(env, ["STRIPE_SECRET", "STRIPE_SECRET_KEY", "STRIPE_APIKEY"])
    if not key:
        main_missing("STRIPE_SECRET")
    return key


def fetch_balance(secret: str) -> tuple[bool, str]:
    r = requests.get(
        "https://api.stripe.com/v1/balance",
        auth=(secret, ""),
        timeout=30,
    )
    if r.status_code == 401:
        return False, "Chave inválida (401)"
    if r.status_code != 200:
        return False, f"HTTP {r.status_code}: {r.text[:200]}"
    data = r.json()
    lines = []
    for bucket in ("available", "pending"):
        for item in data.get(bucket, []):
            amount = item.get("amount", 0) / 100
            cur = item.get("currency", "?").upper()
            lines.append(f"{bucket}: {amount:.2f} {cur}")
    return True, "\n".join(lines) if lines else "OK — conta activa"


def run_interactive(env: dict[str, str]) -> int:
    secret = stripe_key(env)
    print(f"Stripe — chave …{secret[-6:]}")
    ok, msg = fetch_balance(secret)
    if not ok:
        print(msg)
        return 1
    print("Chave válida. Saldo:")
    print(msg)
    return 0


def main() -> None:
    args = env_arg_parser("Stripe checker").parse_args()
    env = load_env_keys(Path(args.env))
    if is_interactive():
        sys.exit(run_interactive(env))
    ok, msg = fetch_balance(stripe_key(env))
    print(msg)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
