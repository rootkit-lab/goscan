#!/usr/bin/env python3
"""Valida SendGrid e envia email de teste."""

from __future__ import annotations

import sys
from pathlib import Path

import requests

from envutil import (
    env_arg_parser,
    is_interactive,
    load_env_keys,
    main_missing,
    pick_key,
    prompt_multiline,
    prompt_required,
)


def sendgrid_key(env: dict[str, str]) -> str:
    key = pick_key(env, ["SENDGRID_API_KEY", "SENDGRID_APIKEY", "SENDGRID_MESSAGE_API_KEY"])
    if not key:
        main_missing("SENDGRID_API_KEY")
    return key


def validate_key(api_key: str) -> tuple[bool, str]:
    r = requests.get(
        "https://api.sendgrid.com/v3/user/profile",
        headers={"Authorization": f"Bearer {api_key}"},
        timeout=30,
    )
    if r.status_code == 401:
        return False, "Chave inválida (401)"
    if r.status_code != 200:
        return False, f"HTTP {r.status_code}: {r.text[:200]}"
    profile = r.json()
    return True, f"Conta: {profile.get('username') or profile.get('email', '?')}"


def send_mail(api_key: str, from_addr: str, to_addr: str, subject: str, body: str) -> None:
    payload = {
        "personalizations": [{"to": [{"email": to_addr}]}],
        "from": {"email": from_addr},
        "subject": subject,
        "content": [{"type": "text/plain", "value": body}],
    }
    r = requests.post(
        "https://api.sendgrid.com/v3/mail/send",
        headers={"Authorization": f"Bearer {api_key}", "Content-Type": "application/json"},
        json=payload,
        timeout=30,
    )
    if r.status_code not in (200, 202):
        raise RuntimeError(f"HTTP {r.status_code}: {r.text[:300]}")


def run_interactive(env: dict[str, str]) -> int:
    api_key = sendgrid_key(env)
    ok, msg = validate_key(api_key)
    if not ok:
        print(msg)
        return 1
    print(f"SendGrid OK — {msg}")

    from_addr = pick_key(env, ["MAIL_FROM_ADDRESS", "MAIL_FROM", "SENDGRID_FROM"])
    if not from_addr:
        from_addr = prompt_required("Remetente (email)")
    print("\nPreencha o email de teste:")
    to_addr = prompt_required("Destinatário")
    subject = prompt_required("Assunto")
    body = prompt_multiline("Mensagem")

    try:
        send_mail(api_key, from_addr, to_addr, subject, body)
    except Exception as exc:
        print(f"Falha ao enviar: {exc}")
        return 1

    print(f"OK — email enviado para {to_addr}")
    return 0


def main() -> None:
    args = env_arg_parser("SendGrid checker").parse_args()
    env = load_env_keys(Path(args.env))
    if is_interactive():
        sys.exit(run_interactive(env))
    ok, msg = validate_key(sendgrid_key(env))
    print(msg)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
