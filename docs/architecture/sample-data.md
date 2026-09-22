# HLIMS Sample Data

This document shows how the current database schema could represent known
hardware, local machines, cloud machines, networks, and service endpoints. It is a review
fixture, not seed data. Serial numbers, UUIDs, public IDs, provider IDs, and
public addresses are fictionalized so sensitive identifiers are not committed.

Foreign-key columns are displayed using readable record names where that makes
the examples easier to follow. The database stores the referenced record's
internal UUIDv7.

All `created_at` and `updated_at` values are Unix seconds. The examples use
`1789257600` only to keep the tables compact.

## Machine Providers

`machine_providers` identifies the organization or environment responsible for
supplying and operating a Machine. Every Area and Machine has a provider.

| id | public_id | name | slug | created_at | updated_at |
|---|---|---|---|---:|---:|
| `019d0000-0000-7000-8000-000000000000` | `homeprovider` | Homelab | `homelab` | 1789257600 | 1789257600 |
| `019d0000-0000-7001-8000-000000000001` | `do7m4k2p9x3q` | DigitalOcean | `digitalocean` | 1789257600 | 1789257600 |

## Areas

`areas` is the only location level for now. Every Area references the
environment responsible for it; no Area kind is needed yet.

| id | public_id | machine_provider_id | name | slug | provider_code | notes | created_at | updated_at |
|---|---|---|---|---|---|---|---:|---:|
| `019d0000-0000-7010-8000-000000000010` | `home4q8m2v7k` | Homelab | Primary Home | `primary-home` | null | Default location for owned hardware | 1789257600 | 1789257600 |
| `019d0000-0000-7011-8000-000000000011` | `sfo3k6p2m9qx` | DigitalOcean | San Francisco 3 | `san-francisco-3` | `sfo3` | DigitalOcean region | 1789257600 | 1789257600 |

## Manufacturers

`manufacturers` provides canonical identities for Product manufacturers.

| id | public_id | name | slug | created_at | updated_at |
|---|---|---|---|---:|---:|
| `019d0000-0000-7012-8000-000000000012` | `hpmanuf8k2q5` | Hewlett-Packard | `hp` | 1789257600 | 1789257600 |
| `019d0000-0000-7013-8000-000000000013` | `fwmanuf7p3kx` | Framework | `framework` | 1789257600 | 1789257600 |
| `019d0000-0000-7014-8000-000000000014` | `intelman4n8q` | Intel | `intel` | 1789257600 | 1789257600 |
| `019d0000-0000-7015-8000-000000000015` | `amdmanuf6p2k` | AMD | `amd` | 1789257600 | 1789257600 |
| `019d0000-0000-7016-8000-000000000016` | `hynixman3q8p` | Hynix/Hyundai | `hynix` | 1789257600 | 1789257600 |
| `019d0000-0000-7017-8000-000000000017` | `micronmn5k2q` | Micron Technology | `micron` | 1789257600 | 1789257600 |
| `019d0000-0000-7018-8000-000000000018` | `wdmanuf8p3kx` | Western Digital | `western-digital` | 1789257600 | 1789257600 |
| `019d0000-0000-7019-8000-000000000019` | `samsung7k2qx` | Samsung | `samsung` | 1789257600 | 1789257600 |
| `019d0000-0000-701a-8000-00000000001a` | `msmanuf4n8p2` | Microsoft | `microsoft` | 1789257600 | 1789257600 |
| `019d0000-0000-701b-8000-00000000001b` | `deskpi7m2q9k` | DeskPi | `deskpi` | 1789257600 | 1789257600 |
| `019d0000-0000-701c-8000-00000000001c` | `tplink8n4q2x` | TP-Link | `tp-link` | 1789257600 | 1789257600 |
| `019d0000-0000-701d-8000-00000000001d` | `hfiber7m3q2k` | H!Fiber | `h-fiber` | 1789257600 | 1789257600 |

## Products

`products` describes models that can have many physical Asset instances.

