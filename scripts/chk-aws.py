#!/usr/bin/env python3
"""Valida credenciais AWS (STS + S3 opcional)."""

from __future__ import annotations

import sys
from pathlib import Path

import boto3
from botocore.exceptions import BotoCoreError, ClientError

from envutil import env_arg_parser, is_interactive, load_env_keys, main_missing, pick_key, prompt_optional


def aws_config(env: dict[str, str]) -> dict[str, str]:
    access = pick_key(env, ["AWS_ACCESS_KEY_ID", "AWS_KEY", "S3_ACCESS_KEY"])
    secret = pick_key(env, ["AWS_SECRET_ACCESS_KEY", "AWS_SECRET", "S3_SECRET_KEY"])
    if not access or not secret:
        main_missing("AWS_ACCESS_KEY_ID + AWS_SECRET_ACCESS_KEY")
    region = pick_key(env, ["AWS_DEFAULT_REGION", "AWS_REGION", "S3_REGION"]) or "us-east-1"
    return {"access": access, "secret": secret, "region": region}


def run_interactive(env: dict[str, str]) -> int:
    cfg = aws_config(env)
    print(f"AWS detectado — região {cfg['region']}")

    try:
        session = boto3.Session(
            aws_access_key_id=cfg["access"],
            aws_secret_access_key=cfg["secret"],
            region_name=cfg["region"],
        )
        sts = session.client("sts")
        identity = sts.get_caller_identity()
    except (ClientError, BotoCoreError) as exc:
        print(f"Falha STS: {exc}")
        return 1

    print("Chave válida:")
    print(f"  Account: {identity.get('Account')}")
    print(f"  ARN:     {identity.get('Arn')}")
    print(f"  UserId:  {identity.get('UserId')}")

    list_s3 = prompt_optional("Listar buckets S3? (s/N)", "n").lower()
    if list_s3 in ("s", "sim", "y", "yes"):
        try:
            s3 = session.client("s3")
            resp = s3.list_buckets()
            buckets = [b["Name"] for b in resp.get("Buckets", [])]
            print(f"\nBuckets ({len(buckets)}):")
            for name in buckets[:30]:
                print(f"  - {name}")
            if len(buckets) > 30:
                print("  …")
        except (ClientError, BotoCoreError) as exc:
            print(f"Sem acesso S3: {exc}")

    return 0


def main() -> None:
    args = env_arg_parser("AWS checker").parse_args()
    env = load_env_keys(Path(args.env))
    if is_interactive():
        sys.exit(run_interactive(env))
    try:
        cfg = aws_config(env)
        session = boto3.Session(
            aws_access_key_id=cfg["access"],
            aws_secret_access_key=cfg["secret"],
            region_name=cfg["region"],
        )
        identity = session.client("sts").get_caller_identity()
        print(f"OK — account {identity.get('Account')} arn {identity.get('Arn')}")
        sys.exit(0)
    except (ClientError, BotoCoreError) as exc:
        print(f"Falha: {exc}")
        sys.exit(1)


if __name__ == "__main__":
    main()
