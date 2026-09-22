# Home Network

Status: recommended incremental design; exact hardware remains to be selected.

## Goal

Create an isolated, intentionally experimental homelab network without placing
lab routing, DNS, DHCP, switching, virtualization, or compute in the critical
path for household Internet and Wi-Fi.

The required failure behavior is:

```text
Lab fails -> lab unavailable -> household network remains available
```

## Recommended Topology

Keep the TP-Link Deco in router mode and place a dedicated lab router behind
it:

```text
Internet
  |
TP-Link Deco
  +-- household Wi-Fi and wired devices
  |
  +-- lab router WAN
        |
        +-- dumb switch
              |
              +-- homelab Machines
```

The existing dumb switch may be reused behind the lab router for one flat lab
subnet. Lab Machines must not remain connected to a switch on the Deco side of
the lab router or they will not be isolated by it.

Example addresses are illustrative until the existing network is verified:

```text
Household LAN:  192.168.68.0/24
Lab router WAN: DHCP reservation on the household LAN
Lab LAN:        10.20.0.0/24
```

Use a Deco DHCP reservation for the lab router WAN instead of selecting an
address that might overlap the Deco's dynamic pool. The lab router owns DHCP
and DNS only for downstream lab networks.

## Boundary Policy

Double NAT is acceptable here because failure isolation matters more than
inbound convenience. NAT alone is not an isolation policy: a downstream lab
host can normally initiate traffic to the upstream household LAN unless the lab
router blocks it.

Apply this policy intent for both IPv4 and IPv6:

| Flow | Policy |
|---|---|
| Lab to lab | Allow initially; use host firewalls and segment later as needed |
| Lab to Internet | Allow statefully |
| Lab to household prefixes | Deny before the general Internet allowance |
| Household to lab | Deny new connections |
| Household to lab-router management | Deny |
| Internet to lab | Deny unsolicited traffic |
| Trusted developer to named lab service | Allow through Tailscale policy |
| Public publisher to backend | Allow only the named service and port |

Disable UPnP, NAT-PMP, PCP, exposed-host, and broad port-forwarding features on
the lab router. Do not alter household UPnP solely for the lab. If the lab can
saturate the Internet connection, cap or prioritize its aggregate WAN traffic.

IPv4 double NAT does not provide IPv6 isolation. Initially disable IPv6
forwarding and prefix delegation into the lab without changing household IPv6.
Enable lab IPv6 only after a distinct prefix and equivalent stateful firewall
policy have been tested; do not bridge the household IPv6 segment downstream.

## Trusted Access

Install Tailscale directly on capable lab Machines and trusted development
devices. This gives each node an identity, keeps household devices outside the
access path, and avoids depending on Deco static routes or mutable laptop IPs.

Use a Tailscale subnet router only for devices that cannot run Tailscale, such
as switch management, router management, PXE environments, or BMCs. Advertise
only required lab prefixes, never the household LAN, and restrict access with
tailnet policy. Preserve direct-node access so losing the subnet router does
not remove all administration paths.

Keep private services on direct Tailscale or Tailscale Serve endpoints. Prefer
an outbound tunnel or a VPS reverse proxy over residential port forwarding for
public services. A public publisher should have tailnet access only to the
specific backend it exposes.

## Router Platform

| Platform | Fit |
|---|---|
| OPNsense | Recommended on generic x86-64: open source, mature firewall/VLAN GUI, official API, configuration history, backups, and ZFS snapshots |
| OpenWrt | Recommended on supported embedded hardware: open source, low resource use, UCI configuration, nftables, and strong reproducibility |
| MikroTik RouterOS | Capable and inexpensive with excellent networking depth, but proprietary and less aligned with the open-source preference |
| pfSense CE | Mature open-source alternative, but less attractive than OPNsense when official API automation matters |
| Firewalla | Polished and convenient, but expensive and less portable or reproducible |