| id | public_id | kind | manufacturer_id | name | part_number | notes | created_at | updated_at |
|---|---|---|---|---|---|---|---:|---:|
| `019d0000-0000-7020-8000-000000000020` | `hp6g1d4m8k2q` | `system` | Hewlett-Packard | HP ProDesk 600 G1 DM | `K7M40US#ABA` | Shared by Badger, Brewer, and Buck | 1789257600 | 1789257600 |
| `019d0000-0000-7021-8000-000000000021` | `fw13a7m3p9kx` | `system` | Framework | Laptop 13, AMD Ryzen AI 300 Series | null | Framework chassis | 1789257600 | 1789257600 |
| `019d0000-0000-7022-8000-000000000022` | `i54590t8c4nz` | `processor` | Intel | Core i5-4590T | null | 2.00 GHz base clock | 1789257600 | 1789257600 |
| `019d0000-0000-7023-8000-000000000023` | `ry7350m4p8qx` | `processor` | AMD | Ryzen AI 7 350 with Radeon 860M | null | Framework processor | 1789257600 | 1789257600 |
| `019d0000-0000-7024-8000-000000000024` | `hyn4ga7m2k9q` | `memory` | Hynix/Hyundai | 4 GiB DDR3-1600 SODIMM | `HMT451S6AFR8A-PB` | Badger DIMM3 model | 1789257600 | 1789257600 |
| `019d0000-0000-7025-8000-000000000025` | `hyn4gb3n8p2x` | `memory` | Hynix/Hyundai | 4 GiB DDR3-1600 SODIMM | `HMT451S6BFR8A-PB` | Badger DIMM1 model | 1789257600 | 1789257600 |
| `019d0000-0000-7026-8000-000000000026` | `mic32g5k8p2q` | `memory` | Micron Technology | 32 GiB DDR5-5600 SODIMM | `CT32G56C46S5.M16D1` | Framework memory | 1789257600 | 1789257600 |
| `019d0000-0000-7027-8000-000000000027` | `wd1tb8m3k6pq` | `drive` | Western Digital | WD_BLACK SN850X 1000GB | null | Framework drive | 1789257600 | 1789257600 |
| `019d0000-0000-7028-8000-000000000028` | `sam250g7n2kx` | `drive` | Samsung | SSD 870, exact model pending | null | Buck drive | 1789257600 | 1789257600 |
| `019d0000-0000-7029-8000-000000000029` | `sb3model7k2q` | `system` | Microsoft | Surface Book 3 | null | Detailed hardware pending | 1789257600 | 1789257600 |
| `019d0000-0000-702a-8000-00000000002a` | `rackmate4u2k` | `rack` | DeskPi | RackMate T0 | `RackMate T0` | 4U 10-inch desktop rack | 1789257600 | 1789257600 |
| `019d0000-0000-702b-8000-00000000002b` | `sg605sw8m2qx` | `switch` | TP-Link | TL-SG605 | `TL-SG605` | Five-port unmanaged switch | 1789257600 | 1789257600 |
| `019d0000-0000-702c-8000-00000000002c` | `decobe65p3kx` | `router` | TP-Link | Deco BE65 Pro | `BE65 Pro` | Tri-band mesh router | 1789257600 | 1789257600 |
| `019d0000-0000-702d-8000-00000000002d` | `hppro3500k2q` | `system` | Hewlett-Packard | HP Pro 3500 | null | Used Marketplace system; CPU and RAM pending discovery | 1789257600 | 1789257600 |
| `019d0000-0000-702e-8000-00000000002e` | `i350nic7m2qx` | `network_adapter` | H!Fiber | Dual-port Intel I350-compatible NIC | null | Purchased separately from the HP Pro 3500 | 1789257600 | 1789257600 |

## Processor Specifications

`processor_specs` extends Products whose kind is `processor`.

