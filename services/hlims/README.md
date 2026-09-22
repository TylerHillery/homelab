# HLIMS

HLIMS is the Home Lab Information Management System. It resolves memorable,
inventory-backed paths to services and ad hoc ports without proxying the target
traffic.

```text
/badger/opencode/production
/badger/5173?via=tailnet
```

## Development

HLIMS owns its tool versions and tasks in this directory's `mise.toml`.

```bash
mise install
mise run check
mise run dev
```

Open <http://localhost:8090> while `mise run dev` is running. Use
`mise run serve` when live reload is not needed. The topology-first Web Console
is available at <http://localhost:8090/console/>. It is server-rendered and uses
collapsible Provider sections plus vendored htmx 4 fragments for machine subtree
expansion and refreshes without full-page transitions. The console follows the
system light/dark preference, uses Atom One Light and Atom One Dark colors, and
includes a persistent manual theme toggle. Nested virtual Machines use a
responsive grid and identify their immediate host. Root Machines use the same
responsive grid within each Area. Instance rows open their preferred route
directly and expose a compact menu for available LAN and Tailnet alternatives.
The star on every Machine card updates its favorite state and the quick-access
Favorites group immediately. Machines with a recorded user and host expose a
one-click generic SSH command plus hostname, LAN, and Tailnet alternatives; no
credentials are stored. Small status lights show whether the current browser can
reach each HTTP/HTTPS Instance and LAN/Tailnet route. Machines do not have status
lights because browsers cannot perform ICMP or arbitrary TCP probes.
Browser probes run without replacing topology, so open menus, expanded sections,
and scroll position remain intact. Hover a light for the result, duration, check
time, and relevant browser limitations.

The `dev` task starts with a fresh in-memory demonstration inventory containing
bare-metal Machines that run services directly and contain virtual Machines with
their own services, resolver endpoints, and generated provider and service logos.
All demo addresses and DNS names are illustrative placeholders, not devices on
the operator's LAN or Tailnet, so their reachability probes are expected to
fail. Badger runs two Grafana Instances to exercise same-Service, same-Machine
grouping. The demo also gives Badger, Brewer, and Buck a preferred `tyler`
Machine User. Badger has additional `deploy` and `root` users to demonstrate
user selection. The demo uses one `Home` Area under the Homelab provider; Areas
represent physical sites or provider regions, not machine hierarchy. Normal
startup does not seed data. Pass `--demo` to `hlimsd` explicitly to seed any
other empty disposable database.

The OpenAPI source is `api/openapi.yaml`. The generated Go server and client are
committed under `generated/api`, and the running server publishes the source at
`/openapi.yaml` for additional SDK generation. The Console's `API / V1` link
opens a self-hosted, read-only Swagger UI at `/api-docs/` in a new tab; it does
not depend on a public service being able to reach the private HLIMS deployment.
Operations are grouped by resource using OpenAPI tags.
Machine create and full-replacement update payloads may set `isFavorite`; reads
and topology responses always expose the persisted value.

The API is mounted at `/api/v1`. Inventory relationships are created in parent
first order and use public IDs; writes never create missing dependencies. The
application has no authentication layer, so deploy it only behind a trusted
private ingress. It does not grant cross-origin browser access by default.

Reachability is intentionally a Console-only, per-tab signal rather than an API
or persisted health record. Every 30 seconds the browser follows each resolver
route with a five-second, opaque HTTP probe. A successful HTTP response of any
status means the route was reachable from that browser. A failure can mean the
network is unavailable, but browsers can also reject probes because of TLS,
mixed-content, or private-network policy. Browsers cannot perform arbitrary TCP
checks, so these lights must not be treated as service health monitoring.

## Client

`hlimsd` is the long-running API server. `hlims` is a separate API-only client
built with Cobra; it never imports the server or opens the SQLite database.

```bash
go run ./cmd/hlims --help
go run ./cmd/hlims products list
go run ./cmd/hlims assets list
go run ./cmd/hlims purchases list
go run ./cmd/hlims purchase-summary
go run ./cmd/hlims machines list
go run ./cmd/hlims machine-users list
go run ./cmd/hlims services create --file service.json
go run ./cmd/hlims resolve instance badger grafana production
go run ./cmd/hlims open badger/grafana/production
go run ./cmd/hlims console
```

