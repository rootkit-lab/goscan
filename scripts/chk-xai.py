#!/usr/bin/env python3
"""Valida chave xAI (Grok)."""

from llmutil import OpenAIProvider, run_openai_provider

if __name__ == "__main__":
    run_openai_provider(
        OpenAIProvider(
            label="xAI",
            env_keys=("XAI_API_KEY", "GROK_API_KEY", "XAI_KEY"),
            base_url="https://api.x.ai/v1",
        )
    )
