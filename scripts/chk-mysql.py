#!/usr/bin/env python3
"""Testa conexão MySQL/MariaDB."""

from __future__ import annotations

import sys
from pathlib import Path

import pymysql

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
    resolve_db_hosts,
    run_with_timeout,
    skip_private_db_host,
)

SYSTEM_DBS = frozenset({"information_schema", "performance_schema", "mysql", "sys"})


def classify_mysql_error(exc: Exception) -> str:
    msg = str(exc).lower()
    if "access denied" in msg or "1045" in msg:
        return "auth fail"
    if "not allowed to connect" in msg or "1130" in msg:
        return "host denied"
    if "name or service not known" in msg or "errno -2" in msg or "getaddrinfo" in msg:
        return "dns fail"
    if "timed out" in msg or "timeout" in msg or "2003" in msg:
        return "timeout"
    if "refused" in msg or "errno 111" in msg or "errno 113" in msg:
        return "connect refused"
    return "connect fail"


def db_config(ctx: EnvContext) -> dict[str, object]:
    raw_host = ctx.get(["DB_HOST", "MYSQL_HOST", "DATABASE_HOST"])
    if not raw_host:
        main_missing("DB_HOST")
    hosts = resolve_db_hosts(raw_host, ctx.path, ctx.env)
    if not hosts:
        main_missing("DB_HOST")
    host, note = hosts[0]
    return {
        "hosts": hosts,
        "host": host,
        "port": int(ctx.get(["DB_PORT", "MYSQL_PORT"]) or "3306"),
        "user": ctx.get(["DB_USERNAME", "DB_USER", "MYSQL_USER", "DATABASE_USER"]),
        "password": ctx.get(["DB_PASSWORD", "DB_PASS", "MYSQL_PASSWORD", "DATABASE_PASSWORD"]) or "",
        "database": ctx.get(["DB_DATABASE", "DB_NAME", "MYSQL_DATABASE", "DATABASE_NAME"]),
        "note": note,
    }


def connect(cfg: dict[str, object], host: str):
    kwargs = {
        "host": host,
        "port": cfg["port"],
        "user": cfg["user"],
        "password": cfg["password"],
        "connect_timeout": DEFAULT_CONNECT_TIMEOUT,
        "read_timeout": DEFAULT_OPERATION_TIMEOUT,
        "write_timeout": DEFAULT_OPERATION_TIMEOUT,
        "charset": "utf8mb4",
    }
    if cfg.get("database"):
        kwargs["database"] = cfg["database"]
    return pymysql.connect(**kwargs)


def connect_with_fallback(cfg: dict[str, object]):
    network_socket_timeout(DEFAULT_CONNECT_TIMEOUT)
    hosts: list[tuple[str, str | None]] = cfg.get("hosts") or [(str(cfg["host"]), cfg.get("note"))]
    last_exc: Exception | None = None
    for host, note in hosts:
        log_step(f"[connect] MySQL {host}:{cfg['port']}…")
        if note:
            print_host_note(note)
        try:
            return run_with_timeout(lambda h=host: connect(cfg, h), DEFAULT_OPERATION_TIMEOUT, "Conexão MySQL")
        except Exception as exc:
            last_exc = exc
            log_step(f"[connect] falhou: {exc}")
    if last_exc:
        raise last_exc
    raise RuntimeError("sem hosts MySQL")


SYSTEM_DBS = frozenset({"information_schema", "performance_schema", "mysql", "sys"})
MAX_DBS = 20
MAX_TABLES_PER_DB = 12
MAX_TABLES_SENSITIVE_DB = 25


def _table_rows(cur, db: str, limit: int) -> list[tuple[str, int, int]]:
    cur.execute(
        """
        SELECT table_name,
               COALESCE(table_rows, 0),
               COALESCE(data_length, 0) + COALESCE(index_length, 0)
        FROM information_schema.tables
        WHERE table_schema = %s AND table_type = 'BASE TABLE'
        ORDER BY table_rows DESC, table_name ASC
        LIMIT %s
        """,
        (db, limit),
    )
    return [(str(r[0]), int(r[1] or 0), int(r[2] or 0)) for r in cur.fetchall()]