Choose OPNsense if using a fanless x86 appliance with at least two NICs. Choose
OpenWrt if an explicitly supported embedded router provides sufficient routing
and VPN throughput. Do not run the lab router as a VM on an experimental
Proxmox host: that would make the lab boundary depend on the system being
experimented upon.

## Oxide Development

Oxide development has two materially different network modes. Keep them
separate when planning capacity and isolation:

| Mode | Host | Network requirement | Limitation |
|---|---|---|---|
| Simulated Omicron | Linux, macOS, or illumos | Localhost and ordinary outbound access | Does not run real Instances |
| `helios-engvm` | x86-64 Linux with KVM and libvirt | Libvirt `default` NAT network, commonly `192.168.122.0/24` | The Helios guest remains behind host NAT |
| Non-simulated Omicron | Bare-metal Helios/illumos | Wired attachment to the lab LAN, a gateway, and a reserved contiguous IPv4 range | Required for real Instances using SoftNPU |

The simulated modes do not justify changing the physical network. They can run
on an existing development Machine without an external address pool. The
`helios-engvm` tooling currently targets x86-64 Ubuntu and attaches the guest to
libvirt's hardcoded `default` network; treat that network as local development
infrastructure rather than another physical homelab Network.

For a non-simulated single-sled deployment, connect the Helios Machine by a
wired interface to the flat lab LAN. Reserve a contiguous block of unused
addresses in that LAN and exclude the complete block from DHCP. Omicron divides
this capacity among control-plane services, SoftNPU infrastructure, and
Instance external addresses. Do not assign addresses from the block to ordinary
hosts, even if they appear unused.

An illustrative allocation is:

```text
Lab LAN                 10.20.0.0/24
Lab router              10.20.0.1
Static infrastructure   10.20.0.2-10.20.0.31
DHCP clients            10.20.0.32-10.20.0.127
Reserved for Omicron    10.20.0.128-10.20.0.223
Unallocated growth      10.20.0.224-10.20.0.254
```

This is an allocation pattern, not an Omicron minimum-size requirement. Verify
the current Omicron setup requirements and expected Instance count before
choosing the actual boundaries.

The current non-Gimlet setup uses SoftNPU and proxy ARP to make its external
addresses reachable on the attached LAN. A dumb switch and one flat subnet are
therefore sufficient for the first deployment; a managed switch, dedicated
VLAN, or BGP speaker is not initially required. Introduce a separate experiment
VLAN later if running untrusted Instances or testing routed/BGP topologies.

Installing Tailscale on the Helios host reaches the host but does not by itself
place SoftNPU service and Instance addresses on the tailnet. Reach those
addresses through a narrowly scoped Tailscale subnet route or an SSH tunnel.
Advertise only the reserved Omicron range when practical, and apply tailnet
policy independently of the lab-router firewall.

Helios development images may permit root login with an empty password. Treat
the physical deployment as hostile until credentials and host policy are
hardened: deny access from the household LAN and Internet, create no port
forwards, and permit administration only from explicit trusted paths.

## VLAN Growth

A managed switch is not required for the initial flat lab subnet. Add one when
multiple downstream ports must be assigned to different VLANs over a shared
router trunk. Add a separate VLAN-aware AP only when wireless clients need to
join lab VLANs; do not assume Deco can map arbitrary SSIDs to lab VLANs.

Introduce segmentation only in response to concrete trust boundaries:

```text
management    router, switch, and infrastructure administration
servers       stable services and storage
experiments   Kubernetes, test VMs, and untrusted workloads
provisioning  PXE, installers, and enrollment
```

Default-deny inter-VLAN forwarding and add narrow service exceptions. Keep all
VLAN trunks and failure modes downstream of the lab router.

## HLIMS Impact

