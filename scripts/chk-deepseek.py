#!/usr/bin/env python3
"""Valida chave DeepSeek."""

from llmutil import OpenAIProvider, run_openai_provider

if __name__ == "__main__":
    run_openai_provider(
        OpenAIProvider(
            label="DeepSeek",
            env_keys=("DEEPSEEK_API_KEY",),
            base_url="https://api.deepseek.com/v1",
        )
    )
