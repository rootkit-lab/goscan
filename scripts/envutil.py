#!/usr/bin/env python3
"""Utilitários partilhados para checkers."""

from __future__ import annotations

import argparse
import ipaddress
import os
import re
import secrets
import socket
import sys
from concurrent.futures import ThreadPoolExecutor, TimeoutError as FuturesTimeout
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from urllib.parse import urlparse, urlunparse

LOCAL_HOSTS = frozenset({"127.0.0.1", "localhost", "::1", "0.0.0.0", "[::1]"})
LOCAL_ALIASES = frozenset({
    "mysql", "mariadb", "pgsql", "postgres", "postgresql", "redis", "memcached",
    "mailhog", "mailpit", "db", "database", "mongodb", "mongo", "elasticsearch",
    "host.docker.internal",
})
URL_HINT_KEYS = (
    "APP_URL", "API_URL", "SITE_URL", "WEB_URL", "FRONTEND_URL", "BACKEND_URL",
    "ADMIN_URL", "CANONICAL_URL", "ASSET_URL", "MIX_URL", "VITE_APP_URL",
    "NEXT_PUBLIC_APP_URL", "REACT_APP_API_URL", "SANCTUM_STATEFUL_DOMAINS",
)
URL_IN_VALUE_RE = re.compile(r"https?://[^\s\"'<>]+", re.I)

DEFAULT_CONNECT_TIMEOUT = 15
DEFAULT_HTTP_TIMEOUT = 20
DEFAULT_OPERATION_TIMEOUT = 25

STANDARD_SMTP_PORTS = frozenset({25, 465, 587})

GOSCAN_TEST_EMAIL = "rootmasters@proton.me"


def log_step(msg: str) -> None:
    print(msg, flush=True)


def run_with_timeout(fn, timeout_sec: int = DEFAULT_OPERATION_TIMEOUT, label: str = "Operação"):
    """Executa fn num thread; falha com TimeoutError se exceder o limite."""
    with ThreadPoolExecutor(max_workers=1) as pool:
        fut = pool.submit(fn)
        try:
            return fut.result(timeout=timeout_sec)
        except FuturesTimeout as exc:
            raise TimeoutError(f"{label} excedeu {timeout_sec}s") from exc


def network_socket_timeout(seconds: int = DEFAULT_CONNECT_TIMEOUT) -> None:
    socket.setdefaulttimeout(seconds)


def format_timeout(label: str, seconds: int) -> str:
    return f"{label} excedeu {seconds}s — host inacessível, firewall ou credenciais incorrectas?"


def format_network_error(exc: Exception) -> str:
    if isinstance(exc, TimeoutError):
        return str(exc)
    if isinstance(exc, OSError):
        return f"Rede: {exc}"
    return f"Falha: {exc}"


def is_batch_mode(args=None) -> bool:
    if os.environ.get("GOSCAN_BATCH", "").strip().lower() in ("1", "true", "yes"):
        return True
    if args is not None and getattr(args, "batch", False):
        return True
    return False


def batch_or_interactive(args) -> bool:
    return is_interactive() and not is_batch_mode(args)


def random_email_content(domain: str) -> tuple[str, str]:
    token = secrets.token_hex(4)
    ts = datetime.now(timezone.utc).strftime("%Y-%m-%d %H:%M UTC")
    dom = domain or "unknown"
    subject = f"goscan {dom} {token}"
    body = (
        f"Teste automático goscan\n\n"
        f"domínio: {dom}\n"
        f"hora: {ts}\n"
        f"token: {token}\n"
    )
    return subject, body


def print_summary(msg: str) -> None:
    print(f"SUMMARY: {msg}", flush=True)


def fmt_bytes(n: int) -> str:
    if n < 1024:
        return f"{n} B"
    if n < 1024 * 1024:
        return f"{n / 1024:.1f} KB"
    if n < 1024 * 1024 * 1024:
        return f"{n / (1024 * 1024):.1f} MB"
    return f"{n / (1024 * 1024 * 1024):.2f} GB"


