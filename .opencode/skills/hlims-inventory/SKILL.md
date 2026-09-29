---
name: hlims-inventory
description: Use ONLY when discovering, importing, correcting, or reconciling HLIMS machine, hardware, purchase, network, DNS, and service inventory through its CLI or REST API.
---

# HLIMS inventory

Inventory real, verified facts in the private deployment; keep this public skill generic. Read `services/hlims/README.md`, `docs/architecture/domain-model.md`, and `services/hlims/api/openapi.yaml` when their semantics or request fields matter. The API schema is authoritative. Never seed production/staging from the demo or edit SQLite directly for routine imports.

## Quick path: add or correct a Service

1. Confirm the target API; staging uses `HLIMS_API_URL=http://127.0.0.1:8081/api/v1`. **Look up by slug first:** `hlims services get opencode`, `hlims services list --slug opencode`, `hlims instances list --service opencode`. Reuse existing IDs and check for duplicate slugs/instances before writing.
2. Set a human-facing **display name using the product's actual branding**, e.g. `OpenCode`, `cAdvisor`, `Grafana Alloy`, or `HLIMS`. Set its **lowercase hyphenated slug** separately (`opencode`, `cadvisor`, `grafana-alloy`, `hlims`). Slugs, not display names, identify Services in CLI commands and `go/<service>/<instance>` URLs. Do not force the display name to match the slug. Keep Instance names readable and their slugs stable.
3. **Find an authentic logo while creating the Service.** Check official site/repository artwork or a user-provided project, inspect/crop it for a small square, and convert SVG to PNG if necessary. Upload with `hlims services logo set SERVICE_SLUG --file /private/path/logo.png` (PNG/JPEG/WebP, max 1 MiB). Re-list and verify `hasLogo: true`. If no verified logo exists, ask the user for a source instead of silently leaving initials or using an unrelated mark.
4. Record verified Instances, including port-less workers with no URL. Machine hosting requires a real Machine; `port` is optional. Managed hosting requires `managedProvider`, without a Machine or port. Create Endpoints only for verified client-facing URLs; otherwise there is no working resolver link.
5. Verify the Service, Instances, logo, `/console/services/`, and `/console/machines/`. For corrections, use `hlims services patch SERVICE_SLUG --file changes.json` or `hlims instances patch ID --file changes.json`; patch merges fields and refuses to overwrite a newer record. Only use full `update` when deliberately replacing every writable field. Keep private names, receipts, and URLs in staging, not this public skill.

## Start with existing records

From `services/hlims/`, run the CLI with the repository's pinned Go toolchain (`mise exec -- go run ./cmd/hlims ...`) or an already-built `hlims` binary. Its default API is `http://127.0.0.1:8080/api/v1`; for local staging use `export HLIMS_API_URL=http://127.0.0.1:8081/api/v1`. Confirm which deployment you are touching before writing. HLIMS has no application authentication: connect only through trusted private ingress. All data commands return JSON and never prompt.

```sh
hlims machine-providers list | jq '.items[] | {publicId,name,slug}'
hlims areas list | jq '.items[] | {publicId,name,slug,machineProviderPublicId}'
hlims manufacturers list | jq '.items[] | {publicId,name,slug}'
hlims products list | jq '.items[] | {publicId,name,kind,partNumber,manufacturerPublicId}'
hlims assets list | jq '.items[] | {publicId,name,productPublicId,placement}'
hlims purchases list | jq '.items[] | {publicId,source,orderReference,assetPublicIds}'
hlims machines list | jq '.items[] | {publicId,name,slug,assetPublicId,areaPublicId}'
hlims machine-users list | jq '.items[] | {publicId,machinePublicId,username}'
hlims networks list | jq '.items[] | {publicId,name,kind,areaPublicId}'
hlims addresses list | jq '.items[] | {publicId,machinePublicId,networkPublicId,address}'
hlims services list | jq '.items[] | {publicId,name,slug}'
hlims instances list | jq '.items[] | {publicId,name,slug,servicePublicId,hostingKind,machinePublicId,managedProvider}'
hlims instance-endpoints list | jq '.items[] | {publicId,instancePublicId,addressPublicId,directUrl,name}'
```

Compare normalized names, aliases, model/part numbers, placement, and IDs before creating anything. Reuse existing manufacturer, Product, Service, network, area, and system Asset; never create an `HP` manufacturer if the existing canonical name is `Hewlett-Packard`. Use existing capitalization for manufacturers and products. Reconcile conflicting duplicates instead of importing a second copy. For DNS zones, DNS records, and ingress routes (not exposed as CLI resource commands), check `curl -fsS "$HLIMS_API_URL/dns-zones"`, `/dns-records`, and `/ingress-routes` before POSTing.

### Service identity and artwork

