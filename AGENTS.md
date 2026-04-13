# AGENTS.md

## Repository Rules

- When resolving conflicts against upstream, never merge upstream payment-related code into this project.
- Only merge upstream changes that are directly related to the gateway service itself.
- Treat payment code as locally maintained customization unless an explicit task says otherwise.
- During upstream merges or cherry-picks, local payment code always wins over upstream payment code.
- If an upstream commit mixes gateway fixes with payment changes, merge only the gateway-service portion and keep the local payment implementation unchanged.

## Scope Reminder

- Keep merge and sync work limited to the gateway service itself unless the task explicitly expands scope.
