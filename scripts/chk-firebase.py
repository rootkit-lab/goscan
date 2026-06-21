#!/usr/bin/env python3
"""Valida credenciais Firebase (API key + project)."""

from __future__ import annotations

import sys
from pathlib import Path

import requests

from envutil import (
    DEFAULT_HTTP_TIMEOUT,
    DEFAULT_OPERATION_TIMEOUT,
    env_arg_parser,
    format_network_error,
    is_batch_mode,
    is_interactive,
    load_env_keys,
    log_step,
    main_missing,
    pick_key,
    print_summary,
    run_with_timeout,
)


def firebase_config(env: dict[str, str]) -> tuple[str, str]:
    api_key = pick_key(
        env,
        [
            "FIREBASE_API_KEY",
            "VITE_FIREBASE_API_KEY",
            "REACT_APP_FIREBASE_API_KEY",
            "NEXT_PUBLIC_FIREBASE_API_KEY",
        ],
    )
    project = pick_key(
        env,
        [
            "FIREBASE_PROJECT_ID",
            "VITE_FIREBASE_PROJECT_ID",
            "REACT_APP_FIREBASE_PROJECT_ID",
            "NEXT_PUBLIC_FIREBASE_PROJECT_ID",
        ],
    )
    if not api_key:
        main_missing("FIREBASE_API_KEY / VITE_FIREBASE_API_KEY")
    if not project:
        main_missing("FIREBASE_PROJECT_ID / VITE_FIREBASE_PROJECT_ID")
    return api_key, project


def validate_project(api_key: str, project: str) -> tuple[bool, str]:
    url = f"https://firebase.googleapis.com/v1beta1/projects/{project}"
    r = requests.get(url, params={"key": api_key}, timeout=DEFAULT_HTTP_TIMEOUT)
    if r.status_code == 403:
        return False, "API key sem permissão ou project inválido (403)"
    if r.status_code == 404:
        return False, f"Project {project} não encontrado (404)"
    if r.status_code != 200:
        return False, f"HTTP {r.status_code}: {r.text[:200]}"
    name = r.json().get("displayName") or project
    return True, f"Project OK — {name}"


def run_batch(env: dict[str, str]) -> int:
    api_key, project = firebase_config(env)
    log_step("Firebase…")
    try:
        ok, msg = run_with_timeout(
            lambda: validate_project(api_key, project),
            DEFAULT_OPERATION_TIMEOUT,
            "Firebase",
        )
    except Exception as exc:
        print(format_network_error(exc), flush=True)
        return 1
    if not ok:
        print(msg, flush=True)
        return 1
    print_summary(msg)
    return 0


def main() -> None:
    args = env_arg_parser("Firebase checker").parse_args()
    env = load_env_keys(Path(args.env))
    if is_batch_mode(args):
        sys.exit(run_batch(env))
    if is_interactive():
        _, project = firebase_config(env)
        print(f"Firebase — project {project}", flush=True)
    sys.exit(run_batch(env))


if __name__ == "__main__":
    main()
