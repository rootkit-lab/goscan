#!/usr/bin/env python3
"""Valida token Replicate."""

from llmutil import run_simple_bearer_check

if __name__ == "__main__":
    run_simple_bearer_check(
        "Replicate",
        ["REPLICATE_API_TOKEN", "REPLICATE_API_KEY"],
        "https://api.replicate.com/v1/account",
        ok_summary=lambda r: f"Conta: {r.json().get('username', '?')}",
    )
