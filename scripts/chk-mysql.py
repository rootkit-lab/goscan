#!/usr/bin/env python3
"""Testa conexão MySQL/MariaDB."""

from __future__ import annotations

import sys
from pathlib import Path

import pymysql

from envutil import (
    EnvContext,
    env_arg_parser,
    is_interactive,
    load_env_context,
    main_missing,
    print_host_note,
    prompt_optional,
)


def db_config(ctx: EnvContext) -> dict[str, object]:
    raw_host = ctx.get(["DB_HOST", "MYSQL_HOST", "DATABASE_HOST"])
    if not raw_host:
        main_missing("DB_HOST")
    host, note = ctx.resolve_host(raw_host)
    if not host:
        main_missing("DB_HOST")
    port = int(ctx.get(["DB_PORT", "MYSQL_PORT"]) or "3306")
    user = ctx.get(["DB_USERNAME", "DB_USER", "MYSQL_USER", "DATABASE_USER"])
    if not user:
        main_missing("DB_USERNAME")
    password = ctx.get(["DB_PASSWORD", "DB_PASS", "MYSQL_PASSWORD", "DATABASE_PASSWORD"]) or ""
    database = ctx.get(["DB_DATABASE", "DB_NAME", "MYSQL_DATABASE", "DATABASE_NAME"])
    return {"host": host, "port": port, "user": user, "password": password, "database": database, "note": note}


def connect(cfg: dict[str, object]):
    kwargs = {
        "host": cfg["host"],
        "port": cfg["port"],
        "user": cfg["user"],
        "password": cfg["password"],
        "connect_timeout": 10,
        "charset": "utf8mb4",
    }
    if cfg.get("database"):
        kwargs["database"] = cfg["database"]
    return pymysql.connect(**kwargs)


def run_query(conn, sql: str) -> None:
    with conn.cursor() as cur:
        cur.execute(sql)
        if cur.description:
            cols = [d[0] for d in cur.description]
            rows = cur.fetchmany(20)
            print(" | ".join(cols))
            print("-" * 40)
            for row in rows:
                print(" | ".join(str(v) for v in row))
            if len(rows) == 20:
                print("… (limitado a 20 linhas)")
        else:
            print(f"OK — {cur.rowcount} linha(s) afectada(s)")


def run_interactive(ctx: EnvContext) -> int:
    cfg = db_config(ctx)
    print("MySQL detectado:")
    print(f"  Host: {cfg['host']}:{cfg['port']}")
    print_host_note(cfg.get("note"))
    print(f"  User: {cfg['user']}")
    print(f"  DB:   {cfg.get('database') or '(nenhuma)'}")

    try:
        conn = connect(cfg)
    except Exception as exc:
        print(f"Falha na conexão: {exc}")
        return 1

    print("Conexão OK.")
    with conn.cursor() as cur:
        cur.execute("SELECT VERSION()")
        print(f"Versão: {cur.fetchone()[0]}")

    while True:
        sql = prompt_optional("SQL (vazio=sair)", "")
        if not sql:
            break
        try:
            run_query(conn, sql)
        except Exception as exc:
            print(f"Erro: {exc}")

    conn.close()
    return 0


def main() -> None:
    args = env_arg_parser("MySQL checker").parse_args()
    ctx = load_env_context(Path(args.env))
    conn_type = (ctx.get(["DB_CONNECTION"]) or "mysql").lower()
    if conn_type not in ("mysql", "mysqli", "mariadb", ""):
        print(f"SKIP: DB_CONNECTION={conn_type} (use chk-postgres para pgsql)")
        sys.exit(2)
    if is_interactive():
        sys.exit(run_interactive(ctx))
    try:
        conn = connect(db_config(ctx))
        with conn.cursor() as cur:
            cur.execute("SELECT 1")
            print(f"OK — MySQL respondeu: {cur.fetchone()[0]}")
        conn.close()
        sys.exit(0)
    except Exception as exc:
        print(f"Falha: {exc}")
        sys.exit(1)


if __name__ == "__main__":
    main()
