# AGENTS.md

## Repository Rules

- When resolving conflicts against upstream, never merge upstream payment-related code into this project.
- Only merge upstream changes that are directly related to the gateway service itself.
- Treat payment code as locally maintained customization unless an explicit task says otherwise.
- During upstream merges or cherry-picks, local payment code always wins over upstream payment code.
- If an upstream commit mixes gateway fixes with payment changes, merge only the gateway-service portion and keep the local payment implementation unchanged.
- For upstream sync work, do not directly merge `upstream/main` by default. First compare the diverged commits, then cherry-pick only the specific upstream commits that are safe and relevant to the gateway service.
- Prefer small, self-contained, gateway-related fixes when cherry-picking. Skip commits that are payment-focused, broad refactors, or otherwise high-risk unless the task explicitly requires them.
- If a commit mixes gateway changes with payment changes, do not cherry-pick it wholesale. Either pick only the gateway-related hunks or skip the commit.
- If an upstream fix is already present locally or cherry-picks as an empty change, skip it instead of forcing a duplicate commit.

## Scope Reminder

- Keep merge and sync work limited to the gateway service itself unless the task explicitly expands scope.
