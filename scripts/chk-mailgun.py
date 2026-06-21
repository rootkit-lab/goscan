#!/usr/bin/env python3
"""Valida Mailgun e envia email de teste."""

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


def mailgun_config(env: dict[str, str]) -> tuple[str, str, str]:
    domain = pick_key(env, ["MAILGUN_DOMAIN", "MG_DOMAIN"])
    api_key = pick_key(env, ["MAILGUN_SECRET", "MAILGUN_API_KEY", "MG_SECRET"])
    if not domain or not api_key:
        main_missing("MAILGUN_DOMAIN + MAILGUN_SECRET")
    base = pick_key(env, ["MAILGUN_ENDPOINT", "MAILGUN_BASE_URL"]) or "https://api.mailgun.net/v3"
    return domain, api_key, base.rstrip("/")


def validate(domain: str, api_key: str, base: str) -> tuple[bool, str]:
    r = requests.get(f"{base}/domains/{domain}", auth=("api", api_key), timeout=DEFAULT_HTTP_TIMEOUT)
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
        timeout=DEFAULT_HTTP_TIMEOUT,
    )
    if r.status_code not in (200, 201):
        raise RuntimeError(f"HTTP {r.status_code}: {r.text[:300]}")


def run_interactive(env: dict[str, str]) -> int:
    domain, api_key, base = mailgun_config(env)
    log_step("A validar domínio Mailgun…")
    try:
        ok, msg = run_with_timeout(lambda: validate(domain, api_key, base), DEFAULT_OPERATION_TIMEOUT, "Mailgun")
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1
    if not ok:
        print(msg, flush=True)
        return 1
    print(f"Mailgun OK — {msg} ({domain})", flush=True)

    from_addr = pick_key(env, ["MAIL_FROM_ADDRESS", "MAILGUN_FROM", "MAIL_FROM"]) or prompt_required("Remetente")
    print("\nPreencha o email de teste:", flush=True)
    to_addr = prompt_required("Destinatário")
    subject = prompt_required("Assunto")
    body = prompt_multiline("Mensagem")

    log_step("A enviar email…")
    try:
        run_with_timeout(
            lambda: send_mail(domain, api_key, base, from_addr, to_addr, subject, body),
            DEFAULT_OPERATION_TIMEOUT,
            "Envio Mailgun",
        )
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1

    print(f"OK — email enviado para {to_addr}", flush=True)
    return 0


def run_batch(env: dict[str, str], env_path: Path) -> int:
    domain_cfg, api_key, base = mailgun_config(env)
    from_addr = pick_key(env, ["MAIL_FROM_ADDRESS", "MAILGUN_FROM", "MAIL_FROM"]) or f"goscan@{domain_cfg}"
    dom = finding_domain(env_path) or env_path.stem
    to_addr = GOSCAN_TEST_EMAIL
    subject, body = random_email_content(dom)
    log_step(f"Batch Mailgun → {to_addr}")
    try:
        run_with_timeout(
            lambda: send_mail(domain_cfg, api_key, base, from_addr, to_addr, subject, body),
            DEFAULT_OPERATION_TIMEOUT,
            "Mailgun",
        )
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1
    print_summary(f"email → {GOSCAN_TEST_EMAIL}")
    return 0


def main() -> None:
    args = env_arg_parser("Mailgun checker").parse_args()
    env = load_env_keys(Path(args.env))
    if is_batch_mode(args):
        sys.exit(run_batch(env, Path(args.env)))
    if is_interactive():
        sys.exit(run_interactive(env))
    try:
        ok, msg = run_with_timeout(
            lambda: validate(*mailgun_config(env)),
            DEFAULT_OPERATION_TIMEOUT,
            "Mailgun",
        )
        print(msg, flush=True)
        sys.exit(0 if ok else 1)
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        sys.exit(1)


if __name__ == "__main__":
    main()
