#!/usr/bin/env python3
"""Valida FCM server key (legacy)."""

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


def fcm_key(env: dict[str, str]) -> str:
    key = pick_key(env, ["FCM_SERVER_KEY", "FIREBASE_SERVER_KEY", "FCM_KEY"])
    if not key:
        main_missing("FCM_SERVER_KEY")
    return key


def validate(server_key: str) -> tuple[bool, str]:
    r = requests.post(
        "https://fcm.googleapis.com/fcm/send",
        headers={
            "Authorization": f"key={server_key}",
            "Content-Type": "application/json",
        },
        json={"to": "/topics/goscan-dry-run", "dry_run": True, "data": {"ping": "1"}},
        timeout=DEFAULT_HTTP_TIMEOUT,
    )
    if r.status_code == 401:
        return False, "Server key inválida (401)"
    data = r.json()
    if data.get("failure", 0) == 1 and data.get("results"):
        err = data["results"][0].get("error", "")
        if err in ("InvalidRegistration", "NotRegistered", "MismatchSenderId"):
            return True, f"Key válida — dry_run ({err})"
    if r.status_code == 200 and data.get("success") in (0, 1):
        return True, "FCM server key válida"
    return False, f"HTTP {r.status_code}: {r.text[:200]}"


def run_batch(env: dict[str, str]) -> int:
    key = fcm_key(env)
    log_step("FCM…")
    try:
        ok, msg = run_with_timeout(lambda: validate(key), DEFAULT_OPERATION_TIMEOUT, "FCM")
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1
    if not ok:
        print(msg, flush=True)
        return 1
    print_summary(msg)
    return 0


def main() -> None:
    args = env_arg_parser("FCM checker").parse_args()
    env = load_env_keys(Path(args.env))
    if is_batch_mode(args):
        sys.exit(run_batch(env))
    if is_interactive():
        print(f"FCM — …{fcm_key(env)[-6:]}", flush=True)
    sys.exit(run_batch(env))


if __name__ == "__main__":
    main()
