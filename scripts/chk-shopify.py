#!/usr/bin/env python3
"""Valida acesso Shopify Admin API."""

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


def shopify_config(env: dict[str, str]) -> tuple[str, str, str]:
    token = pick_key(
        env,
        ["SHOPIFY_ACCESS_TOKEN", "SHOPIFY_ADMIN_API_ACCESS_TOKEN", "SHOPIFY_API_PASSWORD"],
    )
    store = pick_key(env, ["SHOPIFY_STORE_URL", "SHOPIFY_SHOP", "SHOPIFY_DOMAIN", "SHOPIFY_STORE"])
    version = pick_key(env, ["SHOPIFY_API_VERSION"]) or "2024-10"
    if not token:
        main_missing("SHOPIFY_ACCESS_TOKEN")
    if not store:
        main_missing("SHOPIFY_STORE_URL")
    store = store.rstrip("/")
    if not store.startswith("http"):
        store = f"https://{store}"
    return store, token, version


def validate(store: str, token: str, version: str) -> tuple[bool, str]:
    url = f"{store}/admin/api/{version}/shop.json"
    r = requests.get(url, headers={"X-Shopify-Access-Token": token}, timeout=DEFAULT_HTTP_TIMEOUT)
    if r.status_code == 401:
        return False, "Token inválido (401)"
    if r.status_code == 404:
        return False, "Loja ou versão API não encontrada (404)"
    if r.status_code != 200:
        return False, f"HTTP {r.status_code}: {r.text[:200]}"
    shop = r.json().get("shop", {})
    return True, f"Loja: {shop.get('name', '?')} — {shop.get('myshopify_domain', '?')}"


def run_batch(env: dict[str, str]) -> int:
    store, token, version = shopify_config(env)
    log_step("Shopify…")
    try:
        ok, msg = run_with_timeout(
            lambda: validate(store, token, version),
            DEFAULT_OPERATION_TIMEOUT,
            "Shopify",
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
    args = env_arg_parser("Shopify checker").parse_args()
    env = load_env_keys(Path(args.env))
    if is_batch_mode(args):
        sys.exit(run_batch(env))
    if is_interactive():
        store, _, _ = shopify_config(env)
        print(f"Shopify — {store}", flush=True)
    sys.exit(run_batch(env))


if __name__ == "__main__":
    main()
