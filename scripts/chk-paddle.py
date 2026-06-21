#!/usr/bin/env python3
"""Valida credenciais Paddle (Billing ou Classic)."""

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


def validate_billing(api_key: str) -> tuple[bool, str]:
    r = requests.get(
        "https://api.paddle.com/event-types",
        headers={"Authorization": f"Bearer {api_key}"},
        timeout=DEFAULT_HTTP_TIMEOUT,
    )
    if r.status_code == 401:
        return False, "API key inválida (401)"
    if r.status_code != 200:
        return False, f"HTTP {r.status_code}: {r.text[:200]}"
    count = len(r.json().get("data", []))
    return True, f"Paddle Billing OK — {count} event types"


def validate_classic(vendor_id: str, auth_code: str) -> tuple[bool, str]:
    r = requests.post(
        "https://vendors.paddle.com/api/2.0/product/get_products",
        json={"vendor_id": vendor_id, "vendor_auth_code": auth_code},
        timeout=DEFAULT_HTTP_TIMEOUT,
    )
    if r.status_code != 200:
        return False, f"HTTP {r.status_code}: {r.text[:200]}"
    data = r.json()
    if not data.get("success"):
        return False, data.get("error", {}).get("message", "Falha Paddle Classic")
    count = len(data.get("response", {}).get("products", []))
    return True, f"Paddle Classic OK — {count} produtos"


def run_batch(env: dict[str, str]) -> int:
    billing_key = pick_key(env, ["PADDLE_API_KEY", "PADDLE_SECRET_KEY"])
    vendor = pick_key(env, ["PADDLE_VENDOR_ID", "VENDOR_ID"])
    auth = pick_key(env, ["PADDLE_AUTH_CODE", "PADDLE_VENDOR_AUTH_CODE"])

    log_step("Paddle…")
    try:
        if billing_key:
            ok, msg = run_with_timeout(
                lambda: validate_billing(billing_key),
                DEFAULT_OPERATION_TIMEOUT,
                "Paddle",
            )
        elif vendor and auth:
            ok, msg = run_with_timeout(
                lambda: validate_classic(vendor, auth),
                DEFAULT_OPERATION_TIMEOUT,
                "Paddle",
            )
        else:
            main_missing("PADDLE_API_KEY ou PADDLE_VENDOR_ID + PADDLE_AUTH_CODE")
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1
    if not ok:
        print(msg, flush=True)
        return 1
    print_summary(msg)
    return 0


def main() -> None:
    args = env_arg_parser("Paddle checker").parse_args()
    env = load_env_keys(Path(args.env))
    if is_batch_mode(args):
        sys.exit(run_batch(env))
    if is_interactive():
        print("Paddle — a validar…", flush=True)
    sys.exit(run_batch(env))


if __name__ == "__main__":
    main()
