#!/usr/bin/env python3
"""Testa credenciais SMTP enviando email de teste."""

from __future__ import annotations

import smtplib
import ssl
import sys
from email.mime.text import MIMEText
from pathlib import Path

from envutil import (
    EnvContext,
    env_arg_parser,
    is_interactive,
    load_env_context,
    main_missing,
    print_host_note,
    prompt_multiline,
    prompt_required,
)


def smtp_config(ctx: EnvContext) -> tuple[str, int, str, str, str, str, str | None]:
    raw_host = ctx.get(["MAIL_HOST", "SMTP_HOST", "MAILER_HOST", "EMAIL_HOST"])
    if not raw_host:
        main_missing("MAIL_HOST/SMTP_HOST")
    host, note = ctx.resolve_host(raw_host)
    if not host:
        main_missing("MAIL_HOST/SMTP_HOST")
    port_raw = ctx.get(["MAIL_PORT", "SMTP_PORT", "MAILER_PORT", "EMAIL_PORT"]) or "587"
    user = ctx.get(["MAIL_USERNAME", "SMTP_USERNAME", "MAIL_USER", "SMTP_USER", "EMAIL_USERNAME"])
    if not user:
        main_missing("MAIL_USERNAME/SMTP_USERNAME")
    password = ctx.get(["MAIL_PASSWORD", "SMTP_PASSWORD", "EMAIL_PASSWORD"])
    if not password:
        main_missing("MAIL_PASSWORD/SMTP_PASSWORD")
    from_addr = ctx.get(["MAIL_FROM_ADDRESS", "MAIL_FROM", "SMTP_FROM", "EMAIL_FROM"]) or user
    encryption = (ctx.get(["MAIL_ENCRYPTION", "SMTP_ENCRYPTION", "MAILER_ENCRYPTION"]) or "tls").lower()
    return host, int(port_raw), user, password, from_addr, encryption, note


def send_mail(
    host: str,
    port: int,
    user: str,
    password: str,
    from_addr: str,
    encryption: str,
    to_addr: str,
    subject: str,
    body: str,
) -> None:
    msg = MIMEText(body, "plain", "utf-8")
    msg["Subject"] = subject
    msg["From"] = from_addr
    msg["To"] = to_addr

    context = ssl.create_default_context()
    if encryption == "ssl" or port == 465:
        with smtplib.SMTP_SSL(host, port, context=context, timeout=30) as server:
            server.login(user, password)
            server.send_message(msg)
    else:
        with smtplib.SMTP(host, port, timeout=30) as server:
            server.ehlo()
            if encryption in ("tls", "starttls", ""):
                server.starttls(context=context)
                server.ehlo()
            server.login(user, password)
            server.send_message(msg)


def run_interactive(ctx: EnvContext) -> int:
    host, port, user, password, from_addr, encryption, note = smtp_config(ctx)
    print("Configuração SMTP detectada:")
    print(f"  Host: {host}:{port}")
    print_host_note(note)
    print(f"  User: {user}")
    print(f"  From: {from_addr}")
    print(f"  Enc:  {encryption}")
    print("\nPreencha o email de teste:")

    to_addr = prompt_required("Destinatário")
    subject = prompt_required("Assunto")
    body = prompt_multiline("Mensagem")

    try:
        send_mail(host, port, user, password, from_addr, encryption, to_addr, subject, body)
    except Exception as exc:
        print(f"Falha SMTP: {exc}")
        return 1

    print(f"OK — email enviado para {to_addr}")
    return 0


def main() -> None:
    args = env_arg_parser("SMTP checker").parse_args()
    ctx = load_env_context(Path(args.env))
    if is_interactive():
        sys.exit(run_interactive(ctx))

    host, port, user, password, from_addr, encryption, _note = smtp_config(ctx)
    to_addr = ctx.get(["MAIL_TO", "SMTP_TO"])
    subject = ctx.get(["MAIL_SUBJECT", "SMTP_SUBJECT"])
    body = ctx.get(["MAIL_BODY", "SMTP_BODY"])
    if not to_addr or not subject or not body:
        print("Modo não-interactivo requer MAIL_TO, MAIL_SUBJECT e MAIL_BODY no .env")
        sys.exit(2)
    try:
        send_mail(host, port, user, password, from_addr, encryption, to_addr, subject, body)
        print(f"OK — email enviado para {to_addr}")
        sys.exit(0)
    except Exception as exc:
        print(f"Falha SMTP: {exc}")
        sys.exit(1)


if __name__ == "__main__":
    main()
