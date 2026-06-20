#!/usr/bin/env python3
"""Valida Mailgun e envia email de teste."""

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


def mailgun_config(env: dict[str, str]) -> tuple[str, str, str]:
    domain = pick_key(env, ["MAILGUN_DOMAIN", "MG_DOMAIN"])
    api_key = pick_key(env, ["MAILGUN_SECRET", "MAILGUN_API_KEY", "MG_SECRET"])
    if not domain or not api_key:
        main_missing("MAILGUN_DOMAIN + MAILGUN_SECRET")
    base = pick_key(env, ["MAILGUN_ENDPOINT", "MAILGUN_BASE_URL"]) or "https://api.mailgun.net/v3"
    return domain, api_key, base.rstrip("/")


def validate(domain: str, api_key: str, base: str) -> tuple[bool, str]:
    r = requests.get(f"{base}/domains/{domain}", auth=("api", api_key), timeout=30)
    if r.status_code == 401:
        return False, "API key inválida (401)"
    if r.status_code == 404:
        return False, f"Domínio '{domain}' não encontrado"
    if r.status_code != 200:
        return False, f"HTTP {r.status_code}: {r.text[:200]}"
    state = r.json().get("domain", {}).get("state", "?")
    return True, f"Domínio activo — state {state}"


def send_mail(domain: str, api_key: str, base: str, from_addr: str, to_addr: str, subject: str, body: str) -> None:
    r = requests.post(
        f"{base}/{domain}/messages",
        auth=("api", api_key),
        data={"from": from_addr, "to": to_addr, "subject": subject, "text": body},
        timeout=30,
    )
    if r.status_code not in (200, 201):
        raise RuntimeError(f"HTTP {r.status_code}: {r.text[:300]}")


def run_interactive(env: dict[str, str]) -> int:
    domain, api_key, base = mailgun_config(env)
    ok, msg = validate(domain, api_key, base)
    if not ok:
        print(msg)
        return 1
    print(f"Mailgun OK — {msg} ({domain})")

    from_addr = pick_key(env, ["MAIL_FROM_ADDRESS", "MAILGUN_FROM", "MAIL_FROM"]) or prompt_required("Remetente")
    print("\nPreencha o email de teste:")
    to_addr = prompt_required("Destinatário")
    subject = prompt_required("Assunto")
    body = prompt_multiline("Mensagem")

    try:
        send_mail(domain, api_key, base, from_addr, to_addr, subject, body)
    except Exception as exc:
        print(f"Falha ao enviar: {exc}")
        return 1

    print(f"OK — email enviado para {to_addr}")
    return 0


def main() -> None:
    args = env_arg_parser("Mailgun checker").parse_args()
    env = load_env_keys(Path(args.env))
    if is_interactive():
        sys.exit(run_interactive(env))
    ok, msg = validate(*mailgun_config(env))
    print(msg)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
