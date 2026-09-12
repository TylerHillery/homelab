# 0001: Use golink as the HLIMS Foundation

Status: accepted

## Decision

HLIMS will hard-fork Tailscale golink inside the homelab monorepo. It will keep
the Go standard library HTTP stack and introduce OpenAPI generation, goose
migrations, sqlc queries, a proper JSON API, and catalog resources.

## Rationale

golink already provides the desired Tailscale identity, direct redirects,
SQLite persistence, embedded server-rendered UI, exports, and operational
model. The upstream package is not composable: its handler, identity logic, and
application state are private and coupled to `Run`. Importing it would require
duplicating core behavior.

The fork accepts responsibility for upstream updates in exchange for a clean
API-first product boundary.

## Consequences

- The original BSD-3-Clause license and attribution remain in the service.
- Existing golink behavior is protected by tests while internals evolve.
- HLIMS owns its API and database migration compatibility.
- Upstream changes are selectively reviewed and integrated.
