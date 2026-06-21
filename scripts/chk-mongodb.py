#!/usr/bin/env python3
"""Testa conexão MongoDB."""

from __future__ import annotations

import sys
from pathlib import Path
from urllib.parse import urlparse

from pymongo import MongoClient
from pymongo.errors import PyMongoError

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
    print_host_note,
    print_summary,
    prompt_optional,
    resolve_service_host,
    run_with_timeout,
)


def mongo_uri(ctx: EnvContext) -> tuple[str, str | None]:
    url = ctx.get(["MONGODB_URI", "MONGO_URI", "MONGO_URL", "DATABASE_URL"])
    if url and url.startswith("mongodb"):
        resolved, note = ctx.resolve_url(url)
        return resolved or url, note
    raw_host = ctx.get(["DB_HOST", "MONGO_HOST", "MONGODB_HOST"])
    if not raw_host:
        main_missing("MONGODB_URI/MONGO_URL")
    host, note = resolve_service_host(raw_host, ctx.path, ctx.env, ("mongo", "db"))
    port = ctx.get(["MONGO_PORT", "MONGODB_PORT"]) or "27017"
    user = ctx.get(["MONGO_USERNAME", "MONGODB_USERNAME", "DB_USERNAME"])
    password = ctx.get(["MONGO_PASSWORD", "MONGODB_PASSWORD", "DB_PASSWORD"]) or ""
    auth_db = ctx.get(["MONGO_AUTH_DB"]) or "admin"
    if user:
        uri = f"mongodb://{user}:{password}@{host}:{port}/?authSource={auth_db}"
    else:
        uri = f"mongodb://{host}:{port}/"
    return uri, note


def connect(uri: str) -> MongoClient:
    return MongoClient(uri, serverSelectionTimeoutMS=DEFAULT_CONNECT_TIMEOUT * 1000)


MAX_DBS = 20
MAX_COLS = 12
MAX_COLS_SENSITIVE = 25


def introspect_mongo(client: MongoClient) -> tuple[str, str]:
    lines: list[str] = []
    total_size = 0
    total_docs = 0
    db_count = 0
    sensitive_hits: list[SensitiveHit] = []

    try:
        build = client.admin.command("buildInfo")
        lines.append(f"Versão: {build.get('version', '?')}")
    except Exception:
        pass

    db_names = [n for n in client.list_database_names() if n not in ("admin", "local", "config")]

    for name in db_names[:MAX_DBS]:
        stats = client[name].command("dbStats")
        size = int(stats.get("dataSize", 0) or 0) + int(stats.get("indexSize", 0) or 0)
        cols = int(stats.get("collections", 0) or 0)
        objs = int(stats.get("objects", 0) or 0)
        total_size += size
        total_docs += objs
        db_count += 1

        db_tags = classify_sensitive(name, is_db=True)
        flag = f" ⚠[{','.join(db_tags)}]" if db_tags else ""
        lines.append(f"  {name}: {cols} cols · {objs:,} docs · ~{fmt_bytes(size)}{flag}")
        if db_tags:
            sensitive_hits.append(SensitiveHit("db", "", name, objs, db_tags))

        show_cols = bool(db_tags) or is_sensitive(name, is_db=True) or objs >= 5_000
        if show_cols and cols > 0:
            db = client[name]
            limit = MAX_COLS_SENSITIVE if show_cols else MAX_COLS
            for col in db.list_collection_names()[:limit]:
                try:
                    cnt = int(db[col].estimated_document_count())
                except Exception:
                    cnt = 0
                col_tags = classify_sensitive(col)
                if col_tags or show_cols:
                    tflag = f" ⚠[{','.join(col_tags)}]" if col_tags else ""
                    lines.append(f"    · {col}: {cnt:,} docs{tflag}")
                    if col_tags:
                        sensitive_hits.append(SensitiveHit("collection", name, col, cnt, col_tags))

    if len(db_names) > MAX_DBS:
        lines.append(f"  … +{len(db_names) - MAX_DBS} bases")
    lines.append(f"Total: {db_count} DBs · {total_docs:,} docs · ~{fmt_bytes(total_size)}")
    lines.extend(format_sensitive_section(merge_hits(sensitive_hits)))

    summary = build_summary(
        db_count, total_docs, total_size, merge_hits(sensitive_hits), unit="docs"
    )
    return "\n".join(lines), summary


def run_interactive(ctx: EnvContext) -> int:
    uri, note = mongo_uri(ctx)
    host = urlparse(uri).hostname or uri[:40]
    print(f"MongoDB → {host}", flush=True)
    print_host_note(note)
    log_step("A ligar…")
    try:
        client = run_with_timeout(lambda: connect(uri), DEFAULT_OPERATION_TIMEOUT, "MongoDB")
        client.admin.command("ping")
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1
    report, _ = introspect_mongo(client)
    print(report, flush=True)
    while True:
        cmd = prompt_optional("Comando (show dbs / vazio=sair)", "")
        if not cmd:
            break
        if cmd.lower().startswith("show"):
            for n in client.list_database_names():
                print(n, flush=True)
    client.close()
    return 0


def run_batch(ctx: EnvContext) -> int:
    uri, _ = mongo_uri(ctx)
    log_step("MongoDB…")
    try:
        client = run_with_timeout(lambda: connect(uri), DEFAULT_OPERATION_TIMEOUT, "MongoDB")
        run_with_timeout(lambda: client.admin.command("ping"), DEFAULT_OPERATION_TIMEOUT, "MongoDB ping")
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1
    report, summary = introspect_mongo(client)
    print(report, flush=True)
    client.close()
    print_summary(summary)
    return 0


def main() -> None:
    args = env_arg_parser("MongoDB checker").parse_args()
    ctx = load_env_context(Path(args.env))
    conn = (ctx.get(["DB_CONNECTION"]) or "").lower()
    if conn and conn not in ("mongodb", "mongo", ""):
        has_mongo = bool(ctx.get(["MONGODB_URI", "MONGO_URI", "MONGO_URL"]))
        if not has_mongo:
            print(f"SKIP: DB_CONNECTION={conn}")
            sys.exit(2)
    if is_batch_mode(args):
        sys.exit(run_batch(ctx))
    if is_interactive():
        sys.exit(run_interactive(ctx))
    try:
        uri, _ = mongo_uri(ctx)
        client = connect(uri)
        client.admin.command("ping")
        _, summary = introspect_mongo(client)
        client.close()
        print_summary(summary)
        sys.exit(0)
    except PyMongoError as exc:
        print(format_network_error(exc), flush=True)
        sys.exit(1)


if __name__ == "__main__":
    main()
