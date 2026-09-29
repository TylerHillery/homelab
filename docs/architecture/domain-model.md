# HLIMS Domain Model

Status: current.

HLIMS records owned hardware, runnable machines, network addresses, services,
and reachable endpoints.

## Identity

- Internal UUIDv7 IDs define database relationships.
- Public NanoIDs identify API resources.
- Editable names are display values.
- Stable lowercase slugs identify named resources in routes and automation.
- Service and Instance display names may preserve product branding, such as
  `OpenCode`, `cAdvisor`, or `HLIMS`. Lowercase, hyphenated slugs are the stable
  identifiers used in CLI lookups and resolver paths.
- Canonical `go/<service>/<instance>` links identify a deployment without
  assuming it has a Machine. Use `?host=<machine-slug>` or `?host=managed` only
  when the same Service and Instance slugs identify multiple deployments.
- External serial numbers, provider IDs, and system UUIDs remain attributes.

## Inventory

| Resource | Meaning |
|---|---|
| Manufacturer | Canonical hardware vendor |
| Product | Reusable hardware model and kind-specific specification |
| Asset | One owned or managed physical item |
| Purchase | One transaction associated with one or more Assets |
| Machine Provider | Environment responsible for Machines and Areas |
| Area | High-level physical or provider location |

Assets may be placed in an Area, contained by a system or rack Asset, or left
unplaced. Contained Assets inherit their effective Area. Product port profiles
describe grouped model capabilities, not physical connections.

A Purchase stores the full order total and only the selected Assets and lines
worth inventorying. A selected line can be a bundle or kit with several
Assets; its pre-tax subtotal is separate from the order total. Omitted items,
tax, and shipping are not silently attributed to a single Asset. Purchases may
have an external order reference and receipt links; no payment credentials are
stored.

Selected lines can also represent household equipment that is worth inventorying
but excluded from homelab-only spending. Inclusion belongs to the Purchase line,
not the Product or Asset: ownership and expense classification are different.
Order total, tracked-item subtotal, and homelab-only subtotal are separate sums.

Wi-Fi specifications belong to the reusable Product, alongside wired port
profiles. Individual router or access-point Assets share those capabilities but
retain their own placement, role notes, and Purchase. Product capabilities do
not identify which installed port has a cable attached.

## Compute

A Machine is a runnable bare-metal or virtual environment. Bare-metal Machines
may reference a system Asset. Virtual Machines may reference a parent Machine.
Machine capacity describes logical resources; physical components remain Asset
inventory. `cpuCount` records cores on physical Machines or vCPUs on virtual
Machines. `cpuThreadCount` optionally records logical hardware threads on a
physical Machine.

An installed processor or memory module is an Asset contained by a Machine's
backing system Asset. Its Product records reusable specifications: processor
model, generation, and codename, or memory technology and module capacity.
The console reads those relationships; virtual Machines show assigned capacity
without claiming ownership of their host's physical components.

A Machine User records a login name, not credentials or SSH trust. One preferred
user can guide clients when constructing SSH commands.

## Networking

A Network is an address scope such as LAN, tailnet, cloud network, public
Internet, or a machine's loopback. An Address belongs to one Network and one
Machine, Area, or eligible network-equipment Asset.

An owned DNS Zone has DNS Records. An A or AAAA Record points to an Address;
several names may point to the same IP. An Instance Endpoint can select a DNS
Record for its URL instead of assuming one DNS name per Address. The record
describes the observed DNS mapping; the DNS provider remains authoritative.

The current model inventories addresses, not complete routing or cabling. Ports,
interfaces, prefixes, pools, VLAN attachment, and routed relationships are added
only when a real consumer needs them. See the [HLIMS Roadmap](hlims-roadmap.md).

## Services

A Service is a conceptual application. A machine-hosted Instance associates it
with one Machine and optionally records a backend port when it listens on one.
Workers and ingress-only processes need no port or client-facing URL, but remain
visible on their Machine and in the Services catalog. A managed Instance has no
Machine or backend port; it records the external provider instead. A
machine-hosted Instance Endpoint records a verified scheme, machine Address,
port, and path.
A managed Endpoint records a verified direct HTTPS URL, including any project
path or query, without inventing a Machine, Address, or owned DNS Zone. Multiple
labeled managed URLs can link to a tenant and a particular project. The resolver
redirects clients to a preferred Endpoint when one exists and never proxies
traffic. Machine endpoints select DNS or direct IP; `auto` uses DNS when
available. An Address does not imply a service listens on both names and IPs.

An Ingress Route associates a client-facing Instance Endpoint with an ingress
Instance (such as Caddy), distinguishing a proxy upstream, static document root,
or redirect target. The application Instance remains associated with the host
Machine even if its own backend port is not publicly exposed. These are
inventory snapshots, not deployed web-server configuration.

A Machine-local Deployment optionally groups Instances under a Docker Compose
project. It records the project name, working directory, ordered files, and
each member Instance's service key. A systemd-managed Instance instead records
its own unit name, scope, and user where relevant; sharing the same systemd
manager does not make multiple units a Deployment. Without either mechanism,
an Instance remains independently inventoried. An Instance Dependency identifies
the exact provider Instance consumed by another Instance, even across
Deployments; it does not encode startup order or live health.
A `static_content` Instance in a Compose Deployment instead identifies a path
to files published by another member. Its `served_by` relation points at that
server Instance; the static files are not a fabricated container.

## Core Relationships

```text
Manufacturer -> Product -> Asset -> Machine
Machine Provider -> Area -> Machine
Network -> Address -> Machine, Area, or network Asset
Service -> Instance (Machine or managed provider) -> Instance Endpoint (Address or direct URL)
Machine -> Compose Deployment -> optional member Instances; Instance -> provider Instance (uses)
Machine -> Instance -> optional systemd unit (scope and user)
DNS Zone -> DNS Record -> Address; Instance Endpoint -> DNS Record
Instance Endpoint -> Ingress Route -> ingress Instance
Purchase -> one or more Assets
```

The API contract and persistence constraints are authoritative:

- [`api/openapi.yaml`](../../services/hlims/api/openapi.yaml)
- [`db/migrations`](../../services/hlims/db/migrations)
