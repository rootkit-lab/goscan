#!/usr/bin/env python3
"""Valida bot Telegram."""

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


def telegram_token(env: dict[str, str]) -> str:
    token = pick_key(env, ["TELEGRAM_API_TOKEN", "TELEGRAM_BOT_TOKEN", "TELEGRAM_TOKEN"])
    if not token:
        main_missing("TELEGRAM_API_TOKEN")
    return token


def validate(token: str) -> tuple[bool, str]:
    r = requests.get(f"https://api.telegram.org/bot{token}/getMe", timeout=DEFAULT_HTTP_TIMEOUT)
    if r.status_code != 200:
        return False, f"HTTP {r.status_code}: {r.text[:200]}"
    data = r.json()
    if not data.get("ok"):
        return False, data.get("description", "Token inválido")
    bot = data.get("result", {})
    username = bot.get("username", "?")
    return True, f"Bot @{username} — {bot.get('first_name', '')}".strip()


def run_batch(env: dict[str, str]) -> int:
    token = telegram_token(env)
    log_step("Telegram…")
    try:
        ok, msg = run_with_timeout(lambda: validate(token), DEFAULT_OPERATION_TIMEOUT, "Telegram")
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1
    if not ok:
        print(msg, flush=True)
        return 1
    print_summary(msg)
    return 0


def main() -> None:
    args = env_arg_parser("Telegram checker").parse_args()
    env = load_env_keys(Path(args.env))
    if is_batch_mode(args):
        sys.exit(run_batch(env))
    if is_interactive():
        print("Telegram — a validar bot…", flush=True)
    sys.exit(run_batch(env))


if __name__ == "__main__":
    main()
