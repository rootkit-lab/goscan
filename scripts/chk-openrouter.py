#!/usr/bin/env python3
"""Valida chave OpenRouter."""

from llmutil import OpenAIProvider, run_openai_provider

if __name__ == "__main__":
    run_openai_provider(
        OpenAIProvider(
            label="OpenRouter",
            env_keys=("OPENROUTER_API_KEY",),
            base_url="https://openrouter.ai/api/v1",
        )
    )