| product_id | core_count | thread_count | base_clock_mhz | virtualization |
|---|---:|---:|---:|---|
| Intel Core i5-4590T | 4 | 4 | 2000 | VT-x |
| AMD Ryzen AI 7 350 | 8 | 16 | 2000 | AMD-V |

## Memory Specifications

`memory_specs` extends Products whose kind is `memory`. Installed memory for a
system is the sum of the specifications for its contained memory Assets.

| product_id | capacity_bytes | memory_type | form_factor | speed_mts |
|---|---:|---|---|---:|
| Hynix `HMT451S6AFR8A-PB` | 4294967296 | DDR3 | SODIMM | 1600 |
| Hynix `HMT451S6BFR8A-PB` | 4294967296 | DDR3 | SODIMM | 1600 |
| Micron `CT32G56C46S5.M16D1` | 34359738368 | DDR5 | SODIMM | 5600 |

## Drive Specifications

`media_kind` describes the storage technology. `interface_kind` describes how
it connects or is presented. An NVMe SSD is therefore `ssd` plus `nvme`.

| product_id | capacity_bytes | media_kind | interface_kind |
|---|---:|---|---|
| WD_BLACK SN850X 1000GB | 1000204886016 | `ssd` | `nvme` |
| Samsung SSD 870 | 250000000000 | `ssd` | `sata` |

## Rack Specifications

`rack_specs` extends Products whose kind is `rack`. `mounting_standard`
describes the physical standard rather than a rack location.

| product_id | rack_units | mounting_standard |
|---|---:|---|
| DeskPi RackMate T0 | 4 | 10-inch |

## Product Port Profiles

`product_port_profiles` is an ordered inventory of grouped capabilities for
router, switch, access-point, and network-adapter Products. A row describes a
count of equivalent ports, not individual ports, cables, or current connections.
`name` is optional; `speed_mbps` records the speed of each port in the group.

| product_id | position | name | port_count | connector | speed_mbps |
|---|---:|---|---:|---|---:|
| TP-Link TL-SG605 | 0 | null | 5 | RJ45 | 1000 |
| TP-Link Deco BE65 Pro | 0 | 5 GbE | 2 | RJ45 | 5000 |
| TP-Link Deco BE65 Pro | 1 | 2.5 GbE | 1 | RJ45 | 2500 |
| H!Fiber Intel I350-compatible NIC | 0 | null | 2 | RJ45 | 1000 |

## Product Links

`product_links` stores ordered URLs. Link kinds classify the destination; the
link itself is not an uploaded document.

| product_id | position | kind | label | url |
|---|---:|---|---|---|
| DeskPi RackMate T0 | 0 | `manufacturer` | Product page | `https://products.example.invalid/deskpi/rackmate-t0` |
| DeskPi RackMate T0 | 1 | `retailer` | Example retailer | `https://shop.example.invalid/rackmate-t0` |
| TP-Link TL-SG605 | 0 | `manual` | Installation guide | `https://docs.example.invalid/tp-link/tl-sg605/manual` |
| TP-Link TL-SG605 | 1 | `datasheet` | Specifications | `https://docs.example.invalid/tp-link/tl-sg605/datasheet` |
| TP-Link Deco BE65 Pro | 0 | `support` | Support | `https://support.example.invalid/tp-link/deco-be65-pro` |

## Assets

`assets` represents individual physical items. `parent_asset_id` records current
containment, and `parent_slot` may describe placement within that parent.

