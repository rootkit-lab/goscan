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


def list_models(api_key: str) -> tuple[bool, list[str], str]:
    url = "https://api.openai.com/v1/models"
    headers = {"Authorization": f"Bearer {api_key}"}
    r = requests.get(url, headers=headers, timeout=DEFAULT_HTTP_TIMEOUT)
    if r.status_code != 200:
        return False, [], f"Erro {r.status_code}: {r.text[:200]}"
    models = sorted(m.get("id", "") for m in r.json().get("data", []) if m.get("id"))
    chat_models = [m for m in models if "gpt" in m or "o1" in m or "o3" in m or "o4" in m]
    return True, (chat_models or models[:20]), "OK"


def chat(api_key: str, model: str, prompt: str) -> str:
    url = "https://api.openai.com/v1/chat/completions"
    headers = {"Authorization": f"Bearer {api_key}", "Content-Type": "application/json"}
    r = requests.post(
        url,
        headers=headers,
        json={"model": model, "messages": [{"role": "user", "content": prompt}], "max_tokens": 1024},
        timeout=CHAT_TIMEOUT,
    )
    if r.status_code != 200:
        raise RuntimeError(f"HTTP {r.status_code}: {r.text[:300]}")
    return r.json()["choices"][0]["message"]["content"].strip()


def run_interactive(api_key: str) -> int:
    log_step("A listar modelos OpenAI…")
    try:
        ok, models, msg = run_with_timeout(lambda: list_models(api_key), DEFAULT_OPERATION_TIMEOUT, "OpenAI")
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1
    if not ok:
        print(msg, flush=True)
        return 1
    print(f"Chave válida — {len(models)} modelos disponíveis.", flush=True)
    model = select_from_list("Modelos OpenAI", models)
    if not model:
        return 1
    print(f"\nModelo seleccionado: {model}", flush=True)
    chat_loop(lambda p: run_with_timeout(lambda: chat(api_key, model, p), CHAT_TIMEOUT, "OpenAI chat"), "openai")
    return 0


def main() -> None:
    args = env_arg_parser("OpenAI API checker").parse_args()
    env = load_env_keys(Path(args.env))
    api_key = pick_key(env, ["OPENAI_API_KEY", "OPENAI_KEY"], args.key)
    if not api_key:
        main_missing("OPENAI_API_KEY")
    if is_interactive():
        sys.exit(run_interactive(api_key))
    try:
        ok, models, msg = run_with_timeout(lambda: list_models(api_key), DEFAULT_OPERATION_TIMEOUT, "OpenAI")
        if ok:
            print(f"OK — modelos: {', '.join(models[:5])}", flush=True)
        else:
            print(msg, flush=True)
        sys.exit(0 if ok else 1)
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        sys.exit(1)


if __name__ == "__main__":
    main()
