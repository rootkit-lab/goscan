#!/usr/bin/env python3
"""Testa conexão Redis."""

from __future__ import annotations

import sys
from pathlib import Path
from urllib.parse import urlparse

import redis

from envutil import (
    EnvContext,
    env_arg_parser,
    is_interactive,
    load_env_context,
    main_missing,
    print_host_note,
    prompt_optional,
)


def redis_config(ctx: EnvContext) -> dict[str, object]:
    url = ctx.get(["REDIS_URL"])
    if url:
        resolved, note = ctx.resolve_url(url)
        return {"url": resolved or url, "note": note}

    raw_host = ctx.get(["REDIS_HOST", "REDIS_URL_HOST"])
    if not raw_host:
        main_missing("REDIS_HOST/REDIS_URL")
    host, note = ctx.resolve_host(raw_host)
    port = int(ctx.get(["REDIS_PORT"]) or "6379")
    password = ctx.get(["REDIS_PASSWORD", "REDIS_PASS"])
    db = int(ctx.get(["REDIS_DB"]) or "0")
    return {"host": host, "port": port, "password": password, "db": db, "note": note}


def connect(cfg: dict[str, object]):
    if cfg.get("url"):
        return redis.from_url(str(cfg["url"]), socket_connect_timeout=10, decode_responses=True)
    return redis.Redis(
        host=str(cfg["host"]),
        port=int(cfg["port"]),
        password=cfg.get("password") or None,
        db=int(cfg.get("db", 0)),
        socket_connect_timeout=10,
        decode_responses=True,
    )


def run_interactive(ctx: EnvContext) -> int:
    cfg = redis_config(ctx)
    if "url" in cfg:
        host = urlparse(str(cfg["url"])).hostname or "?"
        print(f"Redis URL → {host}")
    else:
        print(f"Redis: {cfg['host']}:{cfg['port']} db={cfg.get('db', 0)}")
    print_host_note(cfg.get("note"))

    try:
        client = connect(cfg)
        pong = client.ping()
    except Exception as exc:
        print(f"Falha na conexão: {exc}")
        return 1

    print(f"Conexão OK — PING={pong}")
    info = client.info("server")
    print(f"Versão: {info.get('redis_version', '?')}")

    while True:
        cmd = prompt_optional("Comando (GET chave / KEYS padrão / vazio=sair)", "")
        if not cmd:
            break
        parts = cmd.split(maxsplit=1)
        op = parts[0].upper()
        try:
            if op == "GET" and len(parts) == 2:
                print(client.get(parts[1]))
            elif op == "KEYS" and len(parts) == 2:
                keys = client.keys(parts[1])[:30]
                for k in keys:
                    print(k)
                if len(keys) == 30:
                    print("… (limitado a 30 chaves)")
            elif op == "PING":
                print(client.ping())
            elif op == "DBSIZE":
                print(client.dbsize())
            else:
                print("Comandos: GET <key>, KEYS <pattern>, PING, DBSIZE")
        except Exception as exc:
            print(f"Erro: {exc}")

    return 0


def main() -> None:
    args = env_arg_parser("Redis checker").parse_args()
    ctx = load_env_context(Path(args.env))
    if is_interactive():
        sys.exit(run_interactive(ctx))
    try:
        client = connect(redis_config(ctx))
        print(f"OK — PING={client.ping()}")
        sys.exit(0)
    except Exception as exc:
        print(f"Falha: {exc}")
        sys.exit(1)


if __name__ == "__main__":
    main()