The client uses `http://127.0.0.1:8080/api/v1` by default. Set
`HLIMS_API_URL` or pass `--api-url` to target another deployment. The `console`
command opens the Bubble Tea inventory browser. Use up/down or `j`/`k` to move,
Enter/right to expand the provider, area, and Machine hierarchy, left to collapse
or select a parent, `r` to refresh, and `q` to quit. Machine details include
capacity, immediate host context for nested virtual Machines, and attached
service instances arranged as compact cards.

CLI data commands emit JSON and return nonzero on errors. Create and update read
JSON from stdin by default, making discovery and mutation easy to compose:

```bash
hlims openapi > /tmp/hlims-openapi.yaml
hlims products list --compact | jq -r '.items[] | [.kind, .name] | @tsv'
hlims machines list --compact | jq -r '.items[].publicId'
printf '%s\n' '{"machinePublicId":"badmach8k2q5","username":"tyler","isPreferred":true}' | hlims machine-users create
printf '%s\n' '{"name":"Grafana"}' | hlims services create
hlims open --print 'badger/grafana/production/d/overview?refresh=30s'
```

Purchases are standalone CRUD resources. Each records `totalPriceCents` and
`currency`, optional `purchasedOn`, `source`, and `notes`, typed `receipt`,
`listing`, or `other` URL links, and one or more `assetPublicIds`. An Asset can
belong to at most one Purchase and exposes the relationship as
`purchasePublicId`. A bundle, including a mini-PC bundle, is one Purchase
associated with multiple Assets.

`GET /api/v1/purchase-summary` and `hlims purchase-summary` group totals by
currency, add each Purchase exactly once, and report both Purchase and
associated Asset counts. An explicit zero total records a free transaction; it
does not stand for an unknown price.

Machine Providers and Services can have a PNG, JPEG, or WebP logo up to 1 MiB
stored directly in SQLite. Their responses expose `hasLogo`; logo bytes use
separate endpoints so collection and topology responses remain small:

```bash
curl --request PUT --header 'Content-Type: image/png' \
  --data-binary @grafana.png \
  http://127.0.0.1:8080/api/v1/services/SERVICE_PUBLIC_ID/logo
curl http://127.0.0.1:8080/api/v1/services/SERVICE_PUBLIC_ID/logo --output grafana.png
curl --request DELETE http://127.0.0.1:8080/api/v1/services/SERVICE_PUBLIC_ID/logo

curl --request PUT --header 'Content-Type: image/png' \
  --data-binary @homelab.png \
  http://127.0.0.1:8080/api/v1/machine-providers/PROVIDER_PUBLIC_ID/logo
curl http://127.0.0.1:8080/api/v1/machine-providers/PROVIDER_PUBLIC_ID/logo --output homelab.png
curl --request DELETE http://127.0.0.1:8080/api/v1/machine-providers/PROVIDER_PUBLIC_ID/logo
```

`hlims open` accepts canonical instance paths, ad hoc Machine-port paths,
optional `go/` prefixes, and full go URLs. It resolves through the API and opens
the final URL in the local browser. Extra path segments, query values, and URL
fragments are forwarded. Use `--print` to resolve the same path without browser
interaction. On WSL it prefers `wslview`, then the mounted Windows browser
launcher; `BROWSER` overrides platform detection.
Set `BROWSER` to the launcher executable path; wrapper arguments should live in
a separate script so paths containing spaces remain unambiguous.

## Tailscale Deployment

HLIMS has no runtime dependency on Tailscale. A Tailscale node named `go` can
run HLIMS on loopback and publish it to the Tailnet with Tailscale Serve:

```bash
hlimsd --listen=127.0.0.1:8080 --sqlitedb=/var/lib/hlims/hlims.db
tailscale serve --bg --http=80 http://127.0.0.1:8080
```

With MagicDNS enabled, permitted Tailnet devices can use paths such as
`http://go/badger/opencode/production`. Do not enable Funnel, expose the HLIMS
listener publicly, or trust proxy identity headers from an unprotected listener.

Tailscale remains a first-class Network kind and endpoint option. Other users
can place the same HTTP server behind their preferred VPN or private reverse
proxy.

## Upstream History

- [Fork point and upstream integration policy](docs/upstream.md)
- [Original golink documentation](docs/upstream-golink.md)