| id | public_id | product_id | parent_asset_id | parent_slot | area_id | name | serial_number | system_uuid | notes |
|---|---|---|---|---|---|---|---|---|---|
| `019d0000-0000-7030-8000-000000000030` | `badger8m2k5q` | HP ProDesk | null | null | Primary Home | Badger chassis | `SN-BADGER-EXAMPLE` | `019d1000-0000-7000-8000-000000000001` | null |
| `019d0000-0000-7031-8000-000000000031` | `badcpu4n8p2x` | Intel i5-4590T | Badger chassis | null | null | null | null | null | null |
| `019d0000-0000-7032-8000-000000000032` | `baddim3k7q2m` | Hynix `AFR8A` | Badger chassis | null | null | null | `RAM-BADGER-1` | null | null |
| `019d0000-0000-7033-8000-000000000033` | `baddim1p8x4n` | Hynix `BFR8A` | Badger chassis | null | null | null | `RAM-BADGER-2` | null | null |
| `019d0000-0000-7034-8000-000000000034` | `buck7m2q9k4x` | HP ProDesk | null | null | Primary Home | Buck chassis | `SN-BUCK-EXAMPLE` | `019d1000-0000-7000-8000-000000000002` | null |
| `019d0000-0000-7035-8000-000000000035` | `brewer3n8p5q` | HP ProDesk | null | null | Primary Home | Brewer chassis | null | null | Inventory details pending |
| `019d0000-0000-7036-8000-000000000036` | `frame7k2m9qx` | Framework Laptop | null | null | Primary Home | Framework 13 chassis | `SN-FRAMEWORK-EXAMPLE` | `019d1000-0000-7000-8000-000000000003` | null |
| `019d0000-0000-7037-8000-000000000037` | `fwcpu8p3n6k2` | AMD Ryzen AI 7 350 | Framework chassis | null | null | null | null | null | null |
| `019d0000-0000-7038-8000-000000000038` | `fwdim4q8m2nx` | Micron 32 GiB | Framework chassis | null | null | null | `RAM-FRAMEWORK-1` | null | null |
| `019d0000-0000-7039-8000-000000000039` | `fwnvme7p2k9m` | WD_BLACK SN850X | Framework chassis | null | null | null | `DRIVE-FRAMEWORK-1` | null | null |
| `019d0000-0000-703a-8000-00000000003a` | `sb3asset8m2q` | Microsoft Surface Book 3 | null | null | Primary Home | Surface Book 3 chassis | null | null | Hardware details pending |
| `019d0000-0000-703b-8000-00000000003b` | `rackasset4u2` | DeskPi RackMate T0 | null | null | Primary Home | Lab rack | `RACK-T0-EXAMPLE` | null | null |
| `019d0000-0000-703c-8000-00000000003c` | `switch5p2kqx` | TP-Link TL-SG605 | Lab rack | null | null | Lab switch | `SG605-EXAMPLE-01` | null | Mounted in the rack |
| `019d0000-0000-703d-8000-00000000003d` | `decomain7k2q` | TP-Link Deco BE65 Pro | null | null | Primary Home | Main Deco | `BE65P-EXAMPLE-01` | null | Primary router |
| `019d0000-0000-703e-8000-00000000003e` | `decoup1m8k2q` | TP-Link Deco BE65 Pro | null | null | Primary Home | Upstairs Deco | `BE65P-EXAMPLE-02` | null | Mesh unit |
| `019d0000-0000-703f-8000-00000000003f` | `decoup2n7p3x` | TP-Link Deco BE65 Pro | null | null | Primary Home | Office Deco | `BE65P-EXAMPLE-03` | null | Mesh unit |
| `019d0000-0000-704d-8000-00000000004d` | `pro3500m8k2q` | HP Pro 3500 | null | null | Primary Home | HP Pro 3500 Marketplace system | null | null | CPU and RAM pending discovery |
| `019d0000-0000-704e-8000-00000000004e` | `i350asset7k2` | H!Fiber Intel I350-compatible NIC | HP Pro 3500 Marketplace system | PCIe x16 | null | Dual-port NIC | null | null | Purchased separately; price and date unknown |

Assets with `parent_asset_id = null` are top-level equipment or uninstalled
spares. Installed components normally leave `area_id` null because their Area
is derived through the parent system or rack Asset. Only system and rack Assets
can contain children. The HP Pro 3500's unknown CPU and RAM are omitted pending
discovery rather than represented by placeholder component Assets.

The API derives `purchasePublicId` from the Purchase's required primary Asset
and any additional `purchase_assets` associations; it is null when an Asset has
no recorded Purchase. Purchase data is not embedded in `assets`.

