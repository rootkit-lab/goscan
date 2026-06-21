#!/usr/bin/env python3
"""Valida chave Together AI."""

from llmutil import OpenAIProvider, run_openai_provider

if __name__ == "__main__":
    run_openai_provider(
        OpenAIProvider(
            label="Together",
            env_keys=("TOGETHER_API_KEY", "TOGETHERAI_API_KEY", "TOGETHER_AI_API_KEY"),
            base_url="https://api.together.xyz/v1",
        )
    )
