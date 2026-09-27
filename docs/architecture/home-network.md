# Home Network

Status: current base design; segmentation grows only when required.

## Goal

Keep experimental lab infrastructure out of the household network's critical
path.

```text
Lab failure -> lab unavailable -> household network remains available
```

## Topology

```text
Internet
  |
household router
  +-- household devices
  |
  +-- lab firewall WAN
        |
        +-- lab switch
              |
              +-- lab machines
```

The household router remains responsible for household connectivity. OPNsense
owns lab routing, DHCP, DNS, and policy. Site addresses are configuration, not
architecture; discover them with the [OPNsense setup runbook](../../opnsense/SETUP.md).

## Policy

| Flow | Policy |
|---|---|
| Lab to lab | Allow initially |
| Lab to Internet | Allow statefully |
| Lab to household private networks | Deny before Internet allowance |
| Household or Internet to lab | Deny unsolicited connections |
| Trusted operator to lab management | Allow through tailnet policy |
| Public publisher to backend | Allow only the named service and port |

NAT is not an isolation policy. The firewall must explicitly block upstream
private networks. Apply equivalent policy before enabling IPv6 in the lab.

Disable automatic port mapping and broad port forwards on the lab firewall.
Prefer private ingress or an outbound publisher for services.

## Management

Install Tailscale directly on capable machines. Use subnet routing only for
devices that cannot run it, and never advertise the household network. Keep
tailnet access policy separate from firewall policy.

OPNsense is managed through Ansible over its tailnet endpoint. Tailscale Serve
provides trusted HTTPS. The operational sources of truth are:

- [`opnsense/site.example.yml`](../../opnsense/site.example.yml): site variable
  template
- `opnsense/group_vars/all/site.yml`: ignored site configuration
- [`opnsense/playbooks/site.yml`](../../opnsense/playbooks/site.yml)
- [`opnsense/SETUP.md`](../../opnsense/SETUP.md)

## Growth

Start with one flat lab network and an unmanaged switch. Add VLANs only for a
concrete trust boundary such as management, stable services, experiments, or
provisioning. Use default-deny inter-VLAN policy and narrow exceptions.

HLIMS inventories networks and endpoints; it does not copy firewall rules,
routes, leases, or Tailscale policy. Proposed model additions are tracked in the
[HLIMS Roadmap](hlims-roadmap.md).