- Service display names are free to reflect official branding; **the slug is the CLI- and URL-friendly identity**. Derive a lowercase hyphenated slug on creation, preserve that slug when changing a display name, and compare slugs and names before creating duplicates. This applies equally to internal workers and managed cloud Services.
- **Find and attach a real logo whenever adding a new Service.** First check `services list` for reuse and `hasLogo`; do not replace a verified existing logo. For a new Service, research the project's official website/repository assets or user-provided project artwork, select a recognizable icon suitable for a small square, and obtain a permitted PNG, JPEG, or WebP file under 1 MiB. Rasterize SVG artwork or crop official artwork when necessary; inspect the result. Upload via the CLI logo command, then GET/re-list to verify `hasLogo: true`. Do this during the same import, including URL-less workers; never leave a newly added Service on initials by default. If no authentic artwork can be found, ask the user for a source rather than silently skipping the logo or substituting an unrelated brand's mark. Keep images/private inventory in the ignored staging database, not in tracked files.

For a vetted local PNG:

```sh
hlims services logo set SERVICE_SLUG --file /private/path/logo.png
hlims services get SERVICE_SLUG | jq '{name,slug,hasLogo}'
```

## Discover before recording

- Determine the authoritative host and virtualization boundaries. Use an approved existing SSH/Tailscale path; inspect one machine at a time, read-only. For Linux: `hostnamectl`, `cat /etc/os-release`, `lscpu`, `free -b`, `lsblk -b -J -o NAME,MODEL,SIZE,TYPE,ROTA,TRAN,MOUNTPOINT`, `ip -j addr`, `ip -j route`, `ss -ltnp`, `ss -lunp`, `getent passwd`, `systemctl --type=service --state=running`, `tailscale status --json`, `tailscale serve status --json`. Use `dmidecode -t 17` (if permitted) for DIMMs and `smartctl -i` for actual drive identity where needed. Don't confuse a partition/loop volume with a drive.
- On FreeBSD/OPNsense, use `uname -a`, `sysctl hw.model hw.ncpu hw.physmem`, `ifconfig -a`, `geom disk list`, `sockstat -l`, and read-only configuration inspection instead of assuming Linux tooling. For Windows, use PowerShell `Get-CimInstance Win32_Processor`, `Get-CimInstance Win32_PhysicalMemory`, `Get-PhysicalDisk`, `Get-NetIPAddress`, `Get-NetAdapter`, and `Get-NetTCPConnection -State Listen`. Treat WSL as a separate guest Machine; its guest IPs and usable RAM are not the host's installed hardware.
- Identify physically installed modules from readable labels, DMI/SMBIOS, or reliable manufacturer documentation and matching model numbers; use invoices only when installation is corroborated. Verify each CPU's physical core and logical-thread counts; for VMs, `cpuCount` is allocated vCPUs, not host cores, and physical CPU/RAM components belong only to the host's system Asset. Verify the OS/version/kernel and disk model/interface rather than inferring them from a product family or purchase. If sources disagree, leave uncertain fields unset and ask for photos, labels, firmware details, or permission to search the web for exact documentation.
- Enumerate real interactive/SSH-capable users, not every system account. Verify actual login capability before marking `machine-users.isPreferred`; do not assume `root`, `tyler`, or `github` exists everywhere. Never record credentials or authentication material.
- Cross-check observed interfaces with DHCP/DNS, router, cloud metadata, and Tailscale as appropriate. Identify LAN, tailnet, cloud VPC, public, and loopback separately, including IPv6 when usable. Dynamic DHCP/WSL IPs are snapshots; do not invent permanent addresses, public exposure, routes, or services. Directly Tailscale-enabled machines do not require an extra subnet route. Do not change a machine's networking while discovering inventory.
- Distinguish a listening socket (`ss`/`sockstat`/Windows listener), an application's configured port, an ingress listener (Caddy/Tailscale Serve), and an externally reachable URL. Check reverse-proxy and Serve configuration, DNS resolution from the intended network, and a non-mutating request such as `curl -I` where possible. Container or loopback-only listeners do not prove reachability on the LAN, tailnet, or public IP. Attribute backend services to the machine where they actually run, with ingress routes on the proxy machine. Do not assume HTTPS works on a raw IP when the certificate is for a name.

## Normalize capacities and ownership

