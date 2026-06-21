#!/usr/bin/env python3
"""Valida credenciais PayPal (OAuth)."""

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
        timeout=DEFAULT_HTTP_TIMEOUT,
    )
    if r.status_code == 401:
        return False, "Credenciais inválidas (401)"
    if r.status_code != 200:
        return False, f"HTTP {r.status_code}: {r.text[:200]}"
    data = r.json()
    scope = data.get("scope", "")
    expires = data.get("expires_in", "?")
    if len(scope) > 120:
        return True, f"Token OK — expira em {expires}s\nScope: {scope[:120]}…"
    return True, f"Token OK — expira em {expires}s\nScope: {scope}"


def run_interactive(env: dict[str, str]) -> int:
    client_id, secret, base = paypal_config(env)
    mode = "live" if "sandbox" not in base else "sandbox"
    print(f"PayPal detectado — modo {mode}", flush=True)
    print(f"  Client ID: …{client_id[-8:]}", flush=True)
    log_step("A obter token OAuth…")
    try:
        ok, msg = run_with_timeout(
            lambda: get_token(client_id, secret, base),
            DEFAULT_OPERATION_TIMEOUT,
            "PayPal OAuth",
        )
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1
    if not ok:
        print(msg, flush=True)
        return 1
    print(msg, flush=True)
    return 0


def main() -> None:
    args = env_arg_parser("PayPal checker").parse_args()
    env = load_env_keys(Path(args.env))
    if is_interactive():
        sys.exit(run_interactive(env))
    try:
        ok, msg = run_with_timeout(
            lambda: get_token(*paypal_config(env)),
            DEFAULT_OPERATION_TIMEOUT,
            "PayPal OAuth",
        )
        print(msg, flush=True)
        sys.exit(0 if ok else 1)
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        sys.exit(1)


if __name__ == "__main__":
    main()
