# Shared Libraries (`libs/`)

This document describes the purpose and contents of the `libs/` directory.

The `libs/` directory contains **shared, reusable packages** that are used across multiple services in the system.
These packages are **not business-domain specific** and are designed to be stable, shared, and predictable.

---

## Design Principles

The shared libraries follow a few explicit rules:

- No business logic
- No service-specific behavior
- No direct database access
- No cross-service orchestration

Their purpose is to **reduce duplication**, not to introduce new abstractions.

---

## Why `libs/` Exists

In a multi-service Go codebase, some functionality is inevitably repeated:
- HTTP middleware
- authentication helpers
- configuration loading
- logging
- error handling

The `libs/` directory centralizes this functionality to:
- keep services small and focused
- enforce consistent behavior across services
- avoid copy-paste reuse

This approach mirrors common Go monorepo patterns used in production systems.

---

## Package Overview

Below is a high-level overview of the types of packages contained in `libs/`.

---

### Configuration

Responsible for:
- loading environment variables
- validating required configuration
- providing typed configuration structs

Characteristics:
- environment-driven
- fails fast on missing or invalid configuration
- no hardcoded defaults for secrets

Used by:
- all services at startup

---

### Logging

Provides:
- structured logging
- consistent log formatting
- log levels controlled via configuration

Characteristics:
- logs written to stdout
- suitable for containerized environments
- no business context embedded

Used by:
- all services
- background workers and schedulers

---

### HTTP Middlewares

Provides reusable HTTP middleware such as:
- request logging
- panic recovery
- CORS handling
- request size limits
- request ID injection

Characteristics:
- framework-agnostic where possible
- composable
- no service-specific logic

Used by:
- all HTTP-facing services

---

### Authentication Utilities

Provides:
- JWT creation and validation helpers
- token parsing utilities
- shared authorization helpers

Characteristics:
- cryptographic concerns isolated from handlers
- no persistence logic
- no user-specific business rules

Used by:
- services requiring authentication or authorization

---

### HTTP Utilities (/handlers)

Provides shared helpers for:
- request parsing
- response formatting
- error responses

Characteristics:
- consistent API responses across services
- centralized error shape
- avoids duplication in handlers

---

## What Does *Not* Belong in `libs/`

To keep shared libraries healthy, the following are intentionally excluded:

- database repositories
- domain models
- service-specific clients
- background job definitions
- orchestration logic

Those concerns belong inside individual services.

---

## Notes for Reviewers

- `libs/` is intentionally small and conservative
- Packages evolve slowly and are treated as stable APIs
- The goal is clarity and consistency, not abstraction depth