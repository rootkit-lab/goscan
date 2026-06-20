#!/usr/bin/env python3
import sys
from pathlib import Path

import requests

from envutil import chat_loop, env_arg_parser, is_interactive, load_env_keys, main_missing, pick_key, select_from_list


def list_models(api_key: str) -> tuple[bool, list[str], str]:
    url = "https://api.groq.com/openai/v1/models"
    headers = {"Authorization": f"Bearer {api_key}"}
    r = requests.get(url, headers=headers, timeout=30)
    if r.status_code != 200:
        return False, [], f"Erro {r.status_code}: {r.text[:200]}"
    models = [m.get("id", "") for m in r.json().get("data", []) if m.get("id")]
    return True, models, "OK"


def chat(api_key: str, model: str, prompt: str) -> str:
    url = "https://api.groq.com/openai/v1/chat/completions"
    headers = {"Authorization": f"Bearer {api_key}", "Content-Type": "application/json"}
    r = requests.post(
        url,
        headers=headers,
        json={"model": model, "messages": [{"role": "user", "content": prompt}], "max_tokens": 1024},
        timeout=60,
    )
    if r.status_code != 200:
        raise RuntimeError(f"HTTP {r.status_code}: {r.text[:300]}")
    return r.json()["choices"][0]["message"]["content"].strip()


def run_interactive(api_key: str) -> int:
    ok, models, msg = list_models(api_key)
    if not ok:
        print(msg)
        return 1
    print(f"Chave válida — {len(models)} modelos disponíveis.")
    model = select_from_list("Modelos Groq", models)
    if not model:
        return 1
    print(f"\nModelo seleccionado: {model}")
    chat_loop(lambda p: chat(api_key, model, p), "groq")
    return 0


def main() -> None:
    args = env_arg_parser("Groq API checker").parse_args()
    env = load_env_keys(Path(args.env))
    api_key = pick_key(env, ["GROQ_API_KEY"], args.key)
    if not api_key:
        main_missing("GROQ_API_KEY")
    if is_interactive():
        sys.exit(run_interactive(api_key))
    ok, models, msg = list_models(api_key)
    if ok:
        print(f"OK — modelos: {', '.join(models[:8])}")
    else:
        print(msg)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
