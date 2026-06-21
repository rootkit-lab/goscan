#!/usr/bin/env python3
"""Valida credenciais AWS (STS + S3 opcional)."""

from __future__ import annotations

import sys
from pathlib import Path

import boto3
from botocore.config import Config
from botocore.exceptions import BotoCoreError, ClientError

from envutil import (
    DEFAULT_HTTP_TIMEOUT,
    DEFAULT_OPERATION_TIMEOUT,
    env_arg_parser,
    format_network_error,
    is_interactive,
    load_env_keys,
    log_step,
    main_missing,
    pick_key,
    prompt_optional,
    run_with_timeout,
)

BOTO_CONFIG = Config(
    connect_timeout=DEFAULT_HTTP_TIMEOUT,
    read_timeout=DEFAULT_HTTP_TIMEOUT,
    retries={"max_attempts": 2, "mode": "standard"},
)


def aws_config(env: dict[str, str]) -> dict[str, str]:
    access = pick_key(env, ["AWS_ACCESS_KEY_ID", "AWS_KEY", "S3_ACCESS_KEY"])
    secret = pick_key(env, ["AWS_SECRET_ACCESS_KEY", "AWS_SECRET", "S3_SECRET_KEY"])
    if not access or not secret:
        main_missing("AWS_ACCESS_KEY_ID + AWS_SECRET_ACCESS_KEY")
    region = pick_key(env, ["AWS_DEFAULT_REGION", "AWS_REGION", "S3_REGION"]) or "us-east-1"
    return {"access": access, "secret": secret, "region": region}


def sts_identity(cfg: dict[str, str]) -> dict:
    session = boto3.Session(
        aws_access_key_id=cfg["access"],
        aws_secret_access_key=cfg["secret"],
        region_name=cfg["region"],
    )
    return session.client("sts", config=BOTO_CONFIG).get_caller_identity()


def run_interactive(env: dict[str, str]) -> int:
    cfg = aws_config(env)
    print(f"AWS detectado — região {cfg['region']}", flush=True)

    log_step("A validar credenciais (STS)…")
    try:
        identity = run_with_timeout(lambda: sts_identity(cfg), DEFAULT_OPERATION_TIMEOUT, "STS AWS")
    except (ClientError, BotoCoreError, Exception) as exc:
        print(format_network_error(exc), flush=True)
        return 1

    print("Chave válida:", flush=True)
    print(f"  Account: {identity.get('Account')}", flush=True)
    print(f"  ARN:     {identity.get('Arn')}", flush=True)
    print(f"  UserId:  {identity.get('UserId')}", flush=True)

    list_s3 = prompt_optional("Listar buckets S3? (s/N)", "n").lower()
    if list_s3 in ("s", "sim", "y", "yes"):
        try:
            log_step("A listar buckets S3…")
            session = boto3.Session(
                aws_access_key_id=cfg["access"],
                aws_secret_access_key=cfg["secret"],
                region_name=cfg["region"],
            )
            resp = run_with_timeout(
                lambda: session.client("s3", config=BOTO_CONFIG).list_buckets(),
                DEFAULT_OPERATION_TIMEOUT,
                "S3 list_buckets",
            )
            buckets = [b["Name"] for b in resp.get("Buckets", [])]
            print(f"\nBuckets ({len(buckets)}):", flush=True)
            for name in buckets[:30]:
                print(f"  - {name}", flush=True)
            if len(buckets) > 30:
                print("  …", flush=True)
        except (ClientError, BotoCoreError, Exception) as exc:
            print(f"Sem acesso S3: {exc}", flush=True)

    return 0


def main() -> None:
    args = env_arg_parser("AWS checker").parse_args()
    env = load_env_keys(Path(args.env))
    if is_interactive():
        sys.exit(run_interactive(env))
    try:
        identity = run_with_timeout(lambda: sts_identity(aws_config(env)), DEFAULT_OPERATION_TIMEOUT, "STS AWS")
        print(f"OK — account {identity.get('Account')} arn {identity.get('Arn')}", flush=True)
        sys.exit(0)
    except (ClientError, BotoCoreError, Exception) as exc:
        print(format_network_error(exc), flush=True)
        sys.exit(1)


if __name__ == "__main__":
    main()
