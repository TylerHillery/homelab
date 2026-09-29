# HLIMS Console Pages

Status: Machines (including Deployments), Services, Inventory, and Orders are
current. Networks & DNS remains deferred.

## Purpose

`/console/machines/` is **Machine Topology**: Machine Provider → Area or
region → Machine → nested Machine. Its cards summarize recorded capacity,
addresses, users, hosted Services, Instances, and available endpoints. Favorites
provide a shortcut to Machines. Internal workers and ingress-only Instances
without direct links still appear on their Machine cards, without invented
resolver URLs. This view should not also be a managed-service, DNS, or
purchasing dashboard.

HLIMS needs other ways to explore the same relational inventory. Each page
should answer a distinct question, then link to related records rather than
copying every field into Machine Topology.

## Proposed Pages

| Page | Primary question | Source entities |
|---|---|---|
| Machine Topology | What runs where and how can I reach it? | Machine Providers, Areas, Machines, Deployments, Addresses, Machine Users, Services, Instances, Endpoints |
| Inventory | What physical equipment do I own, including spares? | Manufacturers, Products, Assets, placement, installed components |
| Orders | What did I buy, when, for how much, and which Assets were included? | Purchases, selected lines, Purchase Assets, links |
| Networks & DNS | Which addresses and names exist, and what do they point to? | Networks, Addresses, DNS Zones, DNS Records |
| Services | Which applications and managed subscriptions exist, and where can I open them? | Services, Instances, Endpoints, Machines |

### Inventory

Separate **Product** (a reusable model and its specifications) from **Asset**
(a particular physical item). Show Assets grouped by kind and placement, with
filters for memory, network equipment, systems, racks, and uninstalled parts.
An uninstalled memory module must remain visible even though it cannot appear
under any Machine card. A Product with no Assets is a catalog entry, not owned
stock. Show model specifications, physical units, and where each unit is
installed or stored. A system Asset can link to its Machine and contained
processor, memory, and drive Assets.

The compact grid leads with filters and uniformly sized Product cards. Keep
variable-length Asset lists, placement, and notes in click-controlled floating
panels that open without shifting other cards; panels can open above the card
near the bottom of the viewport. A Product or Asset also opens a right-side
detail drawer without losing the grid or its Machine/Purchase links.
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
capabilities, not which installed jacks are connected. Keep site-specific cable
drawings in private documentation rather than presenting them as a universal
HLIMS topology.

### Purchases

List purchases independently of Machine placement. Show date, source, private
order reference, full order total, selected inventory lines, included Assets, and
receipt or listing links. Omit retailer lines unrelated to the homelab. A line
may describe a kit of multiple physical Assets without inventing a Product for
the kit. Selected subtotals exclude tax, shipping, and omitted items; never
present the full mixed-order total as the cost of one included Asset. Treat a
bundle as one Purchase with multiple Assets, keep currencies separate, and
distinguish zero cost from an unknown price.

Display one Purchase per row in a single list, newest purchase date first.
Undated records go last; do not split the list into yearly card grids.

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

### Services

Show Service → Instance → optional verified Endpoints, with the host Machine as
optional context. Managed Instances name their external provider; their direct
HTTPS URLs may include labeled project links. Machine-hosted Instances may
record a backend port but workers need neither a port nor a URL. Never turn a
declared or observed process into an unverified link. Keep service cards a
consistent height: show a primary route when one exists, with the
variable-length instance and endpoint detail in click-controlled floating
panels that can open above the card near the bottom of the viewport. A
future ingress detail view can show which Instance proxies, serves, or redirects
an Endpoint; the current catalog lists direct links without claiming to manage
the external provider or its DNS.

When the same Service has Instances in several Deployments, show a separate
section for each Deployment inside that Service's detail panel. Its member
list includes only that Deployment's Instances; independent and managed
Instances keep their own section. Inside each Machine card, Compose Deployments
group only their assigned Instances, with service keys, file paths, and
cross-Deployment usage. Systemd-managed Instances appear together by
system/user manager and user, showing full unit names without inventing a
Deployment for each unit.
Static-file Instances can appear under their Compose project with a content
path and a `served_by` link to the actual server, without being shown as a
separate container.
Independent Instances keep the original Machine Service layout. Neither view
reports “running” from static inventory: runtime status is not checked yet.

## Navigation Rules

- Keep the Machines, Services, Inventory, and Orders navigation visible on all
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
2. Add Networks & DNS, then wire cross-links from record → address → Machine
   and endpoint → ingress/backing Instance.
3. Add editing flows only for workflows that cannot be handled comfortably by
   the existing API/CLI. Keep OpenAPI and SQLite constraints authoritative.

Decide filters, page density, and which record types merit dedicated detail
views using real staging inventory before implementation.
