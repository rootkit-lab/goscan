#!/usr/bin/env python3
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
    return {
        "x-api-key": api_key,
        "anthropic-version": "2023-06-01",
        "content-type": "application/json",
    }


def list_models(api_key: str) -> tuple[bool, list[str], str]:
    url = "https://api.anthropic.com/v1/models"
    r = requests.get(url, headers=headers(api_key), timeout=DEFAULT_HTTP_TIMEOUT)
    if r.status_code != 200:
        return False, [], f"Erro {r.status_code}: {r.text[:200]}"
    models = [m.get("id", "") for m in r.json().get("data", []) if m.get("id")]
    if not models:
        models = [
            "claude-sonnet-4-20250514",
            "claude-3-5-haiku-20241022",
            "claude-3-5-sonnet-20241022",
        ]
    return True, models, "OK"


def chat(api_key: str, model: str, prompt: str) -> str:
    url = "https://api.anthropic.com/v1/messages"
    r = requests.post(
        url,
        headers=headers(api_key),
        json={"model": model, "max_tokens": 1024, "messages": [{"role": "user", "content": prompt}]},
        timeout=CHAT_TIMEOUT,
    )
    if r.status_code != 200:
        raise RuntimeError(f"HTTP {r.status_code}: {r.text[:300]}")
    parts = r.json().get("content", [])
    return "".join(p.get("text", "") for p in parts if p.get("type") == "text").strip()


def run_interactive(api_key: str) -> int:
    log_step("A listar modelos Claude…")
    try:
        ok, models, msg = run_with_timeout(lambda: list_models(api_key), DEFAULT_OPERATION_TIMEOUT, "Claude")
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1
    if not ok:
        print(msg, flush=True)
        return 1
    print(f"Chave válida — {len(models)} modelos disponíveis.", flush=True)
    model = select_from_list("Modelos Claude", models)
    if not model:
        return 1
    print(f"\nModelo seleccionado: {model}", flush=True)
    chat_loop(lambda p: run_with_timeout(lambda: chat(api_key, model, p), CHAT_TIMEOUT, "Claude chat"), "claude")
    return 0


def main() -> None:
    args = env_arg_parser("Claude API checker").parse_args()
    env = load_env_keys(Path(args.env))
    api_key = pick_key(env, ["ANTHROPIC_API_KEY", "CLAUDE_API_KEY"], args.key)
    if not api_key:
        main_missing("ANTHROPIC_API_KEY")
    if is_interactive():
        sys.exit(run_interactive(api_key))
    try:
        ok, _, msg = run_with_timeout(lambda: list_models(api_key), DEFAULT_OPERATION_TIMEOUT, "Claude")
        if ok:
            print("OK — chave válida", flush=True)
        else:
            print(msg, flush=True)
        sys.exit(0 if ok else 1)
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        sys.exit(1)


if __name__ == "__main__":
    main()
