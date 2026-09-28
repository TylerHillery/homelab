# HLIMS Domain Model

Status: current.

HLIMS records owned hardware, runnable machines, network addresses, services,
and reachable endpoints.

## Identity

- Internal UUIDv7 IDs define database relationships.
- Public NanoIDs identify API resources.
- Editable names are display values.
- Stable lowercase slugs identify named resources in routes and automation.
- Service and Instance names should match their lowercase, hyphenated slugs so
  CLI listings and `go/<machine>/<service>/<instance>` paths use the same names.
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

A Service is a conceptual application. An Instance places one Service on one
Machine and records its backend port. An Instance Endpoint records one
client-facing scheme, address, port, and path. The resolver redirects clients to
an endpoint and never proxies traffic. Its host selection is explicit: DNS or
direct IP; legacy `auto` endpoints use DNS when available. An Address records a
device's IP and optional DNS name, but does not imply a service listens on both.
Record separate endpoints only for URLs that actually serve the application.

An Ingress Route associates a client-facing Instance Endpoint with an ingress
Instance (such as Caddy), distinguishing a proxy upstream, static document root,
or redirect target. The application Instance remains associated with the host
Machine even if its own backend port is not publicly exposed. These are
inventory snapshots, not deployed web-server configuration.

## Core Relationships

```text
Manufacturer -> Product -> Asset -> Machine
Machine Provider -> Area -> Machine
Network -> Address -> Machine, Area, or network Asset
Service -> Instance -> Instance Endpoint -> Address
DNS Zone -> DNS Record -> Address; Instance Endpoint -> DNS Record
Instance Endpoint -> Ingress Route -> ingress Instance
Purchase -> one or more Assets
```

The API contract and persistence constraints are authoritative:

- [`api/openapi.yaml`](../../services/hlims/api/openapi.yaml)
- [`db/migrations`](../../services/hlims/db/migrations)
