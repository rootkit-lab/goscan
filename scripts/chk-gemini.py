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
    url = "https://generativelanguage.googleapis.com/v1beta/models"
    r = requests.get(url, params={"key": api_key}, timeout=DEFAULT_HTTP_TIMEOUT)
    if r.status_code != 200:
        return False, [], f"Erro {r.status_code}: {r.text[:200]}"
    models = r.json().get("models", [])
    names = [
        m.get("name", "").replace("models/", "")
        for m in models
        if "generateContent" in (m.get("supportedGenerationMethods") or [])
    ]
    if not names:
        return False, [], "Nenhum modelo com generateContent"
    return True, names, "OK"


def generate(api_key: str, model: str, prompt: str) -> str:
    url = f"https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent"
    r = requests.post(
        url,
        params={"key": api_key},
        json={"contents": [{"parts": [{"text": prompt}]}]},
        timeout=CHAT_TIMEOUT,
    )
    if r.status_code != 200:
        raise RuntimeError(f"HTTP {r.status_code}: {r.text[:300]}")
    data = r.json()
    parts = data.get("candidates", [{}])[0].get("content", {}).get("parts", [])
    text = "".join(p.get("text", "") for p in parts)
    return text.strip() or "(resposta vazia)"


def run_interactive(api_key: str) -> int:
    log_step("A listar modelos Gemini…")
    try:
        ok, models, msg = run_with_timeout(lambda: list_models(api_key), DEFAULT_OPERATION_TIMEOUT, "Gemini")
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1
    if not ok:
        print(msg, flush=True)
        return 1
    print(f"Chave válida — {len(models)} modelos disponíveis.", flush=True)
    model = select_from_list("Modelos Gemini", models)
    if not model:
        return 1
    print(f"\nModelo seleccionado: {model}", flush=True)
    chat_loop(
        lambda p: run_with_timeout(lambda: generate(api_key, model, p), CHAT_TIMEOUT, "Gemini"),
        "gemini",
    )
    return 0


def main() -> None:
    args = env_arg_parser("Gemini API checker").parse_args()
    env = load_env_keys(Path(args.env))
    api_key = pick_key(env, ["GEMINI_API_KEY", "GOOGLE_API_KEY", "GOOGLE_GENERATIVE_AI_KEY"], args.key)
    if not api_key:
        main_missing("GEMINI_API_KEY/GOOGLE_API_KEY")
    if is_interactive():
        sys.exit(run_interactive(api_key))
    try:
        ok, models, msg = run_with_timeout(lambda: list_models(api_key), DEFAULT_OPERATION_TIMEOUT, "Gemini")
        if ok:
            preview = ", ".join(models[:8])
            print(f"OK — modelos: {preview}", flush=True)
        else:
            print(msg, flush=True)
        sys.exit(0 if ok else 1)
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        sys.exit(1)


if __name__ == "__main__":
    main()