## Purchases

`purchases` represents transactions independently of Assets. Every Purchase
has a total and currency; date, source, and notes are optional. The known $35 HP
Pro 3500 transaction has its own Purchase.

| id | public_id | primary_asset_id | total_price_cents | currency | purchased_on | source | notes | created_at | updated_at |
|---|---|---|---:|---|---|---|---|---:|---:|
| `019d0000-0000-7080-8000-000000000080` | `pro3500buy12` | HP Pro 3500 Marketplace system | 3500 | USD | null | Facebook Marketplace, Marathon City | PC only; the installed NIC was purchased separately | 1789257600 | 1789257600 |

Each Purchase has a required `primary_asset_id`. `purchase_assets` associates
any additional physical Assets in the same transaction. Cross-table constraints
enforce that an Asset belongs to at most one Purchase. This single-Asset fixture
therefore has no `purchase_assets` rows.

The H!Fiber NIC remains installed in the HP Pro 3500's `PCIe x16` slot, but it
was not part of the PC transaction. It requires a separate Purchase. Its price
and purchase date are unknown, and this fixture does not invent a total or
misrepresent zero as unknown; because `total_price_cents` and `currency` are
required, that incomplete transaction cannot yet be persisted as a valid row.

A mini-PC bundle is represented differently: one Purchase row identifies one
mini-PC as its primary Asset and associates the remaining mini-PC Assets through
`purchase_assets`. Its total is counted once rather than copied to every
mini-PC.

## Purchase Links

`purchase_links` stores ordered external URLs of kind `receipt`, `listing`, or
`other` for a Purchase.

| purchase_id | position | kind | label | url |
|---|---:|---|---|---|
| HP Pro 3500 Marketplace purchase | 0 | `listing` | Marketplace listing | `https://marketplace.example.invalid/listings/HP-PRO-3500-EXAMPLE` |

`GET /api/v1/purchase-summary` groups Purchases by currency, counts each
Purchase total once, and separately reports associated Asset counts. The
complete Purchase in this fixture produces:

| currency | total_price_cents | purchase_count | asset_count |
|---|---:|---:|---:|
| USD | 3500 | 1 | 1 |

## Asset Links

`asset_links` stores ordered URLs associated with one physical Asset. These
links remain external URLs; HLIMS does not upload files.

| asset_id | position | kind | label | url |
|---|---:|---|---|---|
| Main Deco | 0 | `warranty` | Warranty registration | `https://warranty.example.invalid/tp-link/BE65P-EXAMPLE-01` |
| Main Deco | 1 | `management` | Local management | `https://192.168.68.1/` |
| Lab switch | 0 | `other` | Inventory note | `https://inventory.example.invalid/notes/SG605-EXAMPLE-01` |

## Machines

The following two tables are two views of the same `machines` rows, split so the
wide database table remains readable.

### Identity and Relationships

| id | public_id | asset_id | parent_machine_id | machine_provider_id | area_id | name | slug | kind | virtualization_platform |
|---|---|---|---|---|---|---|---|---|---|
| `019d0000-0000-7040-8000-000000000040` | `badmach8k2q5n` | Badger chassis | null | Homelab | Primary Home | Badger | `badger` | `bare_metal` | null |
| `019d0000-0000-7041-8000-000000000041` | `buckmach7p3kx` | Buck chassis | null | Homelab | Primary Home | Buck | `buck` | `bare_metal` | null |
| `019d0000-0000-7042-8000-000000000042` | `brewmach4n8q2` | Brewer chassis | null | Homelab | Primary Home | Brewer | `brewer` | `bare_metal` | null |
| `019d0000-0000-7043-8000-000000000043` | `fw13mach8m2q` | Framework chassis | null | Homelab | Primary Home | Framework Windows | `framework13` | `bare_metal` | null |
| `019d0000-0000-7044-8000-000000000044` | `fwsl7k3p9m2x` | null | Framework Windows | Homelab | Primary Home | Framework WSL | `framework13-wsl` | `virtual_machine` | `wsl2` |
| `019d0000-0000-7045-8000-000000000045` | `pyprod6n2k8qx` | null | null | DigitalOcean | San Francisco 3 | pypacktrends-prod | `pypacktrends-prod` | `virtual_machine` | null |
| `019d0000-0000-7046-8000-000000000046` | `sb3mach7k2qx` | Surface Book 3 chassis | null | Homelab | Primary Home | Surface Book 3 | `sb3` | `bare_metal` | null |
| `019d0000-0000-704c-8000-00000000004c` | `north5m2q8kx` | null | null | DigitalOcean | San Francisco 3 | Northstar | `northstar` | `virtual_machine` | null |

