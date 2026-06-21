#!/usr/bin/env python3
"""Testa conexão Redis."""

from __future__ import annotations

import sys
from pathlib import Path
from urllib.parse import urlparse

import redis

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
    resolve_redis_host,
    run_with_timeout,
    skip_redis_unlikely,
)


def redis_config(ctx: EnvContext) -> dict[str, object]:
    url = ctx.get(["REDIS_URL"])
    if url:
        resolved, note = ctx.resolve_url(url)
        return {"url": resolved or url, "note": note}

    raw_host = ctx.get(["REDIS_HOST", "REDIS_URL_HOST"])
    if not raw_host:
        main_missing("REDIS_HOST/REDIS_URL")
    host, note = resolve_redis_host(raw_host, ctx.path, ctx.env)
    port = int(ctx.get(["REDIS_PORT"]) or "6379")
    password = ctx.get(["REDIS_PASSWORD", "REDIS_PASS"])
    db = int(ctx.get(["REDIS_DB"]) or "0")
    return {"host": host, "port": port, "password": password, "db": db, "note": note}


def connect(cfg: dict[str, object]):
    common = {
        "socket_connect_timeout": DEFAULT_CONNECT_TIMEOUT,
        "socket_timeout": DEFAULT_OPERATION_TIMEOUT,
        "decode_responses": True,
    }
    if cfg.get("url"):
        return redis.from_url(str(cfg["url"]), **common)
    return redis.Redis(
        host=str(cfg["host"]),
        port=int(cfg["port"]),
        password=cfg.get("password") or None,
        db=int(cfg.get("db", 0)),
        **common,
    )


def ping_client(cfg: dict[str, object]):
    client = connect(cfg)
    return client, client.ping()


def introspect_redis(client) -> tuple[str, str]:
    info = client.info()
    mem = int(info.get("used_memory", 0) or 0)
    version = info.get("redis_version", "?")
    dbsize = client.dbsize()
    lines = [
        f"Versão: {version}",
        f"Memória: ~{fmt_bytes(mem)}",
        f"DB actual: {dbsize} keys",
    ]
    dbs_with_keys = 0
    for i in range(16):
        try:
            n = client.execute_command("SELECT", i)
            _ = n
            c = client.dbsize()
            if c > 0:
                dbs_with_keys += 1
                lines.append(f"  db{i}: {c} keys")
        except Exception:
            break
    summary = f"PING OK · {dbsize} keys · ~{fmt_bytes(mem)} · {dbs_with_keys} DBs"
    return "\n".join(lines), summary


def run_interactive(ctx: EnvContext) -> int:
    cfg = redis_config(ctx)
    if "url" in cfg:
        host = urlparse(str(cfg["url"])).hostname or "?"
        print(f"Redis URL → {host}", flush=True)
    else:
        print(f"Redis: {cfg['host']}:{cfg['port']}", flush=True)
    print_host_note(cfg.get("note"))

    log_step(f"A ligar (timeout {DEFAULT_CONNECT_TIMEOUT}s)…")
    try:
        client, pong = run_with_timeout(lambda: ping_client(cfg), DEFAULT_OPERATION_TIMEOUT, "Conexão Redis")
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1

    print(f"Conexão OK — PING={pong}", flush=True)
    report, _ = introspect_redis(client)
    print(report, flush=True)

    while True:
        cmd = prompt_optional("Comando (GET/KEYS/PING/DBSIZE / vazio=sair)", "")
        if not cmd:
            break
        parts = cmd.split(maxsplit=1)
        op = parts[0].upper()
        try:
            if op == "GET" and len(parts) == 2:
                print(client.get(parts[1]), flush=True)
            elif op == "KEYS" and len(parts) == 2:
                for k in client.keys(parts[1])[:30]:
                    print(k, flush=True)
            elif op == "PING":
                print(client.ping(), flush=True)
            elif op == "DBSIZE":
                print(client.dbsize(), flush=True)
        except Exception as exc:
            print(f"Erro: {exc}", flush=True)
    return 0


def run_batch(ctx: EnvContext) -> int:
    cfg = redis_config(ctx)
    log_step("Redis…")
    try:
        client, pong = run_with_timeout(lambda: ping_client(cfg), DEFAULT_OPERATION_TIMEOUT, "Conexão Redis")
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1
    report, summary = introspect_redis(client)
    print(f"PING={pong}", flush=True)
    print(report, flush=True)
    print_summary(summary)
    return 0


def main() -> None:
    args = env_arg_parser("Redis checker").parse_args()
    ctx = load_env_context(Path(args.env))
    if is_batch_mode(args):
        skip_redis_unlikely(ctx)
        sys.exit(run_batch(ctx))
    if is_interactive():
        sys.exit(run_interactive(ctx))
    try:
        client, _ = run_with_timeout(lambda: ping_client(redis_config(ctx)), DEFAULT_OPERATION_TIMEOUT, "Redis")
        _, summary = introspect_redis(client)
        print_summary(summary)
        sys.exit(0)
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        sys.exit(1)


if __name__ == "__main__":
    main()