def introspect_mysql(conn) -> tuple[str, str]:
    lines: list[str] = []
    total_size = 0
    total_rows = 0
    db_count = 0
    sensitive_hits: list[SensitiveHit] = []

    log_step("[introspect] VERSION()…")
    with conn.cursor() as cur:
        cur.execute("SELECT VERSION()")
        version = cur.fetchone()[0]
        lines.append(f"Versão: {version}")
        cur.execute("SHOW DATABASES")
        dbs = [row[0] for row in cur.fetchall() if row[0] not in SYSTEM_DBS]

    for db in dbs[:MAX_DBS]:
        db_tags = classify_sensitive(db, is_db=True)
        with conn.cursor() as cur:
            cur.execute(
                """
                SELECT COUNT(*) AS n,
                       COALESCE(SUM(data_length + index_length), 0) AS sz,
                       COALESCE(SUM(table_rows), 0) AS row_cnt
                FROM information_schema.tables WHERE table_schema = %s
                """,
                (db,),
            )
            n, sz, row_cnt = cur.fetchone()
            n = int(n or 0)
            sz = int(sz or 0)
            rows = int(row_cnt or 0)
            total_size += sz
            total_rows += rows
            db_count += 1

            flag = ""
            if db_tags:
                flag = f" ⚠[{','.join(db_tags)}]"
                sensitive_hits.append(SensitiveHit("db", "", db, rows, db_tags))

            lines.append(f"  {db}: {n} tables · {rows:,} rows · ~{fmt_bytes(sz)}{flag}")

            show_tables = db_tags or is_sensitive(db, is_db=True)
            limit = MAX_TABLES_SENSITIVE_DB if show_tables else MAX_TABLES_PER_DB
            tables = _table_rows(cur, db, limit)
            for tbl, trows, tsz in tables:
                tbl_tags = classify_sensitive(tbl)
                if tbl_tags or show_tables or trows >= 10_000:
                    tflag = f" ⚠[{','.join(tbl_tags)}]" if tbl_tags else ""
                    lines.append(f"    · {tbl}: {trows:,} rows · ~{fmt_bytes(tsz)}{tflag}")
                    if tbl_tags:
                        sensitive_hits.append(SensitiveHit("table", db, tbl, trows, tbl_tags))

    if len(dbs) > MAX_DBS:
        lines.append(f"  … +{len(dbs) - MAX_DBS} bases")
    lines.append(f"Total: {db_count} DBs · {total_rows:,} rows · ~{fmt_bytes(total_size)}")
    lines.extend(format_sensitive_section(merge_hits(sensitive_hits)))

    summary = build_summary(db_count, total_rows, total_size, merge_hits(sensitive_hits))
    return "\n".join(lines), summary


def run_query(conn, sql: str) -> None:
    with conn.cursor() as cur:
        cur.execute(sql)
        if cur.description:
            cols = [d[0] for d in cur.description]
            rows = cur.fetchmany(20)
            print(" | ".join(cols), flush=True)
            print("-" * 40, flush=True)
            for row in rows:
                print(" | ".join(str(v) for v in row), flush=True)
        else:
            print(f"OK — {cur.rowcount} linha(s) afectada(s)", flush=True)


def run_interactive(ctx: EnvContext) -> int:
    cfg = db_config(ctx)
    if not cfg["user"]:
        main_missing("DB_USERNAME")
    print("MySQL detectado:", flush=True)
    print(f"  Host: {cfg['host']}:{cfg['port']}", flush=True)
    print_host_note(cfg.get("note"))
    print(f"  User: {cfg['user']}", flush=True)

    try:
        conn = connect_with_fallback(cfg)
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1

    print("Conexão OK.", flush=True)
    report, _ = introspect_mysql(conn)
    print(report, flush=True)

    while True:
        sql = prompt_optional("SQL (vazio=sair)", "")
        if not sql:
            break
        try:
            log_step("A executar…")
            run_query(conn, sql)
        except Exception as exc:
            print(f"Erro: {exc}", flush=True)

    conn.close()
    return 0


def run_batch(ctx: EnvContext) -> int:
    raw_host = ctx.get(["DB_HOST", "MYSQL_HOST", "DATABASE_HOST"])
    skip_private_db_host(raw_host)
    cfg = db_config(ctx)
    if not cfg["user"]:
        main_missing("DB_USERNAME")
    try:
        conn = connect_with_fallback(cfg)
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        print_summary(classify_mysql_error(exc))
        return 1
    report, summary = introspect_mysql(conn)
    print(report, flush=True)
    conn.close()
    print_summary(summary)
    return 0


def main() -> None:
    args = env_arg_parser("MySQL checker").parse_args()
    ctx = load_env_context(Path(args.env))
    conn_type = (ctx.get(["DB_CONNECTION"]) or "mysql").lower()
    if conn_type not in ("mysql", "mysqli", "mariadb", ""):
        print(f"SKIP: DB_CONNECTION={conn_type} (use chk-postgres para pgsql)")
        sys.exit(2)
    raw_host = ctx.get(["DB_HOST", "MYSQL_HOST", "DATABASE_HOST"])
    skip_private_db_host(raw_host)
    if is_batch_mode(args):
        sys.exit(run_batch(ctx))
    if is_interactive():
        sys.exit(run_interactive(ctx))
    try:
        conn = connect_with_fallback(db_config(ctx))
        _, summary = introspect_mysql(conn)
        conn.close()
        print_summary(summary)
        sys.exit(0)
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        sys.exit(1)


if __name__ == "__main__":
    main()
