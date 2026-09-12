# HLIMS Architecture

HLIMS begins as a hard fork of Tailscale golink and retains its proven tsnet,
redirect, server-rendered UI, and SQLite behavior. New development treats the
HTTP API as the supported application boundary.

## Principles

- The CLI and external integrations are clients of the API.
- The server-rendered UI and API share the same application services.
- HTMX enhances the hypermedia interface; HTML forms remain functional without
  JavaScript.
- OpenAPI is the external contract; generated types and clients prevent drift.
- UI routes encode navigation and visible state.
- Application rules do not live in HTTP handlers or SQL queries.
- Only repository implementations access SQLite.
- Service instance traffic remains direct and never proxies through HLIMS.
- New abstractions require a concrete second use; small duplication is allowed.

## Boundaries

```text
CLI -------------+--> generated API client --> HTTP API ---+
Future clients --+                                         |
                                                           v
Browser --> HTML/HTMX handlers -----------> application services
                                                           |
                                                 repository interfaces
                                                           |
                                                    sqlc queries
                                                           |
                                                        SQLite
```

The existing golink templates are the starting point for the HLIMS UI and evolve
incrementally. Characterization tests protect their behavior while application
logic is extracted from HTTP handlers.

## Repository ownership

```text
api/                   OpenAPI source and generation configuration
db/migrations/         Goose migration source
db/queries/            Handwritten sqlc query source
generated/api/         Generated API server types and client
generated/db/          Generated SQLite query code
internal/              HLIMS-owned application implementation
tmpl/ and static/      Server-rendered HLIMS UI and static assets
```

Generated files are committed and checked by prek. They are never edited by
hand.

## HTTP representations

The JSON API under `/.api/v1` and the browser-facing hypermedia routes are
separate HTTP adapters. JSON handlers implement the OpenAPI contract. HTML
handlers render complete pages for ordinary requests and template fragments for
HTMX requests. Both adapters call the same application services; neither owns
business rules or persistence logic.

## Implementation sequence

1. Preserve the imported upstream tests and behavior.
2. Establish migrations and generated database queries without changing data.
3. Establish the versioned Links API and generated client.
4. Move link validation and authorization into an application service shared by
   the API, redirect handler, and server-rendered UI.
5. Add Device, Machine, Service, and Instance resources based on real examples.
6. Build the CLI with the generated client.
7. Evolve the server-rendered templates into the HLIMS management UI.
8. Package HLIMS for deployment, then add environment code under `infra/`.

## Upstream policy

The fork point and update policy are recorded in
[`services/hlims/docs/upstream.md`](../../services/hlims/docs/upstream.md).
Upstream changes are reviewed rather than merged automatically.
