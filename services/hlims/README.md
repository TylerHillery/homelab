# HLIMS

HLIMS is the Home Lab Information Management System. It catalogs infrastructure
and resolves inventory-backed paths to client-reachable service URLs without
proxying application traffic.

```text
http://go/opencode/production?host=badger
http://go/grafana/production
http://go/badger/5173?via=tailnet
```

The `go` host is the common entry point. Canonical paths use
`go/<service>/<instance>` for both machine-hosted and managed deployments. When
several deployments have the same Service and Instance slugs, select one with
`?host=<machine-slug>` or `?host=managed`. Machine topology links include the
host selector so they always target the selected Machine. Ad hoc paths with no
recorded Service still use `go/<machine>/<port>`. These segments use stable
lowercase, hyphenated slugs for URLs and CLI lookup.
Display names are separate and can preserve official branding such as
`OpenCode`, `cAdvisor`, or `HLIMS`.

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

### Framework WSL staging

The repository's [`hlims-staging.service`](hlims-staging.service) runs an
independent, persistent user service on `127.0.0.1:8081` (development uses
`8090`; the in-memory `serve` task uses `8080`). From this directory:

```sh
go build -o "$HOME/.local/bin/hlimsd" ./cmd/hlimsd
install -m 644 hlims-staging.service "$HOME/.config/systemd/user/hlims-staging.service"
systemctl --user daemon-reload
systemctl --user enable --now hlims-staging.service
```

Open <http://localhost:8081/console/machines/> on the Framework, including from Windows
when WSL localhost forwarding is enabled. The database is
`~/.local/state/hlims/staging.db`; do not add it to Git. After rebuilding, run
`systemctl --user restart hlims-staging.service`. Inspect with
`systemctl --user status hlims-staging.service` and
`journalctl --user -u hlims-staging.service`. Back up the running database using
Python's SQLite online-backup API:

```sh
python3 -c 'import sqlite3,sys; src=sqlite3.connect(sys.argv[1]); dst=sqlite3.connect(sys.argv[2]); src.backup(dst); dst.close(); src.close()' \
  "$HOME/.local/state/hlims/staging.db" /path/to/private/backup.db
```

The user service starts with the WSL distribution; systemd does not start WSL
at Windows boot.

## Interfaces

The server provides:

| Path | Purpose |
|---|---|
| `/` | Redirect to Machine Topology |
| `/api/v1` | Versioned JSON API |
| `/console/machines/` | Machine Topology, including single-host Compose projects and systemd units |
| `/console/services/` | Service-first catalog of hosted and managed instances |
| `/console/inventory/` | Owned Products and physical Assets, including uninstalled parts |
| `/console/orders/` | Purchase history with selected inventory lines and related Assets |
| `/openapi.yaml` | OpenAPI source |
| `/api-docs/` | Self-hosted API reference |

The OpenAPI source is [`api/openapi.yaml`](api/openapi.yaml). Generated server
and client code is committed under `generated/api` and verified by
`mise run check`.

Inventory groups physical Assets under reusable Product models and shows their
placement, installed Machine, and related Purchase. Orders keep full transaction
totals separate from the pre-tax subtotal of selected inventory lines; unrelated
retailer items can be omitted. Orders are listed one per row, newest first. A
tracked household purchase can be excluded from the homelab spending subtotal
without losing its Assets. Product and Asset details open in a shareable side
panel. Inventory Product cards have a uniform height; their Asset lists open
in floating panels without shifting the grid. The two pages still link back
to Machine Topology and to each other. See the
[console page specification](../../docs/architecture/hlims-console.md) for
current boundaries and deferred sections.

The Services page keeps its cards a uniform height. Select a card's Instances
and endpoints control to inspect its full set of links in a floating panel
without moving other cards. Machine-hosted workers and ingress-only processes
can have no port or endpoint; they remain visible on Machines and Services
without a dead "Open" link. A port is recorded only for a verified listener.