@dataclass
class EnvContext:
    path: Path
    env: dict[str, str]

    def get(self, candidates: list[str], override: str = "") -> str | None:
        return pick_key(self.env, candidates, override)

    def resolve_host(self, raw: str | None) -> tuple[str | None, str | None]:
        return resolve_remote_host(raw, self.path, self.env)

    def resolve_url(self, raw: str | None) -> tuple[str | None, str | None]:
        return resolve_remote_url(raw, self.path, self.env)


def load_env_keys(path: Path) -> dict[str, str]:
    keys: dict[str, str] = {}
    for line in path.read_text(encoding="utf-8", errors="replace").splitlines():
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        k, _, v = line.partition("=")
        keys[k.strip()] = v.strip().strip('"').strip("'")
    return keys


def load_env_context(path: Path) -> EnvContext:
    return EnvContext(path=path.resolve(), env=load_env_keys(path))


def finding_domain(env_path: Path) -> str | None:
    parts = env_path.resolve().parts
    if "by-domain" in parts:
        idx = parts.index("by-domain")
        if idx + 1 < len(parts):
            return parts[idx + 1]
    return None


def is_private_ip(host: str | None) -> bool:
    if not host:
        return False
    h = host.strip().strip("[]")
    try:
        ip = ipaddress.ip_address(h)
        return ip.is_private or ip.is_loopback or ip.is_link_local
    except ValueError:
        return False


def skip_private_db_host(raw: str | None, label: str = "DB_HOST") -> None:
    """SKIP (exit 2) when DB host is RFC1918/loopback — inacessível do scan externo."""
    if not raw:
        return
    host = raw.strip()
    if is_private_ip(host):
        print(f"SKIP: {label}={host} (IP privado/local inacessível externamente)", file=sys.stderr)
        sys.exit(2)


def is_local_host(host: str | None) -> bool:
    if not host:
        return True
    h = host.strip().lower().strip("[]")
    if h in LOCAL_HOSTS:
        return True
    if h in LOCAL_ALIASES:
        return True
    if h.endswith(".local") or h.endswith(".internal"):
        return True
    # hostname sem ponto (ex: redis) — tratar como local/docker
    if "." not in h and not h.isdigit():
        return True
    return False


def host_from_value(val: str) -> str | None:
    val = val.strip()
    if not val:
        return None
    if "://" not in val and "/" not in val.split("?")[0]:
        return val.split(":")[0].strip("[]") or None
    parsed = urlparse(val if "://" in val else f"http://{val}")
    return parsed.hostname


def extract_hosts_from_env(env: dict[str, str]) -> list[str]:
    found: list[str] = []
    seen: set[str] = set()

    def add(host: str | None) -> None:
        if not host or is_local_host(host):
            return
        key = host.lower()
        if key not in seen:
            seen.add(key)
            found.append(host)

    for name in URL_HINT_KEYS:
        if name in env:
            add(host_from_value(env[name]))

    for val in env.values():
        for match in URL_IN_VALUE_RE.finditer(val):
            add(host_from_value(match.group(0)))

    return found


def pick_remote_target(env_path: Path, env: dict[str, str]) -> tuple[str | None, str]:
    file_domain = finding_domain(env_path)
    url_hosts = extract_hosts_from_env(env)

    if file_domain and url_hosts:
        fd = file_domain.lower()
        for host in url_hosts:
            hl = host.lower()
            if hl == fd or hl.endswith("." + fd) or fd.endswith("." + hl) or fd in hl:
                return host, "URL no .env (alinhado com o finding)"
        return url_hosts[0], "URL no .env"

    if file_domain:
        return file_domain, "domínio do finding"

    if url_hosts:
        return url_hosts[0], "URL no .env"

    return None, ""


def resolve_remote_host(raw: str | None, env_path: Path, env: dict[str, str]) -> tuple[str | None, str | None]:
    if raw is None:
        return None, None
    raw = raw.strip()
    if not raw or not is_local_host(raw):
        return raw or None, None

    target, source = pick_remote_target(env_path, env)
    if not target:
        return raw, None
    return target, f"{raw} → {target} ({source})"


