# AGENT.md

## Purpose

This repository contains `sub2api`, an AI API gateway platform with a Go backend and a Vue 3 frontend. When working in this project, prioritize correctness, security, and compatibility with the existing architecture over broad refactors.

## Project Context

- Backend stack: Go, Gin, Ent, PostgreSQL, Redis.
- Frontend stack: Vue 3, Vite, TailwindCSS.
- The project includes deployment scripts, migrations, and admin dashboard code.
- Preserve existing module boundaries unless a task explicitly asks for architectural changes.

## General Working Rules

- Read the relevant files before making edits.
- Keep changes focused; avoid unrelated cleanup.
- Do not overwrite or revert user changes you did not make.
- Follow the existing code style and naming conventions in each area of the repo.
- Prefer small, reviewable diffs over sweeping rewrites.
- When behavior changes, update nearby docs or config examples if needed.

## Backend Guidelines

- Keep handlers thin; put business logic in services or existing domain layers.
- Reuse existing patterns for routing, middleware, logging, and persistence.
- Validate inputs explicitly and return consistent error responses.
- Be careful with concurrency, rate limiting, quota accounting, and billing logic.
- Avoid introducing hidden side effects in request handling or background jobs.
- Preserve backward compatibility for public APIs unless the task explicitly allows breaking changes.

## Frontend Guidelines

- Reuse existing UI patterns, stores, and composables before adding new abstractions.
- Keep components focused and split overly large view logic into composables or smaller components.
- Maintain responsive behavior and avoid breaking dashboard workflows.
- Prefer clear loading, empty, and error states for async views.
- Follow existing Tailwind and Vue conventions instead of introducing a new styling approach.

## Data and Security

- Never hardcode secrets, tokens, or credentials.
- Treat authentication, authorization, billing, and account-selection code as high risk.
- Be cautious when editing database schemas, migrations, and quota-related calculations.
- Preserve auditability: logging and usage tracking changes should remain understandable and traceable.

## Testing and Validation

- After substantive edits, run the most relevant targeted checks available for the changed area.
- Prefer narrow validation first, then broader checks if the change touches shared infrastructure.
- If tests cannot be run, state that clearly and explain what should be verified manually.
- Fix introduced lint or type issues in files you changed.

## Change Style

- Favor minimal patches that solve the requested problem.
- Add comments only when the intent is non-obvious.
- Avoid speculative refactors, dependency churn, or file moves unless they directly support the task.
- Keep commit-ready quality: no placeholder code, no dead branches, no unexplained TODOs.

## Communication

- Explain what changed, where, and why in concise terms.
- Highlight risks, assumptions, and follow-up work when relevant.
- When unsure about scope or behavior, ask before making broad or irreversible changes.
