#!/usr/bin/env python3
"""Valida DSN Sentry (auth check)."""

from __future__ import annotations

import sys
from pathlib import Path
from urllib.parse import urlparse

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


def sentry_dsn(env: dict[str, str]) -> str:
    dsn = pick_key(env, ["SENTRY_LARAVEL_DSN", "SENTRY_DSN", "NEXT_PUBLIC_SENTRY_DSN"])
    if not dsn:
        main_missing("SENTRY_LARAVEL_DSN/SENTRY_DSN")
    return dsn


def validate_dsn(dsn: str) -> tuple[bool, str]:
    parsed = urlparse(dsn)
    if parsed.scheme not in ("http", "https") or not parsed.hostname:
        return False, "DSN inválido"
    # Public key in username, secret in password — ping store endpoint
    project = parsed.path.strip("/").split("/")[-1]
    if not project:
        return False, "DSN sem project id"
    host = parsed.hostname
    url = f"https://{host}/api/{project}/store/"
    auth = f"Sentry sentry_key={parsed.username}"
    if parsed.password:
        auth += f", sentry_secret={parsed.password}"
    r = requests.post(
        url,
        headers={"X-Sentry-Auth": auth, "Content-Type": "application/json"},
        json={"message": "goscan ping", "level": "info"},
        timeout=DEFAULT_HTTP_TIMEOUT,
    )
    if r.status_code in (200, 201):
        return True, f"DSN OK — project {project}"
    if r.status_code == 401:
        return False, "DSN rejeitado (401)"
    return False, f"HTTP {r.status_code}: {r.text[:120]}"


def run_batch(env: dict[str, str]) -> int:
    dsn = sentry_dsn(env)
    log_step("Sentry DSN…")
    try:
        ok, msg = run_with_timeout(lambda: validate_dsn(dsn), DEFAULT_OPERATION_TIMEOUT, "Sentry")
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1
    if not ok:
        print(msg, flush=True)
        return 1
    print_summary(msg)
    return 0


def main() -> None:
    args = env_arg_parser("Sentry checker").parse_args()
    env = load_env_keys(Path(args.env))
    if is_batch_mode(args):
        sys.exit(run_batch(env))
    if is_interactive():
        parsed = urlparse(sentry_dsn(env))
        print(f"Sentry — {parsed.hostname} project …", flush=True)
    sys.exit(run_batch(env))


if __name__ == "__main__":
    main()
