#!/usr/bin/env python3
"""Valida chave Perplexity."""

from __future__ import annotations

import sys
from pathlib import Path

import requests

from envutil import (
    DEFAULT_HTTP_TIMEOUT,
    DEFAULT_OPERATION_TIMEOUT,
    chat_loop,
    env_arg_parser,
    format_network_error,
    is_interactive,
    load_env_keys,
    log_step,
    main_missing,
    pick_key,
    run_with_timeout,
)

CHAT_TIMEOUT = 60
DEFAULT_MODEL = "sonar"


def chat(api_key: str, prompt: str) -> str:
    r = requests.post(
        "https://api.perplexity.ai/chat/completions",
        headers={"Authorization": f"Bearer {api_key}", "Content-Type": "application/json"},
        json={
            "model": DEFAULT_MODEL,
            "messages": [{"role": "user", "content": prompt}],
            "max_tokens": 256,
        },
        timeout=CHAT_TIMEOUT,
    )
    if r.status_code != 200:
        raise RuntimeError(f"HTTP {r.status_code}: {r.text[:300]}")
    return r.json()["choices"][0]["message"]["content"].strip()


def validate(api_key: str) -> tuple[bool, str]:
    r = requests.post(
        "https://api.perplexity.ai/chat/completions",
        headers={"Authorization": f"Bearer {api_key}", "Content-Type": "application/json"},
        json={
            "model": DEFAULT_MODEL,
            "messages": [{"role": "user", "content": "ping"}],
            "max_tokens": 8,
        },
        timeout=DEFAULT_HTTP_TIMEOUT,
    )
    if r.status_code == 401:
        return False, "Chave inválida (401)"
    if r.status_code != 200:
        return False, f"HTTP {r.status_code}: {r.text[:200]}"
    return True, f"OK — modelo {DEFAULT_MODEL}"


def main() -> None:
    args = env_arg_parser("Perplexity API checker").parse_args()
    env = load_env_keys(Path(args.env))
    api_key = pick_key(env, ["PERPLEXITY_API_KEY", "PPLX_API_KEY"], args.key)
    if not api_key:
        main_missing("PERPLEXITY_API_KEY")

    if is_interactive():
        log_step("A validar Perplexity…")
        try:
            ok, msg = run_with_timeout(lambda: validate(api_key), DEFAULT_OPERATION_TIMEOUT, "Perplexity")
        except Exception as exc:
            print(format_network_error(exc), flush=True)
            sys.exit(1)
        if not ok:
            print(msg, flush=True)
            sys.exit(1)
        print(msg, flush=True)
        chat_loop(lambda p: run_with_timeout(lambda: chat(api_key, p), CHAT_TIMEOUT, "Perplexity chat"), "perplexity")
        sys.exit(0)

    try:
        ok, msg = run_with_timeout(lambda: validate(api_key), DEFAULT_OPERATION_TIMEOUT, "Perplexity")
        print(msg, flush=True)
        sys.exit(0 if ok else 1)
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        sys.exit(1)


if __name__ == "__main__":
    main()
