#!/usr/bin/env python3
import sys
from pathlib import Path

import requests

from envutil import chat_loop, env_arg_parser, is_interactive, load_env_keys, main_missing, pick_key, select_from_list


def list_models(api_key: str) -> tuple[bool, list[str], str]:
    url = "https://generativelanguage.googleapis.com/v1beta/models"
    r = requests.get(url, params={"key": api_key}, timeout=30)
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
        timeout=60,
    )
    if r.status_code != 200:
        raise RuntimeError(f"HTTP {r.status_code}: {r.text[:300]}")
    data = r.json()
    parts = data.get("candidates", [{}])[0].get("content", {}).get("parts", [])
    text = "".join(p.get("text", "") for p in parts)
    return text.strip() or "(resposta vazia)"


def run_interactive(api_key: str) -> int:
    ok, models, msg = list_models(api_key)
    if not ok:
        print(msg)
        return 1
    print(f"Chave válida — {len(models)} modelos disponíveis.")
    model = select_from_list("Modelos Gemini", models)
    if not model:
        return 1
    print(f"\nModelo seleccionado: {model}")
    chat_loop(lambda p: generate(api_key, model, p), "gemini")
    return 0


def main() -> None:
    args = env_arg_parser("Gemini API checker").parse_args()
    env = load_env_keys(Path(args.env))
    api_key = pick_key(env, ["GEMINI_API_KEY", "GOOGLE_API_KEY", "GOOGLE_GENERATIVE_AI_KEY"], args.key)
    if not api_key:
        main_missing("GEMINI_API_KEY/GOOGLE_API_KEY")
    if is_interactive():
        sys.exit(run_interactive(api_key))
    ok, models, msg = list_models(api_key)
    if ok:
        preview = ", ".join(models[:8])
        print(f"OK — modelos: {preview}")
    else:
        print(msg)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
