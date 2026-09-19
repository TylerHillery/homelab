# HLIMS Architecture

HLIMS is an inventory-backed HTTP redirect resolver. It catalogs hardware,
Machines, Networks, Addresses, Services, Instances, and reachable endpoints. It
does not provision resources or proxy service traffic.

## Principles

- OpenAPI is the external contract and generates both server bindings and a Go
  client suitable as the basis for additional SDKs.
- Browser redirects and API resolution use the same resolver.
- Redirect destinations are derived from inventory rather than copied into a
  separate short-link store.
- Only sqlc-generated queries access application data in normal operation.
- Service traffic remains direct from the client to the selected endpoint.
- Network exposure is a deployment concern, not an application dependency.
- Tailscale is a first-class Network and endpoint option without being required
  to run HLIMS.

## Request Flow

```text
Browser -> GET /badger/opencode/production -----+
                                                  v
SDK ----> GET /api/v1/resolve/... ----------> resolver
                                                  |
                                    Machine, Service, Instance
                                                  |
                                      selected InstanceEndpoint
                                                  |
                                         HTTP destination URL
```

The browser route returns `302 Found`. The API route returns the same destination
as JSON so clients can inspect it without following the redirect.

Ad hoc development ports use `/badger/5173`. The optional `via=lan|tailnet`
query parameter selects the Network kind; `scheme=http|https` applies only to ad
hoc routes. Resolver query parameters are removed and all remaining query values
are forwarded to the destination.

## Boundaries

```text
OpenAPI source -> generated server/client -> JSON handlers --+
                                                          |
Browser redirect handler ---------------------------------+-> resolver
                                                          |
                                                     sqlc queries
                                                          |
                                                        SQLite
```

Repository ownership:

```text
api/                   OpenAPI source and generator configuration
db/migrations/         Goose schema migrations
db/queries/            sqlc query source
generated/api/         Generated API models, server, and client
generated/db/          Generated SQLite models and queries
```

Generated files are committed and verified by `mise run check`. They are never
edited manually.

## Client Applications

The project builds two independent executables:

```text
hlimsd    long-running HTTP API and redirect server
hlims     Cobra CLI and Bubble Tea terminal console
```

The `hlims` client communicates exclusively through the HTTP API using the
generated OpenAPI client. It does not import server implementation packages,
sqlc queries, migrations, or SQLite. Normal commands provide scriptable CRUD
and resolver access; `hlims console` launches the interactive TUI. Both modes
share a small client-side facade, so terminal presentation remains separate
from generated transport types.

CLI commands are non-interactive and JSON-first so shell scripts and coding
agents can safely inspect and mutate inventory. Writes accept JSON from stdin or
a file, destructive operations require an explicit confirmation flag, and API
failures produce nonzero exits. `hlims openapi` exposes the live contract for
schema discovery. `hlims open` parses the same canonical and ad hoc paths as the
browser resolver, resolves through the API, and launches the destination in the
client machine's browser; `--print` keeps that workflow non-interactive.

This boundary is language-agnostic: another client can be implemented from the
same OpenAPI document without changing or embedding `hlimsd`.

## Deployment

HLIMS serves ordinary HTTP on a configurable address. Production should bind to
loopback or a private container network and place the server behind a trusted
private ingress.

The primary deployment runs on a VPS whose Tailscale node is named `go`:

```text
HLIMS:           127.0.0.1:8080
Tailscale Serve: http://go -> 127.0.0.1:8080
Public ingress:  none
```

MagicDNS makes `http://go/badger/opencode/production` available to devices
permitted by Tailnet ACLs. Funnel must remain disabled. Other deployments may
use WireGuard, a private reverse proxy, or another secure ingress without
changing HLIMS.

The terminology and responsibility boundaries for service managers, container
runtimes, orchestrators, deployment controllers, ingress publishers, and HLIMS
are defined in the [Deployment Model](deployment-model.md).

## API

The versioned API is mounted at `/api/v1`. The source specification is also
served at `/openapi.yaml`. Server URLs in the specification are relative so a
generated client can target `http://go`, a full Tailscale HTTPS hostname, or any
other deployment base URL.

The API provides collection `GET`/`POST` and public-ID item `GET`/`PUT`/`DELETE`
for Machine Providers, Areas, Manufacturers, Products, Assets, Machines, Machine
Users, Networks, Addresses, Services, Instances, and Instance
Endpoints. Related resources must already exist and are referenced by public ID;
writes never create dependencies implicitly. Product and kind-specific
specification writes are atomic. Asset placement is explicitly an Area, a
containing Asset, or an unplaced state.

Machine writes accept an optional `isFavorite` flag that defaults to false.
Machine reads and topology snapshots always include the resulting boolean. The
Web Console can update this state directly from each Machine card and rerenders
the topology fragment so duplicate favorite and provider views stay consistent.

`GET /api/v1/topology` provides a consistent read snapshot grouped by Machine
Provider, Area, recursive Machine hierarchy, Service, and Instance. Both the Web
Console and terminal console use this topology instead of independently joining
flat resource collections. Each topology Instance includes the
resolver-selectable LAN and Tailnet routes currently backed by an Instance
Endpoint. The Web Console is served from `/console/` using embedded Go
templates, custom CSS, and vendored htmx 4. Providers use native collapsible
sections; subtree expansion and refreshes request HTML fragments and do not
require a separate frontend build or full-page navigation. Provider and Service
logos are stored as separate SQLite BLOBs and served through cacheable raw image
endpoints rather than embedded in JSON.

Topology Machines also carry their Machine Users and network Addresses. The Web
Console uses this inventory to build generic `ssh user@host` copy actions, with
the preferred user and hostname first and network DNS/IP alternatives in a
compact menu. Clipboard behavior is delegated so htmx fragments work without
inline scripts, and a legacy copy fallback supports trusted plain-HTTP Console
deployments. HLIMS never stores SSH credentials or modifies client host trust.

Reachability is measured in the Web Console, not by `hlimsd`. The current browser
periodically follows each Instance resolver path with a timed, opaque HTTP
request and updates Instance and LAN/Tailnet route lights in place. Machines do
not expose a status because browser JavaScript cannot use ICMP or open arbitrary
TCP connections to establish Machine reachability. This answers whether the
device viewing HLIMS can reach a concrete HTTP route instead of whether the
HLIMS server can reach it, and no observations are persisted or exposed through
the API.

This signal is explicitly best effort. Browser JavaScript cannot open arbitrary
TCP connections, opaque responses do not expose HTTP status, and failed requests
may reflect TLS, mixed-content, or private-network browser policy rather than a
network outage. Operational service health and historical monitoring remain the
responsibility of systems such as Prometheus. The delegated controller also
rescans htmx fragments without replacing topology cards or disturbing native
details state, route menus, Favorites, or scroll position.

HLIMS intentionally has no application authentication in the current personal
deployment model. Any client that can reach the private ingress can read or
mutate inventory. The server emits no CORS permission headers, so CLIs and
server-side SDKs work normally while arbitrary browser origins are not granted
cross-origin access. Browser clients should use the same origin; explicit origin
allowlisting can be added when a concrete separate browser application exists.

## Upstream History

HLIMS began from Tailscale golink, but does not retain its Link, ownership,
template, search, export, statistics, tsnet, or Tailscale authentication model.
The fork point is recorded in
[`services/hlims/docs/upstream.md`](../../services/hlims/docs/upstream.md).