Deployments optionally group Machine-hosted Instances by Compose project,
recording a project name, working directory, ordered Compose files, and service
keys. Systemd units instead belong to individual Instances with a scope and
user where applicable; Machine cards show units sharing a manager together,
without inventing Deployments for them. Compose groups contain only their
member Services and Instances; instances without either mechanism retain their
existing Service grouping. The global Services catalog groups Instance details
by Deployment. Static files produced in a Compose project can be grouped with
the project as `static_content` and linked to the exact publishing Instance as
`served_by`, without inventing another Compose service. Cross-Deployment usage
identifies exact provider Instances, not start order or live health. Compose
projects and systemd managers use locally bundled icons from the
[Docker Simple Icons artwork](https://github.com/simple-icons/simple-icons/blob/develop/icons/docker.svg)
and the [systemd project logo](https://github.com/systemd/systemd/blob/main/docs/assets/systemd-logo.svg).
Staging is never seeded with demo Deployments; `mise run dev` uses disposable
examples with shared PostgreSQL, separate Compose projects, independent local
Instances, and two systemd units on one Machine.

Selecting a system or rack Asset reveals contained Assets recursively. Component
drawers link to their parent Asset as well as the associated Machine and
Purchase. Model references are shown directly on Asset details when available.

The console shows device addresses and every recorded service endpoint. Set an
endpoint's `hostType` to `dns` or `ip` to choose the actual URL host; the default
`auto` chooses DNS when present, otherwise IP. Add separate endpoints when both
hosts serve the application. An address by itself does not imply that the
service listens on it. The existing service menu lists the full URLs for each
endpoint; the primary link still uses the preferred resolver path. Direct-IP
HTTPS may give a certificate warning when its certificate covers only the DNS
name. Machine cards show an OS line and lightweight inline facts for cores (or
vCPUs), threads, memory, and storage when those facts have been recorded. Hover
or focus a fact for the available CPU model and generation, memory type, storage
interface, or OS details. SSH controls appear when a Machine has a recorded
login user and address. Click or keyboard-activate the network-address label to
inspect IPs without moving the rest of the card.
Physical CPU model, generation, and RAM type come from installed
processor and memory Assets under a Machine's backing system Asset. VM cards
show allocated resources without inheriting the host's physical hardware.
Known OS names display bundled icons from `console/static/os-*.svg`; Ubuntu
shows Linux and Ubuntu, and OPNsense shows FreeBSD and OPNsense. These base OS
labels are derived from the recorded OS name, not a second inventory field;
unknown names keep their text label without an icon. These are local static
assets, whereas uploaded Machine Provider and Service logos stay in SQLite
and are served from their existing API routes. Source an official icon when
adding a Service and upload it as PNG, JPEG, or WebP (up to 1 MiB), not SVG.
The bundled OS artwork sources are
[Linux (Tux, with its white background removed)](https://github.com/edent/SuperTinyIcons/blob/master/images/svg/linux.svg),
[Ubuntu](https://api.iconify.design/logos/ubuntu.svg),
[FreeBSD Beastie](https://commons.wikimedia.org/wiki/File:Daemon-phk.svg),
[Windows](https://api.iconify.design/logos/microsoft-windows-icon.svg),
[illumos](https://commons.wikimedia.org/wiki/File:Illumos_textlogo.svg), and
[OPNsense](https://github.com/simple-icons/simple-icons/blob/develop/icons/opnsense.svg).
Recorded IPs are inventory snapshots, not live DHCP leases; update an address
if its assigned IP changes. Network facts are stored in `networks`,
`addresses`, `instances`, and `instance_endpoints`; add real inventory through
the API or CLI, not demo seeding.

`dns_zones` and `dns_records` inventory owned domains and A/AAAA mappings. An
endpoint can reference a DNS record when several names share one address.
`ingress_routes` records which ingress Instance proxies, serves, or redirects
that endpoint. WSL-only applications use a loopback Network and `127.0.0.1`
endpoints, accessible only on the local host. If Tailscale Serve publishes one,
record a separate tailnet HTTPS endpoint and its proxy route to the loopback
backend.

`hlims` is an API-only client. It does not open the SQLite database.

```sh
hlims products list
hlims deployments list
hlims instance-dependencies list
hlims machines list --slug badger
hlims services get opencode
hlims instances list --service opencode
hlims services logo set opencode --file /path/to/verified-logo.png
hlims services patch opencode --file changes.json
hlims resolve instance opencode production --host badger
hlims resolve instance grafana production
hlims open opencode/production?host=badger
hlims open grafana/production
hlims console
```

The client targets `http://127.0.0.1:8080/api/v1` by default. Set
`HLIMS_API_URL` or pass `--api-url` for another deployment. Data commands emit
JSON. Create, patch, and update commands read JSON from standard input by
default. `patch` merges only supplied fields and uses an ETag to reject stale
updates; `update` replaces the complete resource. Service and Machine Provider
logo commands accept a PNG, JPEG, or WebP file (up to 1 MiB).
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
