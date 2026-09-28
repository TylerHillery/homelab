# Shared Machine Configuration Proposal

Status: proposal for discussion after HLIMS staging.

## Goal

Configure Tailscale on OPNsense and Linux machines through a consistent operator
workflow without treating their installation methods as interchangeable.

## Layout

```text
infrastructure/ansible/
  ansible.cfg
  requirements.yml
  inventories/
    hosts.example.yml
    hosts.local.yml             # ignored real inventory
  group_vars/
    opnsense.yml
    ubuntu.yml
  host_vars/                    # ignored real host settings
  roles/
    tailscale_opnsense/         # OPNsense plugin and API
    tailscale_linux/            # Linux package, daemon, CLI
    opnsense_firewall/          # firewall, DHCP, DNS and GUI
  playbooks/
    opnsense.yml
    ubuntu.yml
```

`services/` remains application source; infrastructure describes deployed
machines. Inventory selects OS-specific roles. Keep existing `opnsense/` working
until the new OPNsense playbook reproduces its check/apply behavior.

## Shared Inputs

Define a documented Tailscale policy per host: node name, tag, accept-DNS,
Tailscale SSH, optional exit-node advertisement, optional subnet routes, and
optional Serve endpoints. OPNsense's exit-node and trusted GUI Serve settings
are explicit opt-ins, never Linux defaults. A new device needs bootstrap access
over LAN/SSH or its local console before Tailscale can become its normal
management path.

## Implementations

- OPNsense: retain `os-tailscale`, its API calls, SSH/HTTPS checks, auth-key
  clearing, and the existing firewall behavior. Do not run Linux installation
  tasks on the firewall.
- Ubuntu: install and enable the Tailscale package and daemon; inspect current
  status; enroll only if necessary; converge node flags; verify membership and
  optional Serve configuration. Re-running with no drift must not re-enroll or
  restart a healthy node.
- Inject single-use, correctly tagged auth keys through fnox/Bitwarden only at
  enrollment. Never store them in inventory, Ansible output, or service files.

## Migration And Verification

1. Create shared inventory and move the current OPNsense tasks without changing
   firewall policy or enrollment behavior.
2. Verify syntax and live OPNsense check mode with zero changes; apply and
   confirm a second run is idempotent.
3. Add the Ubuntu role and validate installation, fresh enrollment, no-op
   repeats, and key-free operation on one non-critical host.
4. Move per-project mise tasks and setup documentation to the shared layout,
   then remove the old project only after parity is verified.

HLIMS records the resulting machine addresses and management endpoints; Ansible
remains the configuration authority. No tailnet policy, live leases, or
credentials are imported into HLIMS by this proposal.