Database triggers reject both direct self-parenting and longer Machine cycles
such as A hosting B while B hosts A.
Every Machine records its Area directly, including bare-metal and locally
hosted virtual Machines.

### Operating System, Capacity, and Cost

| machine | hostname | os_machine_id | operating_system | operating_system_version | kernel | architecture | cpu_count | cpu_allocation | cpu_vendor | memory_bytes | storage_bytes | storage_media_kind | storage_interface_kind | estimated_monthly_cost_cents | cost_currency | notes |
|---|---|---|---|---|---|---|---:|---|---|---:|---:|---|---|---:|---|---|
| Badger | `badger` | null | Ubuntu | 24.04.4 LTS | `6.8.0-139-generic` | `x86_64` | 4 | `dedicated` | Intel | 8589934592 | 250000000000 | `ssd` | `sata` | null | null | null |
| Buck | `buck` | `OS-BUCK-EXAMPLE` | Ubuntu | 24.04.4 LTS | `6.8.0-139-generic` | `x86_64` | 4 | `dedicated` | Intel | 8589934592 | 250000000000 | `ssd` | `sata` | null | null | null |
| Brewer | `brewer` | null | Linux | null | null | `x86_64` | 4 | `dedicated` | Intel | 8589934592 | 250000000000 | `ssd` | `sata` | null | null | Hardware details not verified |
| Framework Windows | `framework13` | null | Windows | null | null | `x86_64` | 16 | `dedicated` | AMD | 34359738368 | 1000204886016 | `ssd` | `nvme` | null | null | null |
| Framework WSL | `framework13` | `OS-WSL-EXAMPLE` | Ubuntu | 26.04 LTS | `6.18.33.2-microsoft-standard-WSL2` | `x86_64` | 16 | `shared` | AMD | null | 1099511627776 | `ssd` | `virtual` | null | null | Populate fixed memory from `.wslconfig` |
| pypacktrends-prod | `pypacktrends-prod` | null | Linux | null | null | `x86_64` | 1 | `shared` | null | 2147483648 | 50000000000 | `ssd` | `virtual` | 1200 | USD | Always-on monthly estimate |
| Surface Book 3 | `sb3` | null | Linux | null | null | null | null | `dedicated` | null | null | null | null | null | null | null | Hardware details pending |
| Northstar | `northstar` | null | Ubuntu | 24.04 LTS | null | `x86_64` | 2 | `shared` | null | 4294967296 | 80000000000 | `ssd` | `virtual` | 799 | USD | Cloud demonstration machine |

An NVMe SSD remains `storage_media_kind = 'ssd'`; NVMe belongs in
`storage_interface_kind`. Cloud storage may use `virtual` when the guest cannot
reliably identify the provider's underlying interface.

`os_machine_id` is the operating-system installation identifier reported by
Linux `/etc/machine-id` or an equivalent platform facility. It is not an IP
address. IP addresses are separate Address records because each Machine can
have public, private, LAN, and Tailscale addresses.

`dedicated` means the Machine owns the physical CPU capacity represented by the
record or the provider promises dedicated processor time. `shared` means a
hypervisor schedules the virtual CPU alongside its host or other tenants. A VM
is not inherently shared; CPU pinning or a dedicated provider plan can make its
allocation dedicated.

