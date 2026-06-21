#!/usr/bin/env python3
"""Valida token HubSpot."""

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


def hubspot_token(env: dict[str, str]) -> str:
    token = pick_key(env, ["HUB_SPOT_API_KEY", "HUBSPOT_APIKEY", "HUBSPOT_ACCESS_TOKEN", "HUBSPOT_API_KEY"])
    if not token:
        main_missing("HUB_SPOT_API_KEY / HUBSPOT_APIKEY")
    return token


def validate(token: str) -> tuple[bool, str]:
    r = requests.get(
        "https://api.hubapi.com/crm/v3/objects/contacts",
        headers={"Authorization": f"Bearer {token}"},
        params={"limit": 1},
        timeout=DEFAULT_HTTP_TIMEOUT,
    )
    if r.status_code == 401:
        return False, "Token inválido (401)"
    if r.status_code not in (200, 403):
        return False, f"HTTP {r.status_code}: {r.text[:200]}"
    if r.status_code == 403:
        return True, "Token válido — sem acesso a contacts (403 scope)"
    total = r.json().get("total", "?")
    return True, f"CRM OK — contacts total {total}"


def run_batch(env: dict[str, str]) -> int:
    token = hubspot_token(env)
    log_step("HubSpot…")
    try:
        ok, msg = run_with_timeout(lambda: validate(token), DEFAULT_OPERATION_TIMEOUT, "HubSpot")
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1
    if not ok:
        print(msg, flush=True)
        return 1
    print_summary(msg)
    return 0


def main() -> None:
    args = env_arg_parser("HubSpot checker").parse_args()
    env = load_env_keys(Path(args.env))
    if is_batch_mode(args):
        sys.exit(run_batch(env))
    if is_interactive():
        print(f"HubSpot — …{hubspot_token(env)[-6:]}", flush=True)
    sys.exit(run_batch(env))


if __name__ == "__main__":
    main()
