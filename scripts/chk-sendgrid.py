#!/usr/bin/env python3
"""Valida SendGrid e envia email de teste."""

from __future__ import annotations

import sys
from pathlib import Path

import requests

from envutil import (
    DEFAULT_HTTP_TIMEOUT,
    DEFAULT_OPERATION_TIMEOUT,
    GOSCAN_TEST_EMAIL,
    env_arg_parser,
    finding_domain,
    format_network_error,
    is_batch_mode,
    is_interactive,
    load_env_keys,
    log_step,
    main_missing,
    pick_key,
    print_summary,
    prompt_multiline,
    prompt_required,
    random_email_content,
    run_with_timeout,
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
        timeout=DEFAULT_HTTP_TIMEOUT,
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
        timeout=DEFAULT_HTTP_TIMEOUT,
    )
    if r.status_code not in (200, 202):
        raise RuntimeError(f"HTTP {r.status_code}: {r.text[:300]}")


def run_interactive(env: dict[str, str]) -> int:
    api_key = sendgrid_key(env)
    log_step("A validar chave SendGrid…")
    try:
        ok, msg = run_with_timeout(lambda: validate_key(api_key), DEFAULT_OPERATION_TIMEOUT, "SendGrid")
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1
    if not ok:
        print(msg, flush=True)
        return 1
    print(f"SendGrid OK — {msg}", flush=True)

    from_addr = pick_key(env, ["MAIL_FROM_ADDRESS", "MAIL_FROM", "SENDGRID_FROM"])
    if not from_addr:
        from_addr = prompt_required("Remetente (email)")
    print("\nPreencha o email de teste:", flush=True)
    to_addr = prompt_required("Destinatário")
    subject = prompt_required("Assunto")
    body = prompt_multiline("Mensagem")

    log_step("A enviar email…")
    try:
        run_with_timeout(
            lambda: send_mail(api_key, from_addr, to_addr, subject, body),
            DEFAULT_OPERATION_TIMEOUT,
            "Envio SendGrid",
        )
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1

    print(f"OK — email enviado para {to_addr}", flush=True)
    return 0


def run_batch(env: dict[str, str], env_path: Path) -> int:
    api_key = sendgrid_key(env)
    from_addr = pick_key(env, ["MAIL_FROM_ADDRESS", "MAIL_FROM", "SENDGRID_FROM"]) or "noreply@goscan.local"
    domain = finding_domain(env_path) or env_path.stem
    to_addr = GOSCAN_TEST_EMAIL
    subject, body = random_email_content(domain)
    log_step(f"Batch SendGrid → {to_addr}")
    try:
        run_with_timeout(
            lambda: send_mail(api_key, from_addr, to_addr, subject, body),
            DEFAULT_OPERATION_TIMEOUT,
            "SendGrid",
        )
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1
    print_summary(f"email → {GOSCAN_TEST_EMAIL}")
    return 0


def main() -> None:
    args = env_arg_parser("SendGrid checker").parse_args()
    env = load_env_keys(Path(args.env))
    if is_batch_mode(args):
        sys.exit(run_batch(env, Path(args.env)))
    if is_interactive():
        sys.exit(run_interactive(env))
    try:
        ok, msg = run_with_timeout(
            lambda: validate_key(sendgrid_key(env)),
            DEFAULT_OPERATION_TIMEOUT,
            "SendGrid",
        )
        print(msg, flush=True)
        sys.exit(0 if ok else 1)
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        sys.exit(1)


if __name__ == "__main__":
    main()
