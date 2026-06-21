#!/usr/bin/env python3
"""Valida token Hugging Face."""

from llmutil import run_simple_bearer_check

if __name__ == "__main__":
    run_simple_bearer_check(
        "Hugging Face",
        ["HF_TOKEN", "HUGGINGFACE_API_KEY", "HUGGING_FACE_HUB_TOKEN"],
        "https://huggingface.co/api/whoami-v2",
        ok_summary=lambda r: f"User: {r.json().get('name') or r.json().get('fullname', '?')}",
    )
