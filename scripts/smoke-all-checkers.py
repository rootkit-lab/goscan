#!/usr/bin/env python3
"""Executa cada checker do registry uma vez (fixtures seguras, sem email real)."""

from __future__ import annotations

import re
import subprocess
import sys
import tempfile
import textwrap
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
REGISTRY = ROOT / "scripts" / "registry.yaml"
PY = ROOT / "scripts" / ".venv" / "bin" / "python"
FINDINGS = ROOT / "var/findings/by-domain"
TIMEOUT = 90

FIXTURES: dict[str, str] = {
    "chk-openai": "OPENAI_API_KEY=sk-test-invalid\n",
    "chk-gemini": "GEMINI_API_KEY=AIzaSyFakeKeyForSmokeTestOnly\n",
    "chk-groq": "GROQ_API_KEY=gsk_fake_smoke_test\n",
    "chk-claud": "ANTHROPIC_API_KEY=sk-ant-fake-smoke\n",
    "chk-xai": "XAI_API_KEY=xai-fake-smoke\n",
    "chk-mistral": "MISTRAL_API_KEY=mistral-fake-smoke\n",
    "chk-deepseek": "DEEPSEEK_API_KEY=ds-fake-smoke\n",
    "chk-perplexity": "PERPLEXITY_API_KEY=pplx-fake-smoke\n",
    "chk-openrouter": "OPENROUTER_API_KEY=or-fake-smoke\n",
    "chk-together": "TOGETHER_API_KEY=tg-fake-smoke\n",
    "chk-cohere": "COHERE_API_KEY=cohere-fake-smoke\n",
    "chk-replicate": "REPLICATE_API_TOKEN=r8_fake_smoke\n",
    "chk-huggingface": "HF_TOKEN=hf_fake_smoke\n",
    "chk-smtp": textwrap.dedent("""\
        MAIL_HOST=127.0.0.1
        MAIL_PORT=1025
        MAIL_USERNAME=user
        MAIL_PASSWORD=pass
        MAIL_ENCRYPTION=
    """),
    "chk-mysql": textwrap.dedent("""\
        DB_CONNECTION=mysql
        DB_HOST=127.0.0.1
        DB_USERNAME=root
        DB_PASSWORD=pass
        DB_DATABASE=test
    """),
    "chk-postgres": textwrap.dedent("""\
        DB_CONNECTION=pgsql
        DB_HOST=127.0.0.1
        DB_USERNAME=postgres
        DB_PASSWORD=pass
        DB_DATABASE=test
    """),
    "chk-redis": "REDIS_HOST=127.0.0.1\nREDIS_PORT=6379\n",
    "chk-mongodb": "MONGODB_URI=mongodb://127.0.0.1:27017/test\n",
    "chk-memcached": "MEMCACHED_HOST=127.0.0.1\nMEMCACHED_PORT=11211\n",
    "chk-aws": textwrap.dedent("""\
        AWS_ACCESS_KEY_ID=AKIAFAKE000000000000
        AWS_SECRET_ACCESS_KEY=fakeSecretKeyForSmokeTestOnly1234567890
        AWS_DEFAULT_REGION=us-east-1
    """),
    "chk-pusher": textwrap.dedent("""\
        PUSHER_APP_ID=1
        PUSHER_APP_KEY=fakekey
        PUSHER_APP_SECRET=fakesecret
        PUSHER_APP_CLUSTER=mt1
    """),
    "chk-stripe": "STRIPE_SECRET_KEY=sk_test_fake_smoke\n",
    "chk-paystack": "PAYSTACK_SECRET_KEY=sk_test_fake_smoke\n",
    "chk-paypal": textwrap.dedent("""\
        PAYPAL_CLIENT_ID=fake
        PAYPAL_CLIENT_SECRET=fake
    """),
    "chk-razorpay": textwrap.dedent("""\
        RAZORPAY_KEY_ID=rzp_test_fake
        RAZORPAY_KEY_SECRET=fake_secret
    """),
    "chk-sendgrid": "SENDGRID_API_KEY=SG.fake.smoke.test.key\n",
    "chk-twilio": textwrap.dedent("""\
        TWILIO_SID=AC00000000000000000000000000000000
        TWILIO_AUTH_TOKEN=fake_auth_token_smoke_test
    """),
    "chk-supabase": textwrap.dedent("""\
        SUPABASE_URL=https://example.supabase.co
        SUPABASE_SERVICE_KEY=fake_service_key
    """),
    "chk-mailgun": textwrap.dedent("""\
        MAILGUN_DOMAIN=sandbox.example.org
        MAILGUN_SECRET=key-fake-smoke
    """),
    "chk-sentry": "SENTRY_DSN=https://fake@o0.ingest.sentry.io/0\n",
    "chk-nexmo": textwrap.dedent("""\
        NEXMO_KEY=fakekey
        NEXMO_SECRET=fakesecret
    """),
    "chk-firebase": textwrap.dedent("""\
        VITE_FIREBASE_API_KEY=AIzaSyFakeSmokeTestKey
        VITE_FIREBASE_PROJECT_ID=fake-project
    """),
    "chk-paddle": "PADDLE_API_KEY=pdl_fake_smoke\n",
    "chk-flutterwave": "FLW_SECRET_KEY=FLWSECK_TEST-fake_smoke-X\n",
    "chk-hubspot": "HUB_SPOT_API_KEY=pat-fake-smoke-test-token\n",
    "chk-fcm": "FCM_SERVER_KEY=fake_fcm_server_key_smoke\n",
    "chk-telegram": "TELEGRAM_API_TOKEN=123456789:AAFakeSmokeTestTokenForBot\n",
    "chk-shopify": textwrap.dedent("""\
        SHOPIFY_STORE_URL=https://fake-shop.myshopify.com
        SHOPIFY_ACCESS_TOKEN=shpat_fake_smoke
        SHOPIFY_API_VERSION=2024-10
    """),
}