def resolve_service_host(
    raw: str | None,
    env_path: Path,
    env: dict[str, str],
    prefixes: tuple[str, ...] = ("db",),
) -> tuple[str | None, str | None]:
    """Resolve local/docker host: APP_URL/finding domain first, then common subdomains."""
    if raw is None:
        return None, None
    raw = raw.strip()
    if not raw or not is_local_host(raw):
        return raw or None, None

    target, source = pick_remote_target(env_path, env)
    if target:
        return target, f"{raw} → {target} ({source})"

    domain = finding_domain(env_path)
    if domain:
        for prefix in prefixes:
            if not prefix:
                continue
            candidate = f"{prefix}.{domain}"
            if candidate and not is_local_host(candidate):
                return candidate, f"{raw} → {candidate} (subdomínio)"
        return domain, f"{raw} → {domain} (domínio do finding)"

    return raw, None


def resolve_mail_host(
    raw: str | None,
    env_path: Path,
    env: dict[str, str],
    port: int = 587,
) -> tuple[str | None, str | None]:
    if raw is None:
        return None, None
    raw = raw.strip()
    if not raw or not is_local_host(raw):
        return raw or None, None

    target, source = pick_remote_target(env_path, env)
    if target:
        return target, f"{raw} → {target} ({source})"

    # Portas não-padrão (ex. 2525): não inventar smtp.{domain}
    if port not in STANDARD_SMTP_PORTS:
        domain = finding_domain(env_path)
        if domain:
            return domain, f"{raw} → {domain} (domínio; porta {port} não-padrão)"
        return raw, None

    return resolve_service_host(raw, env_path, env, ("mail", "smtp"))


def resolve_redis_host(raw: str | None, env_path: Path, env: dict[str, str]) -> tuple[str | None, str | None]:
    """Redis raramente exposto em redis.{domain}; usar só target remoto do .env/finding."""
    return resolve_service_host(raw, env_path, env, ())


def resolve_db_hosts(raw: str | None, env_path: Path, env: dict[str, str]) -> list[tuple[str, str | None]]:
    """Ordered host candidates for DB connect attempts."""
    seen: set[str] = set()
    out: list[tuple[str, str | None]] = []

    def add(host: str | None, note: str | None) -> None:
        if not host:
            return
        key = host.lower()
        if key in seen:
            return
        seen.add(key)
        out.append((host, note))

    primary, note = resolve_service_host(raw, env_path, env, ("db",))
    add(primary, note)
    domain = finding_domain(env_path)
    if domain:
        add(f"db.{domain}", f"fallback db.{domain}")
        add(domain, f"fallback {domain}")
    if not out and raw:
        add(raw.strip(), None)
    return out


def skip_smtp_dev_host(ctx: EnvContext, host: str, port: int) -> None:
    """SKIP quando host local foi mapeado ao domínio web com porta de dev (mailhog etc.)."""
    domain = finding_domain(ctx.path)
    if not domain or port in STANDARD_SMTP_PORTS:
        return
    h = (host or "").lower().rstrip(".")
    d = domain.lower().rstrip(".")
    if h == d or h.endswith("." + d):
        print(
            f"SKIP: MAIL_HOST local → {host}:{port} (porta dev; sem SMTP público)",
            file=sys.stderr,
        )
        sys.exit(2)


def skip_redis_unlikely(ctx: EnvContext) -> None:
    """SKIP Redis quando cache/sessão não usa Redis."""
    cache = (ctx.get(["CACHE_DRIVER", "CACHE_STORE"]) or "").lower()
    session = (ctx.get(["SESSION_DRIVER"]) or "").lower()
    if cache in ("file", "database", "array", "apc", "cookie", "dynamodb"):
        print(f"SKIP: CACHE_DRIVER={cache}", file=sys.stderr)
        sys.exit(2)
    if cache == "" and session in ("file", "cookie", "database"):
        print(f"SKIP: SESSION_DRIVER={session} (sem cache Redis)", file=sys.stderr)
        sys.exit(2)