The current model can represent the first phase with separate household LAN,
lab LAN, and Tailnet Network records. A dedicated router is a router Product
and Asset with one Address on each LAN. A general-purpose x86 router host can
instead remain a Machine backed by a system Asset; network-equipment Assets do
not back Machines. A physical NIC is a `network_adapter` Product and Asset,
normally contained by the system Asset with a descriptive slot such as
`PCIe x16`. Its grouped port profiles describe connector counts and speeds, but
the adapter Asset cannot own management Addresses or back a Machine. This
inventories equipment and addresses but does not model individual ports,
cabling, interfaces, routing, NAT, gateways, VLAN membership, live link state,
or policy.

The current HP Pro 3500 example follows that placement model: its H!Fiber Intel
I350-compatible dual-port NIC remains installed in `PCIe x16`. The $35 Facebook
Marketplace transaction in Marathon City is a Purchase associated only with the
PC Asset. The NIC was purchased separately and must use a separate Purchase;
its unknown price and date must not be inferred from the PC transaction.

Several current abstractions become ambiguous as the network grows:

- `Network.kind` mixes connectivity scopes (`lan`, `tailnet`), administrative
  containers (`cloud_vpc`), and the public Internet.
- One nullable `Network.cidr` cannot represent dual-stack or multiple prefixes.
- Network cannot distinguish assignable space from a range reserved for a
  system such as Omicron.
- Address stores a free-text interface name instead of referencing a stable
  Machine interface.
- Address does not distinguish static, DHCP reservation, DHCP lease, SLAAC,
  provider-assigned, or overlay-assigned values.
- `is_primary` has no defined uniqueness scope.
- `via=lan` cannot select correctly once household, lab, and VLAN Networks all
  have kind `lan`.
- A router Asset is identifiable, but its routed relationships and live links
  are not modeled.
- Instance Endpoint requires the serving Address to belong to the workload
  Machine, which cannot describe a reverse proxy, virtual IP, or NAT exposure.

Evolve the schema only as each network phase requires it.

### Stage 1: Multiple LANs

- Define Network as a logical reachability domain, not physical wiring or a
  complete routing policy.
- Add stable Network selection to resolution instead of relying on kind alone.
- Add the household LAN, flat lab LAN, Tailnet, and lab router to sample data.
- Clarify preferred Address scope and validate addresses against prefixes.

### Stage 2: Interfaces And Prefixes

- Add a child Network Prefix so one Network can have IPv4 and IPv6 prefixes.
- Add an Address Pool beneath a Network Prefix when reserved ranges become an
  operational need. Record its range, purpose, and allocation authority without
  copying dynamic DHCP leases or Omicron's internal allocation state.
- Add Machine-owned Network Interfaces with stable identity, observed name,
  MAC address, kind, and optional parent interface.
- Attach Addresses to interfaces and record assignment method and provenance.
- Add Network Attachments only when tagged VLANs or trunks are introduced.

### Stage 3: Routed Topology

- Add typed Network relationships such as `routed_to`, `contains`, and
  `overlay_on`; do not add an ambiguous `parent_network_id`.
- Allow a routed relationship to identify its router Machine and gateway.
- Add coarse security zones only if HLIMS needs to describe expected access.
- Keep complete routing tables, DHCP leases, nftables rules, and Tailscale ACL
  syntax in their authoritative systems instead of copying them into HLIMS.

HLIMS should inventory topology and access intent, not configure the router or
treat transient reachability probes as authoritative policy.

## Open Questions

- Exact Deco models, hardware revisions, firmware, and current LAN prefix.
- ISP handoff, public IPv4 or CGNAT status, and IPv6 prefix delegation.
- Internet speed and required routed, WireGuard, or Tailscale throughput.
- Exact used router hardware available within the `$100` budget and preference
  for x86 versus embedded hardware.
- Number and link speed of wired lab devices.
- Whether any lab devices need Wi-Fi.
- Which non-Tailscale devices require subnet-routed access.
- Which services, if any, require arbitrary public TCP or UDP exposure.
- Which Machine will run bare-metal Helios and how many Omicron external
  addresses its expected workload requires.
- Whether BGP experimentation is a near-term goal or should wait until the flat
  proxy-ARP deployment is understood.