SKIP_REAL = frozenset({"chk-smtp", "chk-sendgrid", "chk-mailgun", "chk-twilio"})

# DB/cache remotos podem bloquear 90s — smoke usa sempre fixture
FORCE_FIXTURE = frozenset({
    "chk-redis", "chk-mysql", "chk-postgres", "chk-mongodb", "chk-memcached",
})


def load_registry() -> list[dict]:
    text = REGISTRY.read_text(encoding="utf-8")
    scripts: list[dict] = []
    for block in text.split("\n  - id:")[1:]:
        sid_m = re.match(r" (\S+)\n", block)
        path_m = re.search(r"^\s+path: (\S+)\n", block, re.M)
        keys_part = block.split("env_keys:", 1)[-1] if "env_keys:" in block else ""
        keys_m = re.findall(r"^\s+- (\S+)\n", keys_part, re.M)
        if sid_m and path_m:
            scripts.append({"id": sid_m.group(1), "path": path_m.group(1), "env_keys": keys_m})
    return scripts


def load_env_keys(path: Path) -> dict[str, str]:
    keys: dict[str, str] = {}
    for line in path.read_text(encoding="utf-8", errors="replace").splitlines():
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        k, _, v = line.partition("=")
        v = v.strip().strip('"').strip("'")
        if v:
            keys[k.strip()] = v
    return keys


def pick_real_env(script: dict) -> Path | None:
    sid = script["id"]
    if sid in SKIP_REAL or sid in FORCE_FIXTURE:
        return None
    env_keys = script.get("env_keys") or []
    for env_file in sorted(FINDINGS.rglob("*.env")):
        keys = load_env_keys(env_file)
        if not any(keys.get(k) for k in env_keys):
            continue
        conn = keys.get("DB_CONNECTION", "").lower()
        if sid == "chk-postgres" and conn in ("mysql", "mysqli", "mariadb"):
            continue
        if sid == "chk-mysql" and conn in ("pgsql", "postgres", "postgresql"):
            continue
        return env_file
    return None


def run_checker(script_path: Path, env_path: Path) -> tuple[int, str]:
    proc = subprocess.run(
        [str(PY), "-u", str(script_path), "--env", str(env_path), "--batch"],
        cwd=ROOT / "scripts",
        capture_output=True,
        text=True,
        timeout=TIMEOUT,
        env={**subprocess.os.environ, "GOSCAN_BATCH": "1"},
    )
    out = (proc.stdout or "") + (proc.stderr or "")
    return proc.returncode, out.strip()


def classify(exit_code: int, output: str) -> str:
    upper = output.upper()
    if "TRACEBACK" in upper or "IMPORTERROR" in upper or "MODULENOTFOUND" in upper:
        return "crash"
    if exit_code == 0 or output.startswith("OK") or "SUMMARY:" in output or output.startswith("SKIP:"):
        return "ok"
    if exit_code in (1, 124) and any(
        x in upper
        for x in (
            "HTTP", "ERRO", "FALHA", "REDE", "TIMEOUT", "INVÁLID", "INVALID",
            "401", "403", "404", "CONNECTION", "ECONNREFUSED", "NAMERESOLUTION",
            "SKIP:", "MISSING", "FALTAM", "CHAVE", "TOKEN", "CREDEN",
            "PERMISSION", "DON'T HAVE",
        )
    ):
        return "ok"
    return "fail"


def main() -> int:
    if not PY.exists():
        print("❌ Falta venv — corra: make scripts-venv", file=sys.stderr)
        return 1

    registry = load_registry()
    fails = 0

    print(f"Smoke test — {len(registry)} checkers\n")

    with tempfile.TemporaryDirectory(prefix="goscan-smoke-") as tmp:
        tmp_path = Path(tmp)
        for script in registry:
            sid = script["id"]
            path = ROOT / script["path"]
            if not path.exists():
                fails += 1
                print(f"❌ {sid:22s} [missing ] ficheiro não encontrado")
                continue

            env_path = pick_real_env(script)
            source = "finding"
            if env_path is None:
                body = FIXTURES.get(sid)
                if not body:
                    fails += 1
                    print(f"❌ {sid:22s} [no-fix ] sem env compatível nem fixture")
                    continue
                env_path = tmp_path / f"{sid}.env"
                env_path.write_text(body)
                source = "fixture"

            try:
                code, out = run_checker(path, env_path)
            except subprocess.TimeoutExpired:
                fails += 1
                print(f"❌ {sid:22s} [{source:7s}] timeout {TIMEOUT}s")
                continue

            status = classify(code, out)
            summary = out.splitlines()[-1][:120] if out else f"exit {code}"
            if status != "ok":
                fails += 1
            mark = "✅" if status == "ok" else "❌"
            print(f"{mark} {sid:22s} [{source:7s}] exit={code:3d}  {summary}")

    print(f"\n{'=' * 60}")
    print(f"Total: {len(registry)}  OK: {len(registry) - fails}  FAIL: {fails}")
    return 1 if fails else 0


if __name__ == "__main__":
    sys.exit(main())