def parse_port(raw: str | None, default: int = 587) -> int | None:
    if raw is None:
        return default
    val = raw.strip()
    if not val or "YOUR_" in val.upper() or not val.isdigit():
        return None
    return int(val)


def resolve_remote_url(raw: str | None, env_path: Path, env: dict[str, str]) -> tuple[str | None, str | None]:
    if not raw:
        return None, None
    raw = raw.strip()
    parsed = urlparse(raw)
    if not parsed.scheme or not parsed.hostname:
        host, note = resolve_remote_host(raw, env_path, env)
        return host, note

    if not is_local_host(parsed.hostname):
        return raw, None

    target, source = pick_remote_target(env_path, env)
    if not target:
        return raw, None

    port = parsed.port
    netloc = f"{target}:{port}" if port else target
    if parsed.username:
        auth = parsed.username
        if parsed.password:
            auth = f"{auth}:{parsed.password}"
        netloc = f"{auth}@{netloc}"

    new_url = urlunparse((
        parsed.scheme,
        netloc,
        parsed.path or "",
        parsed.params,
        parsed.query,
        parsed.fragment,
    ))
    return new_url, f"{parsed.hostname} → {target} ({source})"


def print_host_note(note: str | None) -> None:
    if note:
        print(f"  ↳ {note}", flush=True)


def env_arg_parser(description: str) -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(description=description)
    p.add_argument("--env", required=True, help="Caminho do ficheiro .env")
    p.add_argument("--key", default="", help="Variável específica (opcional)")
    p.add_argument("--batch", action="store_true", help="Modo batch (sem prompts, email de teste)")
    return p


def pick_key(env: dict[str, str], candidates: list[str], override: str = "") -> str | None:
    if override:
        return env.get(override)
    for c in candidates:
        if env.get(c):
            return env[c]
    return None


def main_missing(key_name: str) -> None:
    print(f"SKIP: {key_name} não encontrada", file=sys.stderr)
    sys.exit(2)


def is_interactive() -> bool:
    return sys.stdin.isatty() and sys.stdout.isatty()


def select_from_list(prompt: str, items: list[str]) -> str | None:
    if not items:
        print("Nenhum item disponível.")
        return None
    print(f"\n{prompt}")
    for i, item in enumerate(items, 1):
        print(f"  {i}. {item}")
    while True:
        raw = input("\nEscolha (número): ").strip()
        if not raw:
            continue
        try:
            idx = int(raw)
            if 1 <= idx <= len(items):
                return items[idx - 1]
        except ValueError:
            pass
        print("Opção inválida.")


def chat_loop(ask_fn, name: str = "assistente") -> None:
    print(f"\nConversa com {name}. Digite 'sair' para terminar.\n", flush=True)
    while True:
        try:
            prompt = input("você> ").strip()
        except (EOFError, KeyboardInterrupt):
            print("\nAté logo.", flush=True)
            break
        if not prompt:
            continue
        if prompt.lower() in ("sair", "exit", "quit"):
            print("Até logo.", flush=True)
            break
        try:
            log_step(f"A consultar {name}…")
            reply = ask_fn(prompt)
            print(f"\n{name}> {reply}\n", flush=True)
        except Exception as exc:
            print(f"Erro: {exc}\n", flush=True)


def prompt_required(label: str) -> str:
    while True:
        try:
            val = input(f"{label}: ").strip()
        except (EOFError, KeyboardInterrupt):
            print()
            raise SystemExit(130)
        if val:
            return val
        print("Campo obrigatório.")


def prompt_optional(label: str, default: str = "") -> str:
    hint = f" [{default}]" if default else ""
    try:
        val = input(f"{label}{hint}: ").strip()
    except (EOFError, KeyboardInterrupt):
        print()
        raise SystemExit(130)
    return val or default


def prompt_multiline(label: str) -> str:
    print(f"{label} (linha vazia para terminar):")
    lines: list[str] = []
    while True:
        try:
            line = input()
        except (EOFError, KeyboardInterrupt):
            print()
            if lines:
                break
            raise SystemExit(130)
        if line == "":
            break
        lines.append(line)
    return "\n".join(lines)

