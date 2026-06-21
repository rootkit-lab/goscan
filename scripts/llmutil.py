#!/usr/bin/env python3
"""Helpers partilhados para checkers de APIs LLM."""

from __future__ import annotations

import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Callable

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


@dataclass(frozen=True)
class OpenAIProvider:
    label: str
    env_keys: tuple[str, ...]
    base_url: str


def bearer_headers(api_key: str, extra: dict[str, str] | None = None) -> dict[str, str]:
    h = {"Authorization": f"Bearer {api_key}"}
    if extra:
        h.update(extra)
    return h


def list_models_openai(api_key: str, base_url: str) -> tuple[bool, list[str], str]:
    url = f"{base_url.rstrip('/')}/models"
    r = requests.get(url, headers=bearer_headers(api_key), timeout=DEFAULT_HTTP_TIMEOUT)
    if r.status_code != 200:
        return False, [], f"Erro {r.status_code}: {r.text[:200]}"
    models = sorted(m.get("id", "") for m in r.json().get("data", []) if m.get("id"))
    return True, models, "OK"


def chat_openai(api_key: str, base_url: str, model: str, prompt: str) -> str:
    url = f"{base_url.rstrip('/')}/chat/completions"
    r = requests.post(
        url,
        headers={**bearer_headers(api_key), "Content-Type": "application/json"},
        json={"model": model, "messages": [{"role": "user", "content": prompt}], "max_tokens": 1024},
        timeout=CHAT_TIMEOUT,
    )
    if r.status_code != 200:
        raise RuntimeError(f"HTTP {r.status_code}: {r.text[:300]}")
    return r.json()["choices"][0]["message"]["content"].strip()


def run_openai_provider(
    provider: OpenAIProvider,
    *,
    model_filter: Callable[[list[str]], list[str]] | None = None,
    default_models: list[str] | None = None,
) -> None:
    args = env_arg_parser(f"{provider.label} API checker").parse_args()
    env = load_env_keys(Path(args.env))
    api_key = pick_key(env, list(provider.env_keys), args.key)
    if not api_key:
        main_missing(" / ".join(provider.env_keys))

    def validate() -> tuple[bool, list[str], str]:
        ok, models, msg = list_models_openai(api_key, provider.base_url)
        if ok and model_filter:
            filtered = model_filter(models)
            if filtered:
                models = filtered
        if ok and not models and default_models:
            models = default_models
        return ok, models, msg

    slug = provider.label.lower().replace(" ", "-")

    if is_interactive():
        log_step(f"A listar modelos {provider.label}…")
        try:
            ok, models, msg = run_with_timeout(validate, DEFAULT_OPERATION_TIMEOUT, provider.label)
        except Exception as exc:
            print(format_network_error(exc), flush=True)
            sys.exit(1)
        if not ok:
            print(msg, flush=True)
            sys.exit(1)
        print(f"Chave válida — {len(models)} modelos disponíveis.", flush=True)
        model = select_from_list(f"Modelos {provider.label}", models)
        if not model:
            sys.exit(1)
        print(f"\nModelo seleccionado: {model}", flush=True)
        chat_loop(
            lambda p: run_with_timeout(
                lambda: chat_openai(api_key, provider.base_url, model, p),
                CHAT_TIMEOUT,
                f"{provider.label} chat",
            ),
            slug,
        )
        sys.exit(0)

    try:
        ok, models, msg = run_with_timeout(validate, DEFAULT_OPERATION_TIMEOUT, provider.label)
        if ok:
            preview = ", ".join(models[:8]) if models else "chave válida"
            print(f"OK — {preview}", flush=True)
        else:
            print(msg, flush=True)
        sys.exit(0 if ok else 1)
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        sys.exit(1)


def run_simple_bearer_check(
    label: str,
    env_keys: list[str],
    url: str,
    *,
    ok_summary: Callable[[requests.Response], str] | None = None,
) -> None:
    args = env_arg_parser(f"{label} checker").parse_args()
    env = load_env_keys(Path(args.env))
    token = pick_key(env, env_keys, args.key)
    if not token:
        main_missing(" / ".join(env_keys))

    def fetch() -> tuple[bool, str]:
        r = requests.get(url, headers=bearer_headers(token), timeout=DEFAULT_HTTP_TIMEOUT)
        if r.status_code == 401:
            return False, "Token inválido (401)"
        if r.status_code != 200:
            return False, f"HTTP {r.status_code}: {r.text[:200]}"
        if ok_summary:
            return True, ok_summary(r)
        return True, "OK — credencial válida"

    if is_interactive():
        log_step(f"A validar {label}…")
    try:
        ok, msg = run_with_timeout(fetch, DEFAULT_OPERATION_TIMEOUT, label)
        print(msg, flush=True)
        sys.exit(0 if ok else 1)
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        sys.exit(1)
