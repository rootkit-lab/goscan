#!/usr/bin/env python3
"""Valida Twilio e permite enviar SMS de teste."""

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
    prompt_optional,
    prompt_required,
    run_with_timeout,
)


def twilio_config(env: dict[str, str]) -> tuple[str, str]:
    sid = pick_key(env, ["TWILIO_SID", "TWILIO_ACCOUNT_SID"])
    token = pick_key(env, ["TWILIO_AUTH_TOKEN", "TWILIO_TOKEN"])
    if not sid or not token:
        main_missing("TWILIO_SID + TWILIO_AUTH_TOKEN")
    return sid, token


def fetch_account(sid: str, token: str) -> tuple[bool, str]:
    r = requests.get(
        f"https://api.twilio.com/2010-04-01/Accounts/{sid}.json",
        auth=(sid, token),
        timeout=DEFAULT_HTTP_TIMEOUT,
    )
    if r.status_code == 401:
        return False, "Credenciais inválidas (401)"
    if r.status_code != 200:
        return False, f"HTTP {r.status_code}: {r.text[:200]}"
    data = r.json()
    return True, f"Conta: {data.get('friendly_name')} — status {data.get('status')}"


def send_sms(sid: str, token: str, from_num: str, to_num: str, body: str) -> None:
    r = requests.post(
        f"https://api.twilio.com/2010-04-01/Accounts/{sid}/Messages.json",
        auth=(sid, token),
        data={"From": from_num, "To": to_num, "Body": body},
        timeout=DEFAULT_HTTP_TIMEOUT,
    )
    if r.status_code not in (200, 201):
        raise RuntimeError(f"HTTP {r.status_code}: {r.text[:300]}")


def run_interactive(env: dict[str, str]) -> int:
    sid, token = twilio_config(env)
    log_step("A validar conta Twilio…")
    try:
        ok, msg = run_with_timeout(lambda: fetch_account(sid, token), DEFAULT_OPERATION_TIMEOUT, "Twilio")
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1
    if not ok:
        print(msg, flush=True)
        return 1
    print(f"Twilio OK — {msg}", flush=True)

    send = prompt_optional("Enviar SMS de teste? (s/N)", "n").lower()
    if send not in ("s", "sim", "y", "yes"):
        return 0

    from_num = pick_key(env, ["TWILIO_FROM", "TWILIO_PHONE_NUMBER"]) or prompt_required("Número remetente (+…)")
    to_num = prompt_required("Número destinatário (+…)")
    body = prompt_required("Mensagem SMS")

    log_step("A enviar SMS…")
    try:
        run_with_timeout(
            lambda: send_sms(sid, token, from_num, to_num, body),
            DEFAULT_OPERATION_TIMEOUT,
            "Twilio SMS",
        )
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1

    print(f"OK — SMS enviado para {to_num}", flush=True)
    return 0


def main() -> None:
    args = env_arg_parser("Twilio checker").parse_args()
    env = load_env_keys(Path(args.env))
    if is_interactive():
        sys.exit(run_interactive(env))
    try:
        ok, msg = run_with_timeout(
            lambda: fetch_account(*twilio_config(env)),
            DEFAULT_OPERATION_TIMEOUT,
            "Twilio",
        )
        print(msg, flush=True)
        sys.exit(0 if ok else 1)
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        sys.exit(1)


if __name__ == "__main__":
    main()
