#!/usr/bin/env python3
"""Testa conexão Memcached."""

from __future__ import annotations

import sys
from pathlib import Path

from pymemcache.client.base import Client

from envutil import (
    DEFAULT_CONNECT_TIMEOUT,
    DEFAULT_OPERATION_TIMEOUT,
    EnvContext,
    env_arg_parser,
    format_network_error,
    is_batch_mode,
    is_interactive,
    load_env_context,
    log_step,
    main_missing,
    print_host_note,
    print_summary,
    prompt_optional,
    run_with_timeout,
)


def memcached_config(ctx: EnvContext) -> tuple[str, int, str | None]:
    raw = ctx.get(["MEMCACHED_HOST", "MEMCACHE_SERVER", "CACHE_HOST"])
    if not raw:
        main_missing("MEMCACHED_HOST")
    host = raw.strip()
    port = int(ctx.get(["MEMCACHED_PORT", "MEMCACHE_PORT"]) or "11211")
    if ":" in host and not host.startswith("/"):
        h, _, p = host.rpartition(":")
        if p.isdigit():
            host, port = h, int(p)
    note = None
    driver = ctx.get(["CACHE_DRIVER", "CACHE_STORE"])
    if driver and driver.lower() not in ("memcached", "memcache", ""):
        note = f"CACHE_DRIVER={driver}"
    resolved, resolve_note = ctx.resolve_host(host)
    if resolve_note:
        note = resolve_note if not note else f"{note}; {resolve_note}"
    return resolved or host, port, note


def probe(host: str, port: int) -> dict:
    client = Client(
        (host, port),
        connect_timeout=DEFAULT_CONNECT_TIMEOUT,
        timeout=DEFAULT_OPERATION_TIMEOUT,
    )
    key = "goscan_ping"
    if not client.set(key, b"1", expire=30):
        raise RuntimeError("set falhou")
    val = client.get(key)
    if val != b"1":
        raise RuntimeError("get falhou")
    stats = client.stats()
    client.close()
    return stats if isinstance(stats, dict) else {}


def run_batch(ctx: EnvContext) -> int:
    host, port, note = memcached_config(ctx)
    log_step(f"Memcached {host}:{port}…")
    print_host_note(note)
    try:
        stats = run_with_timeout(lambda: probe(host, port), DEFAULT_OPERATION_TIMEOUT, "Memcached")
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1
    items = len(stats) if stats else 0
    summary = f"PING OK · {items} stats"
    print_summary(summary)
    return 0


def run_interactive(ctx: EnvContext) -> int:
    host, port, note = memcached_config(ctx)
    print(f"Memcached: {host}:{port}", flush=True)
    print_host_note(note)
    log_step("A ligar…")
    try:
        stats = run_with_timeout(lambda: probe(host, port), DEFAULT_OPERATION_TIMEOUT, "Memcached")
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1
    print("Conexão OK.", flush=True)
    if stats:
        for k in sorted(stats.keys())[:15]:
            print(f"  {k}: {stats[k]}", flush=True)
    while True:
        cmd = prompt_optional("stats / vazio=sair", "")
        if not cmd:
            break
        if cmd.lower() == "stats":
            client = Client((host, port), connect_timeout=DEFAULT_CONNECT_TIMEOUT, timeout=DEFAULT_OPERATION_TIMEOUT)
            for k, v in sorted(client.stats().items())[:20]:
                print(f"  {k}: {v}", flush=True)
            client.close()
    return 0


def main() -> None:
    args = env_arg_parser("Memcached checker").parse_args()
    ctx = load_env_context(Path(args.env))
    driver = (ctx.get(["CACHE_DRIVER", "CACHE_STORE"]) or "").lower()
    if driver and driver not in ("memcached", "memcache", ""):
        print(f"SKIP: CACHE_DRIVER={driver}")
        sys.exit(2)
    if is_batch_mode(args):
        sys.exit(run_batch(ctx))
    if is_interactive():
        sys.exit(run_interactive(ctx))
    sys.exit(run_batch(ctx))


if __name__ == "__main__":
    main()
