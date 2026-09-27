# HLIMS

HLIMS is the Home Lab Information Management System. It catalogs infrastructure
and resolves inventory-backed paths to client-reachable service URLs without
proxying application traffic.

```text
http://go/badger/opencode/production
http://go/badger/5173?via=tailnet
```

The `go` host is the common entry point. Canonical paths use
`go/<machine>/<service>/<instance>`; ad hoc paths use `go/<machine>/<port>`.

See the [architecture](../../docs/architecture/hlims.md) and
[domain model](../../docs/architecture/domain-model.md) for system boundaries
and inventory relationships.

## Development

HLIMS owns its tools and tasks in `mise.toml`.

```sh
mise install
mise run check
mise run dev
```

`mise run dev` serves a disposable demonstration inventory with live reload at
<http://localhost:8090>. `mise run serve` starts an empty in-memory database at
<http://localhost:8080>. Normal startup does not seed data.

## Build And Run

```sh
go build ./cmd/hlimsd ./cmd/hlims
hlimsd --listen=127.0.0.1:8080 --sqlitedb=/var/lib/hlims/hlims.db
```

`hlimsd` requires a SQLite database path for persistent use. Use `--demo` only
with an empty disposable database.

## Interfaces

The server provides:

| Path | Purpose |
|---|---|
| `/api/v1` | Versioned JSON API |
| `/console/` | Web inventory console |
| `/openapi.yaml` | OpenAPI source |
| `/api-docs/` | Self-hosted API reference |

The OpenAPI source is [`api/openapi.yaml`](api/openapi.yaml). Generated server
and client code is committed under `generated/api` and verified by
`mise run check`.

`hlims` is an API-only client. It does not open the SQLite database.

```sh
hlims products list
hlims machines list
hlims resolve instance badger opencode production
hlims open badger/opencode/production
hlims console
```

The client targets `http://127.0.0.1:8080/api/v1` by default. Set
`HLIMS_API_URL` or pass `--api-url` for another deployment. Data commands emit
JSON. Create and update commands read JSON from standard input by default.
`hlims open` accepts paths with or without the leading `go/` and full `go` URLs.
Use `--print` to resolve a path without launching a browser.

## Operations

HLIMS has no application authentication. Deploy it only behind trusted private
ingress; any client that can reach the API can mutate inventory. The server does
not grant cross-origin browser access.

Web Console reachability indicators are browser-local observations, not service
health monitoring. Browser security policy can report an endpoint as
unreachable even when the service is healthy.

HLIMS has no runtime dependency on Tailscale. A private deployment can expose a
loopback listener from a node named `go` with Tailscale Serve:

```sh
hlimsd --listen=127.0.0.1:8080 --sqlitedb=/var/lib/hlims/hlims.db
tailscale serve --bg --http=80 http://127.0.0.1:8080
```

Do not enable Funnel or expose the listener publicly. Another authenticated
private ingress can provide the same boundary.

## Sources Of Truth

- [`api/openapi.yaml`](api/openapi.yaml): API contract
- [`db/migrations`](db/migrations): database schema
- [`../../docs/architecture/hlims.md`](../../docs/architecture/hlims.md): system
  boundaries
- [`../../docs/architecture/domain-model.md`](../../docs/architecture/domain-model.md):
  inventory concepts
