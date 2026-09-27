# 0001: Use golink as the HLIMS Foundation

Status: accepted for the original fork; current behavior is documented in
[`docs/architecture/hlims.md`](../architecture/hlims.md).

## Decision

HLIMS will hard-fork Tailscale golink inside the homelab monorepo. It will keep
the Go standard library HTTP stack and introduce OpenAPI generation, goose
migrations, sqlc queries, a proper JSON API, and catalog resources.

## Rationale

golink provided a useful Go HTTP, redirect, SQLite, and embedded-UI foundation.
The upstream package was not composable: its handler, identity logic, and
application state were private and coupled to `Run`. Importing it would have
required duplicating core behavior.

The fork accepts responsibility for upstream updates in exchange for a clean
API-first product boundary.

## Consequences

- The original BSD-3-Clause license and attribution remain in the service.
- Golink behavior may be removed as HLIMS develops independent boundaries.
- HLIMS owns its API and database migration compatibility.
- Upstream changes are selectively reviewed and integrated.
