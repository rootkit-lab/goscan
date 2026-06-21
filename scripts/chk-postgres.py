#!/usr/bin/env python3
"""Testa conexão PostgreSQL."""

from __future__ import annotations

import sys
from pathlib import Path
from urllib.parse import urlparse

import psycopg2

from dbintrospect import (
    SensitiveHit,
    build_summary,
    classify_sensitive,
    format_sensitive_section,
    is_sensitive,
    merge_hits,
)
from envutil import (
    DEFAULT_CONNECT_TIMEOUT,
    DEFAULT_OPERATION_TIMEOUT,
    EnvContext,
    env_arg_parser,
    fmt_bytes,
    format_network_error,
    is_batch_mode,
    is_interactive,
    load_env_context,
    log_step,
    main_missing,
    network_socket_timeout,
    print_host_note,
    print_summary,
    prompt_optional,
    run_with_timeout,
    skip_private_db_host,
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


def should_skip(ctx: EnvContext) -> str | None:
    conn = (ctx.get(["DB_CONNECTION"]) or "").lower()
    url = ctx.get(["DATABASE_URL", "POSTGRES_URL", "PGDATABASE_URL"]) or ""
    if conn in ("mysql", "mysqli", "mariadb", "mongodb", "mongo", "sqlsrv", "sqlite"):
        return f"DB_CONNECTION={conn}"
    if url.startswith("mysql://"):
        return "DATABASE_URL=mysql"
    if url.startswith(("postgres://", "postgresql://")):
        return None
    if conn in ("pgsql", "postgres", "postgresql"):
        return None
    if ctx.get(["PGHOST", "POSTGRES_HOST", "PGUSER", "POSTGRES_USER"]):
        return None
    if conn and conn not in ("",):
        return f"DB_CONNECTION={conn}"
    if ctx.get(["DB_HOST", "DB_USERNAME"]):
        return "keys DB_* ambíguas (provável MySQL)"
    return None


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
        connect_timeout=DEFAULT_CONNECT_TIMEOUT,
    )


def connect_with_timeout(cfg: dict[str, object]):
    network_socket_timeout(DEFAULT_CONNECT_TIMEOUT)
    return run_with_timeout(lambda: connect(cfg), DEFAULT_OPERATION_TIMEOUT, "Conexão PostgreSQL")


def introspect_postgres(conn, current_db: str = "") -> tuple[str, str]:
    lines: list[str] = []
    total_size = 0
    total_rows = 0
    total_tables = 0
    db_count = 0
    sensitive_hits: list[SensitiveHit] = []

    with conn.cursor() as cur:
        cur.execute("SELECT version()")
        lines.append(f"Versão: {cur.fetchone()[0][:80]}")
        cur.execute(
            """
            SELECT datname, pg_database_size(datname)
            FROM pg_database
            WHERE datistemplate = false
            ORDER BY pg_database_size(datname) DESC
            LIMIT 20
            """
        )
        dbs = cur.fetchall()

    for name, size in dbs:
        size = int(size or 0)
        total_size += size
        db_count += 1
        db_tags = classify_sensitive(name, is_db=True)
        with conn.cursor() as cur:
            cur.execute(
                """
                SELECT COUNT(*)
                FROM information_schema.tables
                WHERE table_catalog = %s
                  AND table_schema NOT IN ('pg_catalog', 'information_schema')
                """,
                (name,),
            )
            tables = int(cur.fetchone()[0] or 0)
            total_tables += tables
        flag = f" ⚠[{','.join(db_tags)}]" if db_tags else ""
        lines.append(f"  {name}: {tables} tables · ~{fmt_bytes(size)}{flag}")
        if db_tags:
            sensitive_hits.append(SensitiveHit("db", "", name, 0, db_tags))

    lines.append(f"Total: {db_count} DBs · {total_tables} tables · ~{fmt_bytes(total_size)}")

    log_step(f"[introspect] tabelas em {current_db or 'actual'}…")
    try:
        with conn.cursor() as cur:
            cur.execute(
                """
                SELECT schemaname, relname, COALESCE(n_live_tup, 0)::bigint
                FROM pg_stat_user_tables
                ORDER BY n_live_tup DESC NULLS LAST
                LIMIT 30
                """
            )
            rows = cur.fetchall()
        if rows:
            lines.append(f"Detalhe · {current_db or 'postgres'}:")
            for schema, rel, nrows in rows:
                nrows = int(nrows or 0)
                total_rows += nrows
                fq = f"{schema}.{rel}"
                tbl_tags = classify_sensitive(rel) or classify_sensitive(fq)
                if tbl_tags or nrows >= 1_000 or is_sensitive(schema, is_db=True):
                    flag = f" ⚠[{','.join(tbl_tags)}]" if tbl_tags else ""
                    lines.append(f"    · {fq}: {nrows:,} rows{flag}")
                    if tbl_tags:
                        sensitive_hits.append(SensitiveHit("table", schema, rel, nrows, tbl_tags))
    except Exception as exc:
        lines.append(f"  (detalhe indisponível: {exc})")

    lines.extend(format_sensitive_section(merge_hits(sensitive_hits)))
    summary = build_summary(db_count, total_rows, total_size, merge_hits(sensitive_hits))
    return "\n".join(lines), summary


