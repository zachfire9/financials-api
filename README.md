# Financials API

This repository is being reset from the original prototype into a new API for a local-first personal financial planning application.

The previous Spring Boot prototype has been removed on the `docs/reset-api-plan` branch so the rewrite can start from a clean baseline.

## Current status

- Existing application code: removed
- Current branch purpose: public-safe planning baseline
- Detailed implementation: not started yet
- Runtime/deployment specifics: intentionally omitted from git until they can be represented with placeholders and local-only config files

## Planning documents

- [Local Hosting Plan](docs/local-hosting-plan.md)

## Public repo boundaries

This repo is public, so it must not contain:

- Real financial data
- Secrets, API keys, passwords, or tokens
- LAN IPs, hostnames, router/firewall details, or machine names
- Local database paths, backup paths, or user-specific service names
- Environment-specific production configuration

Use committed examples with placeholders only, and keep real runtime values in ignored local files such as `.env` when implementation begins.
