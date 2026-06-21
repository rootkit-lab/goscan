#!/usr/bin/env python3
"""Valida chave Cohere."""

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
    select_from_list,
)

CHAT_TIMEOUT = 60


def headers(api_key: str) -> dict[str, str]:
    return {"Authorization": f"Bearer {api_key}"}


def list_models(api_key: str) -> tuple[bool, list[str], str]:
    r = requests.get("https://api.cohere.com/v1/models", headers=headers(api_key), timeout=DEFAULT_HTTP_TIMEOUT)
    if r.status_code != 200:
        return False, [], f"Erro {r.status_code}: {r.text[:200]}"
    models = sorted(m.get("name", "") for m in r.json().get("models", []) if m.get("name"))
    return True, models, "OK"


def chat(api_key: str, model: str, prompt: str) -> str:
    r = requests.post(
        "https://api.cohere.com/v1/chat",
        headers={**headers(api_key), "Content-Type": "application/json"},
        json={"model": model, "message": prompt},
        timeout=CHAT_TIMEOUT,
    )
    if r.status_code != 200:
        raise RuntimeError(f"HTTP {r.status_code}: {r.text[:300]}")
    return r.json().get("text", "").strip() or "(resposta vazia)"


def main() -> None:
    args = env_arg_parser("Cohere API checker").parse_args()
    env = load_env_keys(Path(args.env))
    api_key = pick_key(env, ["COHERE_API_KEY"], args.key)
    if not api_key:
        main_missing("COHERE_API_KEY")

    if is_interactive():
        log_step("A listar modelos Cohere…")
        try:
            ok, models, msg = run_with_timeout(lambda: list_models(api_key), DEFAULT_OPERATION_TIMEOUT, "Cohere")
        except Exception as exc:
            print(format_network_error(exc), flush=True)
            sys.exit(1)
        if not ok:
            print(msg, flush=True)
            sys.exit(1)
        model = select_from_list("Modelos Cohere", models)
        if not model:
            sys.exit(1)
        chat_loop(lambda p: run_with_timeout(lambda: chat(api_key, model, p), CHAT_TIMEOUT, "Cohere chat"), "cohere")
        sys.exit(0)

    try:
        ok, models, msg = run_with_timeout(lambda: list_models(api_key), DEFAULT_OPERATION_TIMEOUT, "Cohere")
        if ok:
            print(f"OK — modelos: {', '.join(models[:8])}", flush=True)
        else:
            print(msg, flush=True)
        sys.exit(0 if ok else 1)
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        sys.exit(1)


if __name__ == "__main__":
    main()
