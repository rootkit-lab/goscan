#!/usr/bin/env python3
"""Valida credenciais PayPal (OAuth)."""

from __future__ import annotations

import sys
from pathlib import Path

import requests

from envutil import env_arg_parser, is_interactive, load_env_keys, main_missing, pick_key


def paypal_config(env: dict[str, str]) -> tuple[str, str, str]:
    client_id = pick_key(env, ["PAYPAL_CLIENT_ID", "PAYPAL_USERNAME"])
    secret = pick_key(env, ["PAYPAL_CLIENT_SECRET", "PAYPAL_SECRET", "PAYPAL_PASSWORD"])
    if not client_id or not secret:
        main_missing("PAYPAL_CLIENT_ID + PAYPAL_CLIENT_SECRET")
    mode = (pick_key(env, ["PAYPAL_MODE"]) or "sandbox").lower()
    base = "https://api-m.paypal.com" if mode == "live" else "https://api-m.sandbox.paypal.com"
    return client_id, secret, base


def get_token(client_id: str, secret: str, base: str) -> tuple[bool, str]:
    r = requests.post(
        f"{base}/v1/oauth2/token",
        auth=(client_id, secret),
        data={"grant_type": "client_credentials"},
        headers={"Accept": "application/json"},
        timeout=30,
    )
    if r.status_code == 401:
        return False, "Credenciais inválidas (401)"
    if r.status_code != 200:
        return False, f"HTTP {r.status_code}: {r.text[:200]}"
    data = r.json()
    scope = data.get("scope", "")
    expires = data.get("expires_in", "?")
    return True, f"Token OK — expira em {expires}s\nScope: {scope[:120]}…" if len(scope) > 120 else f"Token OK — expira em {expires}s\nScope: {scope}"


def run_interactive(env: dict[str, str]) -> int:
    client_id, secret, base = paypal_config(env)
    mode = "live" if "sandbox" not in base else "sandbox"
    print(f"PayPal detectado — modo {mode}")
    print(f"  Client ID: …{client_id[-8:]}")
    ok, msg = get_token(client_id, secret, base)
    if not ok:
        print(msg)
        return 1
    print(msg)
    return 0


def main() -> None:
    args = env_arg_parser("PayPal checker").parse_args()
    env = load_env_keys(Path(args.env))
    if is_interactive():
        sys.exit(run_interactive(env))
    client_id, secret, base = paypal_config(env)
    ok, msg = get_token(client_id, secret, base)
    print(msg)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
