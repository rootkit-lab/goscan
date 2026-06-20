#!/usr/bin/env python3
"""Valida credenciais Pusher e permite disparar evento de teste."""

from __future__ import annotations

import json
import sys
from pathlib import Path

import pusher

from envutil import env_arg_parser, is_interactive, load_env_keys, main_missing, pick_key, prompt_optional, prompt_required


def pusher_config(env: dict[str, str]) -> dict[str, str]:
    app_id = pick_key(env, ["PUSHER_APP_ID", "PUSHER_APPID"])
    key = pick_key(env, ["PUSHER_APP_KEY", "MIX_PUSHER_APP_KEY", "VITE_PUSHER_APP_KEY"])
    secret = pick_key(env, ["PUSHER_APP_SECRET"])
    cluster = pick_key(env, ["PUSHER_APP_CLUSTER", "MIX_PUSHER_APP_CLUSTER", "VITE_PUSHER_APP_CLUSTER"]) or "mt1"
    if not app_id or not key or not secret:
        main_missing("PUSHER_APP_ID + PUSHER_APP_KEY + PUSHER_APP_SECRET")
    return {"app_id": app_id, "key": key, "secret": secret, "cluster": cluster}


def make_client(cfg: dict[str, str]) -> pusher.Pusher:
    return pusher.Pusher(
        app_id=cfg["app_id"],
        key=cfg["key"],
        secret=cfg["secret"],
        cluster=cfg["cluster"],
        ssl=True,
    )


def validate_client(cfg: dict[str, str]) -> tuple[bool, str]:
    try:
        info = make_client(cfg).channels_info()
        channels = info.get("channels") or {}
        return True, f"API OK — {len(channels)} canal(is) activo(s)"
    except Exception as exc:
        return False, str(exc)


def run_interactive(env: dict[str, str]) -> int:
    cfg = pusher_config(env)
    print("Pusher detectado:")
    print(f"  App ID:  {cfg['app_id']}")
    print(f"  Key:     {cfg['key']}")
    print(f"  Cluster: {cfg['cluster']}")

    ok, msg = validate_client(cfg)
    if not ok:
        print(msg)
        return 1
    print("Credenciais válidas.")

    trigger = prompt_optional("Disparar evento de teste? (s/N)", "n").lower()
    if trigger not in ("s", "sim", "y", "yes"):
        return 0

    channel = prompt_required("Canal")
    event = prompt_required("Evento")
    data_raw = prompt_optional("Payload JSON", '{"message":"goscan test"}')
    try:
        payload = json.loads(data_raw)
    except json.JSONDecodeError as exc:
        print(f"JSON inválido: {exc}")
        return 1

    client = make_client(cfg)
    try:
        client.trigger(channel, event, payload)
        print(f"OK — evento '{event}' enviado em '{channel}'")
    except Exception as exc:
        print(f"Falha ao disparar: {exc}")
        return 1
    return 0


def main() -> None:
    args = env_arg_parser("Pusher checker").parse_args()
    env = load_env_keys(Path(args.env))
    if is_interactive():
        sys.exit(run_interactive(env))
    ok, msg = validate_client(pusher_config(env))
    print(msg)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
