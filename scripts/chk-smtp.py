#!/usr/bin/env python3
"""Testa credenciais SMTP enviando email de teste."""

from __future__ import annotations

import smtplib
import ssl
import sys
from email.mime.text import MIMEText
from pathlib import Path

from envutil import (
    DEFAULT_CONNECT_TIMEOUT,
    DEFAULT_OPERATION_TIMEOUT,
    EnvContext,
    GOSCAN_TEST_EMAIL,
    env_arg_parser,
    finding_domain,
    format_network_error,
    is_batch_mode,
    is_interactive,
    is_local_host,
    load_env_context,
    log_step,
    main_missing,
    network_socket_timeout,
    parse_port,
    print_summary,
    prompt_multiline,
    prompt_required,
    random_email_content,
    resolve_mail_host,
    run_with_timeout,
    skip_smtp_dev_host,
)

SMTP_TIMEOUT = DEFAULT_CONNECT_TIMEOUT


def smtp_config(ctx: EnvContext) -> tuple[str, int, str, str, str, str]:
    raw_host = ctx.get(["MAIL_HOST", "SMTP_HOST", "MAILER_HOST", "EMAIL_HOST"])
    if not raw_host:
        main_missing("MAIL_HOST/SMTP_HOST")
    port_raw = ctx.get(["MAIL_PORT", "SMTP_PORT", "MAILER_PORT", "EMAIL_PORT"]) or "587"
    port = parse_port(port_raw, 587)
    if port is None:
        print(f"SKIP: MAIL_PORT inválida ({port_raw})", file=sys.stderr)
        sys.exit(2)
    host, note = resolve_mail_host(raw_host, ctx.path, ctx.env, port)
    if not host:
        main_missing("MAIL_HOST/SMTP_HOST")
    if note:
        log_step(note)
    user = ctx.get(["MAIL_USERNAME", "SMTP_USERNAME", "MAIL_USER", "SMTP_USER", "EMAIL_USERNAME"])
    if not user:
        main_missing("MAIL_USERNAME/SMTP_USERNAME")
    password = ctx.get(["MAIL_PASSWORD", "SMTP_PASSWORD", "EMAIL_PASSWORD"])
    if not password:
        main_missing("MAIL_PASSWORD/SMTP_PASSWORD")
    from_addr = ctx.get(["MAIL_FROM_ADDRESS", "MAIL_FROM", "SMTP_FROM", "EMAIL_FROM"]) or user
    encryption = pick_encryption(ctx, port)
    return host, port, user, password, from_addr, encryption


def pick_encryption(ctx: EnvContext, port: int) -> str:
    enc = ctx.get(["MAIL_ENCRYPTION", "SMTP_ENCRYPTION", "MAILER_ENCRYPTION"])
    if enc is None:
        if port == 465:
            return "ssl"
        if port in (587, 25):
            return "tls"
        return "none"
    enc = enc.strip().lower()
    if enc in ("", "null", "none", "false"):
        if port == 465:
            return "ssl"
        if port in (587, 25, 2525):
            return "tls"
        return "none"
    return enc


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
    log_step(f"[connect] {host}:{port} (timeout {SMTP_TIMEOUT}s, enc={encryption})…")
    network_socket_timeout(SMTP_TIMEOUT)

    if encryption == "ssl" or port == 465:
        with smtplib.SMTP_SSL(host, port, context=context, timeout=SMTP_TIMEOUT) as server:
            log_step("[auth] A autenticar…")
            server.login(user, password)
            log_step("[send] A enviar mensagem…")
            server.send_message(msg)
        return

    with smtplib.SMTP(host, port, timeout=SMTP_TIMEOUT) as server:
        server.ehlo()
        if encryption in ("tls", "starttls"):
            log_step("[tls] STARTTLS…")
            server.starttls(context=context)
            server.ehlo()
        log_step("[auth] A autenticar…")
        server.login(user, password)
        log_step("[send] A enviar mensagem…")
        server.send_message(msg)


def run_interactive(ctx: EnvContext) -> int:
    host, port, user, password, from_addr, encryption = smtp_config(ctx)
    print("Configuração SMTP detectada:", flush=True)
    print(f"  Host: {host}:{port}", flush=True)
    if is_local_host(host):
        print("  ⚠ Host local — confirme MAIL_HOST no .env (ex.: smtp.office365.com)", flush=True)
    print(f"  User: {user}", flush=True)
    print(f"  From: {from_addr}", flush=True)
    print(f"  Enc:  {encryption}", flush=True)
    print("\nPreencha o email de teste:", flush=True)

    to_addr = prompt_required("Destinatário")
    subject = prompt_required("Assunto")
    body = prompt_multiline("Mensagem")

    log_step("A iniciar teste SMTP…")

    try:
        run_with_timeout(
            lambda: send_mail(host, port, user, password, from_addr, encryption, to_addr, subject, body),
            DEFAULT_OPERATION_TIMEOUT,
            "Envio SMTP",
        )
    except smtplib.SMTPException as exc:
        print(f"Falha SMTP: {exc}", flush=True)
        return 1
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1

    print(f"OK — email enviado para {to_addr}", flush=True)
    return 0


def run_batch(ctx: EnvContext) -> int:
    host, port, user, password, from_addr, encryption = smtp_config(ctx)
    skip_smtp_dev_host(ctx, host, port)
    domain = finding_domain(ctx.path) or ctx.path.stem
    to_addr = GOSCAN_TEST_EMAIL
    subject, body = random_email_content(domain)
    log_step(f"Batch SMTP → {to_addr}")

    try:
        run_with_timeout(
            lambda: send_mail(host, port, user, password, from_addr, encryption, to_addr, subject, body),
            DEFAULT_OPERATION_TIMEOUT,
            "Envio SMTP",
        )
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1

    print_summary(f"email → {GOSCAN_TEST_EMAIL}")
    print(f"OK — email enviado → {GOSCAN_TEST_EMAIL}", flush=True)
    return 0


def main() -> None:
    args = env_arg_parser("SMTP checker").parse_args()
    ctx = load_env_context(Path(args.env))
    if is_batch_mode(args):
        sys.exit(run_batch(ctx))
    if is_interactive():
        sys.exit(run_interactive(ctx))

    host, port, user, password, from_addr, encryption = smtp_config(ctx)
    to_addr = ctx.get(["MAIL_TO", "SMTP_TO"])
    subject = ctx.get(["MAIL_SUBJECT", "SMTP_SUBJECT"])
    body = ctx.get(["MAIL_BODY", "SMTP_BODY"])
    if not to_addr or not subject or not body:
        print("Modo não-interactivo requer MAIL_TO, MAIL_SUBJECT e MAIL_BODY no .env")
        sys.exit(2)
    try:
        run_with_timeout(
            lambda: send_mail(host, port, user, password, from_addr, encryption, to_addr, subject, body),
            DEFAULT_OPERATION_TIMEOUT,
            "Envio SMTP",
        )
        print(f"OK — email enviado para {to_addr}", flush=True)
        sys.exit(0)
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        sys.exit(1)


if __name__ == "__main__":
    main()
