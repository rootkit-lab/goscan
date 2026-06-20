#!/usr/bin/env python3
"""Testa conexão PostgreSQL."""

from __future__ import annotations

import sys
from pathlib import Path
from urllib.parse import urlparse

import psycopg2

from envutil import (
    EnvContext,
    env_arg_parser,
    is_interactive,
    load_env_context,
    main_missing,
    print_host_note,
    prompt_optional,
)


def from_database_url(ctx: EnvContext, url: str) -> dict[str, object]:
    resolved, note = ctx.resolve_url(url)
    parsed = urlparse(resolved or url)
    return {
        "host": parsed.hostname or "",
        "port": parsed.port or 5432,
        "user": parsed.username or "",
        "password": parsed.password or "",
        "database": (parsed.path or "/").lstrip("/"),
        "note": note,
    }


def db_config(ctx: EnvContext) -> dict[str, object]:
    url = ctx.get(["DATABASE_URL", "POSTGRES_URL", "PGDATABASE_URL"])
    if url and url.startswith(("postgres://", "postgresql://")):
        return from_database_url(ctx, url)

    raw_host = ctx.get(["DB_HOST", "PGHOST", "POSTGRES_HOST"])
    if not raw_host:
        main_missing("DB_HOST/DATABASE_URL")
    host, note = ctx.resolve_host(raw_host)
    return {
        "host": host,
        "port": int(ctx.get(["DB_PORT", "PGPORT"]) or "5432"),
        "user": ctx.get(["DB_USERNAME", "DB_USER", "PGUSER", "POSTGRES_USER"]) or "postgres",
        "password": ctx.get(["DB_PASSWORD", "DB_PASS", "PGPASSWORD", "POSTGRES_PASSWORD"]) or "",
        "database": ctx.get(["DB_DATABASE", "DB_NAME", "PGDATABASE", "POSTGRES_DB"]) or "postgres",
        "note": note,
    }


def connect(cfg: dict[str, object]):
    return psycopg2.connect(
        host=cfg["host"],
        port=cfg["port"],
        user=cfg["user"],
        password=cfg["password"],
        dbname=cfg["database"],
        connect_timeout=10,
    )


def run_query(conn, sql: str) -> None:
    with conn.cursor() as cur:
        cur.execute(sql)
        if cur.description:
            cols = [d.name for d in cur.description]
            rows = cur.fetchmany(20)
            print(" | ".join(cols))
            print("-" * 40)
            for row in rows:
                print(" | ".join(str(v) for v in row))
        else:
            conn.commit()
            print(f"OK — {cur.rowcount} linha(s) afectada(s)")


def run_interactive(ctx: EnvContext) -> int:
    cfg = db_config(ctx)
    print("PostgreSQL detectado:")
    print(f"  Host: {cfg['host']}:{cfg['port']}")
    print_host_note(cfg.get("note"))
    print(f"  User: {cfg['user']}")
    print(f"  DB:   {cfg['database']}")

    try:
        conn = connect(cfg)
    except Exception as exc:
        print(f"Falha na conexão: {exc}")
        return 1

    print("Conexão OK.")
    with conn.cursor() as cur:
        cur.execute("SELECT version()")
        print(f"Versão: {cur.fetchone()[0]}")

    while True:
        sql = prompt_optional("SQL (vazio=sair)", "")
        if not sql:
            break
        try:
            run_query(conn, sql)
        except Exception as exc:
            conn.rollback()
            print(f"Erro: {exc}")

    conn.close()
    return 0


def main() -> None:
    args = env_arg_parser("PostgreSQL checker").parse_args()
    ctx = load_env_context(Path(args.env))
    conn_type = (ctx.get(["DB_CONNECTION"]) or "").lower()
    has_url = bool(ctx.get(["DATABASE_URL"]))
    if conn_type and conn_type not in ("pgsql", "postgres", "postgresql") and not has_url:
        print(f"SKIP: DB_CONNECTION={conn_type}")
        sys.exit(2)
    if is_interactive():
        sys.exit(run_interactive(ctx))
    try:
        conn = connect(db_config(ctx))
        with conn.cursor() as cur:
            cur.execute("SELECT 1")
            print(f"OK — PostgreSQL respondeu: {cur.fetchone()[0]}")
        conn.close()
        sys.exit(0)
    except Exception as exc:
        print(f"Falha: {exc}")
        sys.exit(1)


if __name__ == "__main__":
    main()