## Machine Users

Machine Users are login identities, not credentials. One preferred account per
Machine gives clients a deterministic default for generated SSH commands.

| id | public_id | machine_id | username | is_preferred | notes | created_at | updated_at |
|---|---|---|---|---:|---|---:|---:|
| `019d0000-0000-7047-8000-000000000047` | `baduser7m2kq` | Badger | `tyler` | 1 | Primary administrative account | 1789257600 | 1789257600 |
| `019d0000-0000-7048-8000-000000000048` | `buckusr8p3nx` | Buck | `tyler` | 1 | Primary administrative account | 1789257600 | 1789257600 |
| `019d0000-0000-7049-8000-000000000049` | `brewusr4n8q2` | Brewer | `tyler` | 1 | Primary administrative account | 1789257600 | 1789257600 |

## Networks

`public` means the public Internet or WAN address scope. It does not mean that
all records in the Network are automatically exposed by HLIMS. If `internet`
would be clearer than `public`, the enum can be renamed before API work begins.

| id | public_id | area_id | name | slug | kind | cidr | notes | created_at | updated_at |
|---|---|---|---|---|---|---|---|---:|---:|
| `019d0000-0000-7050-8000-000000000050` | `homelan7k2p9m` | Primary Home | Home LAN | `home-lan` | `lan` | `192.168.68.0/24` | Private home network | 1789257600 | 1789257600 |
| `019d0000-0000-7051-8000-000000000051` | `tailnet4q8m2x` | null | Personal Tailnet | `personal-tailnet` | `tailnet` | `100.64.0.0/10` | Tailscale overlay | 1789257600 | 1789257600 |
| `019d0000-0000-7052-8000-000000000052` | `inet8n3k2q5m` | null | Public Internet | `public-internet` | `public` | null | Publicly routable addresses | 1789257600 | 1789257600 |
| `019d0000-0000-7053-8000-000000000053` | `dovpc6p2m9kx` | San Francisco 3 | DigitalOcean SFO3 VPC | `digitalocean-sfo3-vpc` | `cloud_vpc` | null | Droplet private network | 1789257600 | 1789257600 |

## Addresses

An Address belongs to exactly one Machine, one Area, or one network-equipment
Asset. Asset targets are limited to router, switch, and access-point Products.
The Area option supports the home ISP address when no router Asset is cataloged.

