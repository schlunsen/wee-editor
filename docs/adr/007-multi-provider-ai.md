# ADR 007: Multi-Provider AI Architecture

## Status

Accepted

## Date

2024-01-01

## Context

While Wee is primarily built around Anthropic's Claude, users have expressed interest in using alternative AI providers for different use cases (cost optimization, specific model capabilities, privacy requirements). The system needs to support multiple AI backends without tightly coupling the agent orchestration logic to a single provider's API.

Providers to support: Anthropic Claude (primary), DeepSeek, GLM, Kimi, and custom Anthropic-compatible endpoints.

## Decision

We implemented a **provider registry pattern** in `internal/providers/` that abstracts AI provider configuration and selection.

**Architecture:**

- **Provider Registry** manages available providers and their configurations
- **Provider Configuration** stored in SQLite (`providers` table) with JSON settings
- **claude-agent-sdk-go** (`github.com/schlunsen/claude-agent-sdk-go`) serves as the primary SDK for agent orchestration
- **OpenRouter integration** enables access to third-party models (DeepSeek, GLM, Kimi) through a unified API
- **Default model** is `claude-sonnet-4-5-20250929` but configurable per session

**Configuration per provider:**

- API endpoint URL
- API key (stored securely)
- Available models
- Rate limits and quotas
- Custom parameters

**Session-level selection:**

- Each agent session can specify its provider and model
- Default provider/model configured globally
- Provider switching without session restart

## Consequences

### Positive

- Users can choose the best model for each task (cost vs capability)
- OpenRouter integration provides access to dozens of models through one API
- Provider configuration is stored in the database, not hardcoded
- New providers can be added without changing core agent orchestration code
- Custom endpoints support enterprise API proxies and self-hosted models
- Per-session provider selection enables A/B testing of models

### Negative

- The claude-agent-sdk-go is tightly coupled to Anthropic's message format; non-Anthropic providers must be compatible
- Different providers have different tool-use capabilities, creating feature parity challenges
- API key management complexity increases with each provider
- Error handling varies across providers (rate limits, token limits, error formats)
- Testing across all providers is difficult to automate
- OpenRouter adds a dependency and potential point of failure for third-party models
