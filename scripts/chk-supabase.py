#!/usr/bin/env python3
"""Valida Supabase (URL + chave anon/service)."""

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
    run_with_timeout,
)


def supabase_config(env: dict[str, str]) -> tuple[str, str]:
    url = pick_key(
        env,
        ["SUPABASE_URL", "VITE_SUPABASE_URL", "NEXT_PUBLIC_SUPABASE_URL", "SUPABASE_PROJECT_URL"],
    )
    key = pick_key(
        env,
        ["SUPABASE_SERVICE_KEY", "SUPABASE_SERVICE_ROLE_KEY", "SUPABASE_KEY", "VITE_SUPABASE_ANON_KEY", "NEXT_PUBLIC_SUPABASE_ANON_KEY"],
    )
    if not url or not key:
        main_missing("SUPABASE_URL + SUPABASE_KEY/ANON_KEY")
    return url.rstrip("/"), key


def probe(url: str, key: str) -> tuple[bool, str]:
    r = requests.get(
        f"{url}/rest/v1/",
        headers={"apikey": key, "Authorization": f"Bearer {key}"},
        timeout=DEFAULT_HTTP_TIMEOUT,
    )
    if r.status_code == 401:
        return False, "Chave inválida (401)"
    if r.status_code >= 400:
        return False, f"HTTP {r.status_code}: {r.text[:200]}"
    return True, "API REST acessível"


def list_tables(url: str, key: str) -> str:
    r = requests.get(
        f"{url}/rest/v1/",
        headers={"apikey": key, "Authorization": f"Bearer {key}", "Accept": "application/openapi+json"},
        timeout=DEFAULT_HTTP_TIMEOUT,
    )
    if r.status_code != 200:
        return f"Não foi possível listar tabelas: HTTP {r.status_code}"
    paths = r.json().get("paths", {})
    tables = sorted(p.strip("/") for p in paths if p.startswith("/") and "{" not in p)
    if not tables:
        return "Nenhuma tabela exposta via REST"
    lines = "\n".join(f"  - {t}" for t in tables[:25])
    suffix = "\n  …" if len(tables) > 25 else ""
    return f"Tabelas ({len(tables)}):\n{lines}{suffix}"


def run_interactive(env: dict[str, str]) -> int:
    url, key = supabase_config(env)
    print(f"Supabase: {url}", flush=True)
    log_step("A validar API REST…")
    try:
        ok, msg = run_with_timeout(lambda: probe(url, key), DEFAULT_OPERATION_TIMEOUT, "Supabase")
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1
    if not ok:
        print(msg, flush=True)
        return 1
    print(f"Chave válida — {msg}", flush=True)

    show = prompt_optional("Listar tabelas REST? (S/n)", "s").lower()
    if show not in ("n", "nao", "no"):
        log_step("A listar tabelas…")
        try:
            print(
                run_with_timeout(lambda: list_tables(url, key), DEFAULT_OPERATION_TIMEOUT, "Supabase tabelas"),
                flush=True,
            )
        except Exception as exc:
            print(format_network_error(exc), flush=True)
    return 0


def main() -> None:
    args = env_arg_parser("Supabase checker").parse_args()
    env = load_env_keys(Path(args.env))
    if is_interactive():
        sys.exit(run_interactive(env))
    try:
        ok, msg = run_with_timeout(
            lambda: probe(*supabase_config(env)),
            DEFAULT_OPERATION_TIMEOUT,
            "Supabase",
        )
        print(msg, flush=True)
        sys.exit(0 if ok else 1)
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        sys.exit(1)


if __name__ == "__main__":
    main()
