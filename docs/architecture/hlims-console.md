# HLIMS Console Pages

Status: Machine Topology, Inventory, and Orders are current. DNS and Services
page concepts remain deferred.

## Purpose

The current `/console/` is **Machine Topology**: Machine Provider → Area or
region → Machine → nested Machine. Its cards summarize recorded capacity,
addresses, users, hosted Services, Instances, and available endpoints. Favorites
provide a shortcut to Machines. Ingress-only Instances with no direct link stay
in the API and appear through the endpoints they publish rather than as empty
service cards. This view should not also be a DNS or purchasing dashboard.

HLIMS needs other ways to explore the same relational inventory. Each page
should answer a distinct question, then link to related records rather than
copying every field into Machine Topology.

## Proposed Pages

| Page | Primary question | Source entities |
|---|---|---|
| Machine Topology | What runs where and how can I reach it? | Machine Providers, Areas, Machines, Addresses, Machine Users, Services, Instances, Endpoints |
| Inventory | What physical equipment do I own, including spares? | Manufacturers, Products, Assets, placement, installed components |
| Orders | What did I buy, when, for how much, and which Assets were included? | Purchases, selected lines, Purchase Assets, links |
| Networks & DNS | Which addresses and names exist, and what do they point to? | Networks, Addresses, DNS Zones, DNS Records |
| Services & Ingress | Which applications run, and how do public URLs reach their backends? | Services, Instances, Endpoints, Ingress Routes |

### Inventory

Separate **Product** (a reusable model and its specifications) from **Asset**
(a particular physical item). Show Assets grouped by kind and placement, with
filters for memory, network equipment, systems, racks, and uninstalled parts.
An uninstalled memory module must remain visible even though it cannot appear
under any Machine card. A Product with no Assets is a catalog entry, not owned
stock. Show model specifications, physical units, and where each unit is
installed or stored. A system Asset can link to its Machine and contained
processor, memory, and drive Assets.

The compact grid leads with filters and Products. A Product or Asset opens a
right-side detail drawer without losing the grid or its Machine/Purchase links.
The Product view shows model specifications, port groups, source links, and all
owned units; selecting an Asset adds its location, role notes, installed Machine,
Purchase, parent Asset, and recursively contained components. A system shows its
installed CPU, RAM, drives, and network adapters; a rack can show contained
systems and their nested parts. Each component links to its own Asset drawer,
while its Machine and Purchase links remain available. The panel has public-ID
deep links, a close control, keyboard and Escape behavior, and a full-screen
layout on small devices.

Do not invent a Product for unidentified spare RAM. Wait for module labels and
capacity before adding its physical Assets. Product port profiles describe model
capabilities, not which installed jacks are connected; a future connections
view needs verified port labels and cable endpoints.

### Purchases

List purchases independently of Machine placement. Show date, source, private
order reference, full order total, selected inventory lines, included Assets, and
receipt or listing links. Omit retailer lines unrelated to the homelab. A line
may describe a kit of multiple physical Assets without inventing a Product for
the kit. Selected subtotals exclude tax, shipping, and omitted items; never
present the full mixed-order total as the cost of one included Asset. Treat a
bundle as one Purchase with multiple Assets, keep currencies separate, and
distinguish zero cost from an unknown price.

An inventoried shared-home item may be excluded from the homelab-only spending
sum per selected Purchase line. Keep its Asset, Product, order line, and full
order total visible. Above the order cards, show only the homelab-only selected
subtotal, before tax and shipping; keep individual full totals on their cards.

### Networks & DNS

Keep owned DNS Zones and their Records here, not on Machine Topology. Show a
record's name, type, target address, Network, and owner Machine or network
equipment. Several names may point to one address. A DNS Record should link to
the target Machine in Machine Topology and to any matching service endpoints.
An Address is not proof that a particular service listens on it; an endpoint
describes a verified access URL. Display loopback addresses as local-only.

### Services & Ingress

Show Service → Instance → client-facing Endpoints, including scheme, selected
DNS name or IP, and network scope. For endpoints with an Ingress Route, show the
ingress Instance and whether it proxies a backend, serves static files, or
redirects. Distinguish a backend port from the published endpoint port.

## Navigation Rules

- Keep the Machines, Inventory, and Orders navigation visible on all three
  pages. Do not include navigation to unimplemented sections.
- Give each resource a stable link keyed by its public ID, independent of its
  editable name or slug. A link from an Asset, DNS Record, or Service to a
  Machine should land on the correct Machine card, including nested Machines.
- Preserve the user's page and filters when following a relationship and going
  back. Provide a clear path from the detail back to its containing list.
- Keep HTML escaped and distinguish destinations that are not reachable from
  the current browser. Never turn stored network inventory into an unverified
  "open" link.
- Keep the OpenAPI reference at `/api-docs/` for full API and schema inspection;
  the console pages are task-focused projections, not replacements for the API.

## Implementation Order When Resumed

1. Finish public-ID deep links to nested Machines; current inventory links can
   target root Machine cards, but lazily loaded descendants need a focus route.
2. Add Networks & DNS and Services & Ingress pages, then wire cross-links from
   record → address → Machine and endpoint → ingress/backing Instance.
3. Add editing flows only for workflows that cannot be handled comfortably by
   the existing API/CLI. Keep OpenAPI and SQLite constraints authoritative.

Decide filters, page density, and which record types merit dedicated detail
views using real staging inventory before implementation.