- `memoryBytes` on a physical Machine represents **installed capacity**, not OS-usable memory (`free`, `hw.physmem`) reduced by reserved regions. RAM specs use binary units: one 8 GiB module = `8589934592` bytes, two = `17179869184` (16 GiB), one 32 GiB module = `34359738368`. Match module count, Product `memorySpec.capacityBytes`, and installed child Assets. If only an OS figure is available, do not silently label it installed capacity; ask for hardware evidence. For a VM, use its verified assigned plan/allocation where available, otherwise state that the guest report is usable memory.
- Drives are usually marketed in **decimal** GB/TB; match Product `driveSpec.capacityBytes` to verified model/label or device size, not filesystem free/used space or arbitrary binary-to-decimal conversion. `Machine.storageBytes` should reflect the verified relevant storage allocation, not a sum of partitions, loop devices, or redundant volumes. Check disk model and bus before choosing `storageMediaKind`/`storageInterfaceKind`.
- Product is a reusable make/model/specification; Asset is one owned item. Create one Asset per physical DIMM, SSD, NIC, or system and put installed components beneath the system Asset (`placement.type=asset`, `parentAssetPublicId`). Uninstalled parts remain placed in an Area or unplaced. Associate a bare-metal Machine with its system Asset; use a VM's `parentMachinePublicId` only when verified. Never attach the host's parts to the guest.
- Purchase lines describe only selected owned items from an order; `totalPriceCents` is the entire transaction (including unrelated items, tax, and shipping), while line `subtotalCents` is the selected item's pretax, preshipping amount. Record a real order reference once, link owned Assets, keep currencies separate, and set `includeInHomelabTotal: false` for tracked shared-home equipment. A kit purchase line can cover multiple module Assets; do not multiply the kit price by its number of sticks. Leave prices/details unknown rather than manufacturing them.

## Write in dependency order, then verify

1. Reuse or create machine-provider → area, manufacturer → Product (and its exact specs/links) → physical Asset → installed child Assets; create/update the Machine pointing to its system Asset. Link Purchase after its Assets exist; optionally attach genuine support/manual/retailer links to the right Product/Asset. Record provider-native/cloud resources as appropriate without inventing physical ownership.
2. Reuse or create networks → addresses attached to **exactly one** Machine, Area, or Asset → verified machine users. `Address.dnsName` may contain one observed name; DNS zones and A/AAAA records model owned names and can associate multiple names with a single address. A recorded Address proves nothing about listeners.
3. Reuse or create Service → Instance. An Instance does **not** require an Endpoint or URL: track internal workers, agents, replication jobs, and ingress processes without inventing one. A machine-hosted Instance (`hostingKind: machine`) needs its real `machinePublicId`; supply `port` only if a listener is verified. Add an Instance Endpoint only for a **verified client-facing URL**: a Machine Address, `scheme`, external `port`, `basePath`, `hostType` (`dns`, `ip`, or `auto`), and preferred choice. A local `127.0.0.1` endpoint belongs to a loopback Network; add a *separate* HTTPS tailnet endpoint if Tailscale Serve proxies it. An ingress route relates an endpoint to a verified proxy/static/redirect Instance and its actual target. A managed Instance (`hostingKind: managed`) instead needs `managedProvider`, **no Machine or port**, and may have endpoints with verified HTTPS `directUrl` (including any project path or query). Label separate managed project links distinctly and set one preferred route if there are links. Canonical paths use `go/<service>/<instance>` **only when an Endpoint exists**; add `?host=<machine-slug>` or `?host=managed` when selection is needed. Never create a fake Machine/Address or claim ownership of a SaaS vendor's DNS Zone.

CLI `create` reads a single JSON document from stdin or `--file`; `get ID_OR_SLUG`, `patch ID_OR_SLUG`, `update PUBLIC_ID`, and `delete PUBLIC_ID --yes` exist for CLI resources. `patch` merges a JSON object with the current resource, strips response-only fields, and sends `If-Match` using the GET ETag: a concurrent update returns HTTP 412 instead of silently losing data. Null deletes a field; arrays replace whole arrays. `update` remains a **full PUT replacement**. Consult `*Write` schemas in `api/openapi.yaml` before submitting. For example, with `MACHINE_ID` set to an existing ID:

```sh
printf '%s\n' '{"memoryBytes":17179869184}' | hlims machines patch "$MACHINE_ID"
```

REST is equivalent for all resources and is required for DNS/ingress resources at present. Example with *placeholder* IDs (first GET/list and reuse matching records):

```sh
curl -fsS "$HLIMS_API_URL/dns-records" | jq '.items[] | {publicId,name,kind,addressPublicId}'
curl -fsS -X POST "$HLIMS_API_URL/dns-records" -H 'Content-Type: application/json' \
  -d '{"zonePublicId":"ZONE_PUBLIC_ID","name":"app","kind":"A","addressPublicId":"ADDRESS_PUBLIC_ID"}'
curl -fsS "$HLIMS_API_URL/topology" | jq '.providers[]? | {name,areas}'
hlims resolve instance SERVICE_SLUG INSTANCE_SLUG
hlims resolve instance SERVICE_SLUG INSTANCE_SLUG --host MACHINE_SLUG
hlims open --print SERVICE_SLUG/INSTANCE_SLUG
hlims open --print 'SERVICE_SLUG/INSTANCE_SLUG?host=managed'
```

Check the created resource, Machines/Services/Inventory/Orders console, resolver URL, and observed listener/DNS against the source evidence. Rerun list comparisons before each new write so repeated imports are idempotent. Keep receipts, real IPs/hostnames, order IDs, serials, and credentials out of tracked files; staging DB and `ORDERS.md` are private/ignored. Do not use the organization Bitwarden vault for personal inventory. Report discrepancies and questions instead of silently guessing.