| id | public_id | network_id | machine_id | area_id | asset_id | name | address | dns_name | interface_name | is_primary | created_at | updated_at |
|---|---|---|---|---|---|---|---|---|---|---:|---:|---:|
| `019d0000-0000-7060-8000-000000000060` | `badlan7m2k9qx` | Home LAN | Badger | null | null | LAN | `192.168.68.60` | null | `enp1s0` | 1 | 1789257600 | 1789257600 |
| `019d0000-0000-7061-8000-000000000061` | `badts4n8p2km` | Personal Tailnet | Badger | null | null | Tailscale | `100.64.0.10` | `badger.example.ts.net` | `tailscale0` | 0 | 1789257600 | 1789257600 |
| `019d0000-0000-7062-8000-000000000062` | `buckts6q2m9nx` | Personal Tailnet | Buck | null | null | Tailscale | `100.64.0.11` | `buck.example.ts.net` | `tailscale0` | 1 | 1789257600 | 1789257600 |
| `019d0000-0000-7063-8000-000000000063` | `fwints8k3p2qm` | Personal Tailnet | Framework Windows | null | null | Tailscale | `100.64.0.12` | `framework13.example.ts.net` | null | 1 | 1789257600 | 1789257600 |
| `019d0000-0000-7064-8000-000000000064` | `fwslts5n9k2qx` | Personal Tailnet | Framework WSL | null | null | Tailscale | `100.64.0.13` | `framework13-wsl.example.ts.net` | `tailscale0` | 1 | 1789257600 | 1789257600 |
| `019d0000-0000-7065-8000-000000000065` | `pypts7m2q8kn` | Personal Tailnet | pypacktrends-prod | null | null | Tailscale | `100.64.0.14` | `pypacktrends-prod.example.ts.net` | `tailscale0` | 1 | 1789257600 | 1789257600 |
| `019d0000-0000-7066-8000-000000000066` | `pyppub4k9m2qx` | Public Internet | pypacktrends-prod | null | null | Public IPv4 | `192.0.2.44` | null | `eth0` | 0 | 1789257600 | 1789257600 |
| `019d0000-0000-7067-8000-000000000067` | `homewan8p3n2k` | Public Internet | null | Primary Home | null | Home ISP | `203.0.113.10` | `home.example.net` | null | 1 | 1789257600 | 1789257600 |
| `019d0000-0000-7068-8000-000000000068` | `sb3ts8m2k4qx` | Personal Tailnet | Surface Book 3 | null | null | Tailscale | `100.64.0.15` | `sb3.example.ts.net` | `tailscale0` | 1 | 1789257600 | 1789257600 |
| `019d0000-0000-7069-8000-000000000069` | `pypvpc8m2k4q` | DigitalOcean SFO3 VPC | pypacktrends-prod | null | null | Private IPv4 | `10.124.0.2` | null | `eth0` | 0 | 1789257600 | 1789257600 |
| `019d0000-0000-706a-8000-00000000006a` | `decoweb7n2kx` | Home LAN | null | null | Main Deco | Management | `192.168.68.1` | `deco.example.home` | null | 1 | 1789257600 | 1789257600 |

## Services

A Service represents software independently of any particular deployment.

| id | public_id | name | slug | description | created_at | updated_at |
|---|---|---|---|---|---:|---:|
| `019d0000-0000-7070-8000-000000000070` | `opencode4m8q` | OpenCode | `opencode` | AI coding agent | 1789257600 | 1789257600 |
| `019d0000-0000-7071-8000-000000000071` | `grafana5k2mx` | Grafana | `grafana` | Metrics visualization | 1789257600 | 1789257600 |

## Instances

An Instance places one Service on one Machine. Its port is the backend listener;
client-facing schemes, ports, and paths belong to Instance Endpoints.

| id | public_id | service_id | machine_id | name | slug | port | notes | created_at | updated_at |
|---|---|---|---|---|---|---:|---|---:|---:|
| `019d0000-0000-7072-8000-000000000072` | `opcdprd7m2kx` | OpenCode | Framework WSL | Production | `production` | 4096 | null | 1789257600 | 1789257600 |
| `019d0000-0000-7073-8000-000000000073` | `grafprd8n4qx` | Grafana | Badger | Production | `production` | 3000 | null | 1789257600 | 1789257600 |

## Instance Endpoints

An endpoint references an Address on the Instance's Machine. The preferred
endpoint is used when a canonical route does not specify `via`.

| id | public_id | instance_id | address_id | name | scheme | port | base_path | is_preferred | notes | created_at | updated_at |
|---|---|---|---|---|---|---:|---|---:|---|---:|---:|
| `019d0000-0000-7074-8000-000000000074` | `opcdlan7m2kx` | OpenCode / Production | Framework WSL / Tailscale | Tailnet direct | `http` | 4096 | `/` | 0 | Requires the backend to bind beyond loopback | 1789257600 | 1789257600 |
| `019d0000-0000-7075-8000-000000000075` | `grafsv8n4q2x` | Grafana / Production | Badger / Tailscale | Tailscale Serve | `https` | 443 | `/grafana` | 1 | Proxies to backend port 3000 | 1789257600 | 1789257600 |

## Deferred Tables

Detailed storage allocation and historical Asset installation tables are
intentionally deferred until their workflows are designed. Individual network
ports, cabling, VLAN assignments, routing relationships, and live link state
are also deferred; Product port profiles are inventory only. None are part of
the current migration.