def classify_postgres_error(exc: Exception) -> str:
    msg = str(exc).lower()
    if "password authentication failed" in msg or "28p01" in msg:
        return "auth fail"
    if "timeout expired" in msg or "timed out" in msg:
        return "timeout"
    if "connection refused" in msg:
        return "connect refused"
    if "name or service not known" in msg:
        return "dns fail"
    return "connect fail"


def run_query(conn, sql: str) -> None:
    with conn.cursor() as cur:
        cur.execute(sql)
        if cur.description:
            cols = [d.name for d in cur.description]
            rows = cur.fetchmany(20)
            print(" | ".join(cols), flush=True)
            for row in rows:
                print(" | ".join(str(v) for v in row), flush=True)
        else:
            conn.commit()
            print(f"OK — {cur.rowcount} linha(s) afectada(s)", flush=True)


def run_interactive(ctx: EnvContext) -> int:
    cfg = db_config(ctx)
    print("PostgreSQL detectado:", flush=True)
    print(f"  Host: {cfg['host']}:{cfg['port']}", flush=True)
    print_host_note(cfg.get("note"))

    log_step(f"A ligar (timeout {DEFAULT_CONNECT_TIMEOUT}s)…")
    try:
        conn = connect_with_timeout(cfg)
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1

    print("Conexão OK.", flush=True)
    report, _ = introspect_postgres(conn, str(cfg.get("database") or ""))
    print(report, flush=True)

    while True:
        sql = prompt_optional("SQL (vazio=sair)", "")
        if not sql:
            break
        try:
            run_query(conn, sql)
        except Exception as exc:
            conn.rollback()
            print(f"Erro: {exc}", flush=True)

    conn.close()
    return 0


def run_batch(ctx: EnvContext) -> int:
    cfg = db_config(ctx)
    host = str(cfg.get("host") or "")
    if host:
        skip_private_db_host(host, "PGHOST")
    log_step(f"PostgreSQL {cfg['host']}:{cfg['port']}…")
    try:
        conn = connect_with_timeout(cfg)
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        print_summary(classify_postgres_error(exc))
        return 1
    report, summary = introspect_postgres(conn, str(cfg.get("database") or ""))
    print(report, flush=True)
    conn.close()
    print_summary(summary)
    return 0


def main() -> None:
    args = env_arg_parser("PostgreSQL checker").parse_args()
    ctx = load_env_context(Path(args.env))
    skip = should_skip(ctx)
    if skip:
        print(f"SKIP: {skip}")
        sys.exit(2)
    raw_host = ctx.get(["DB_HOST", "PGHOST", "POSTGRES_HOST"])
    if raw_host:
        skip_private_db_host(raw_host, "DB_HOST/PGHOST")
    conn_type = (ctx.get(["DB_CONNECTION"]) or "").lower()
    has_url = bool(ctx.get(["DATABASE_URL"]))
    if conn_type and conn_type not in ("pgsql", "postgres", "postgresql") and not has_url:
        print(f"SKIP: DB_CONNECTION={conn_type}")
        sys.exit(2)
    if is_batch_mode(args):
        sys.exit(run_batch(ctx))
    if is_interactive():
        sys.exit(run_interactive(ctx))
    try:
        conn = connect_with_timeout(db_config(ctx))
        _, summary = introspect_postgres(conn, str(db_config(ctx).get("database") or ""))
        conn.close()
        print_summary(summary)
        sys.exit(0)
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        sys.exit(1)


if __name__ == "__main__":
    main()
