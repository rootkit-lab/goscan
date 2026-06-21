#!/usr/bin/env python3
"""Valida chave Mistral AI."""

from llmutil import OpenAIProvider, run_openai_provider

if __name__ == "__main__":
    run_openai_provider(
        OpenAIProvider(
            label="Mistral",
            env_keys=("MISTRAL_API_KEY",),
            base_url="https://api.mistral.ai/v1",
        )
    )
