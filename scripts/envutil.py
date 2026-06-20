#!/usr/bin/env python3
"""Utilitários partilhados para checkers."""

from __future__ import annotations

import argparse
import re
import sys
from dataclasses import dataclass
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
        print(f"  ↳ {note}")


def env_arg_parser(description: str) -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(description=description)
    p.add_argument("--env", required=True, help="Caminho do ficheiro .env")
    p.add_argument("--key", default="", help="Variável específica (opcional)")
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
    print(f"\nConversa com {name}. Digite 'sair' para terminar.\n")
    while True:
        try:
            prompt = input("você> ").strip()
        except (EOFError, KeyboardInterrupt):
            print("\nAté logo.")
            break
        if not prompt:
            continue
        if prompt.lower() in ("sair", "exit", "quit"):
            print("Até logo.")
            break
        try:
            reply = ask_fn(prompt)
            print(f"\n{name}> {reply}\n")
        except Exception as exc:
            print(f"Erro: {exc}\n")


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

