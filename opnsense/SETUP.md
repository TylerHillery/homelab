# OPNsense Setup Runbook

## 1. Record Site Values

On a computer connected to the existing home network, run:

```sh
ip -4 route
```

Record the directly connected home subnet as `<UPSTREAM_CIDR>` and its default
gateway as `<UPSTREAM_GATEWAY>`. Ask the ISP or inspect the upstream router to
determine `<WAN_MODE>`: DHCP, PPPoE, or static.

Choose these values:

| Value | Requirement |
|---|---|
| `<LAB_CIDR>` | Private subnet that does not overlap `<UPSTREAM_CIDR>` or the tailnet |
| `<FIREWALL_LAN_IP>` | Address inside `<LAB_CIDR>` reserved for OPNsense |
| `<DHCP_START>` | First dynamic address inside `<LAB_CIDR>` |
| `<DHCP_END>` | Last dynamic address inside `<LAB_CIDR>` |
| `<TAILSCALE_NODE>` | Unique Tailscale name for the firewall |

## 2. Install OPNsense

1. Install a compatible OPNsense release.
2. Assign WAN and LAN by matching link state or NIC MAC addresses.
3. Configure WAN using `<WAN_MODE>`.
4. Configure LAN as `<FIREWALL_LAN_IP>` with the `<LAB_CIDR>` prefix length.
5. Connect a client to LAN and open `https://<FIREWALL_LAN_IP>`.

## 3. Create API Credentials

1. Sign in to OPNsense.
2. Open **System > Access > Users**.
3. Create an administrative API user and generate an API key.
4. Store it in the personal Bitwarden `homelab` folder as
   `homelab-opnsense-api`:
   - Username: API key
   - Password: API secret

## 4. Prepare Secrets

1. Create a one-time, pre-authorized Tailscale key with the required device tag.
2. Store it as the password of `tailscale-auth-key` in the personal Bitwarden
   `homelab` folder.
3. From the repository root, run:

```sh
mise install --monorepo
mise exec -- bw login
mise run bw:bootstrap-age
mise run bw:unlock
```

## 5. Bootstrap Tailscale

Use any temporary management host that can reach `<FIREWALL_LAN_IP>`. If
needed, forward its HTTPS port:

```sh
ssh -N -L 8443:<FIREWALL_LAN_IP>:443 <user>@<management-host>
```

Through OPNsense:

1. Install the `os-tailscale` plugin.
2. Enter the one-time key under **VPN > Tailscale > Authentication**.
3. Enable Tailscale under **VPN > Tailscale > Settings**.
4. Enable Tailscale SSH and exit-node advertisement.
5. Disable tailnet DNS and subnet-route acceptance.
6. Apply and wait for the node to report `Running`.
7. Clear the saved authentication key and apply again.

Discover the assigned values:

```sh
tailscale ping <TAILSCALE_NODE>
tailscale status --json | jq -r '.MagicDNSSuffix'
```

Record the ping address as `<TAILSCALE_IP>`. Combine the node name and reported
suffix as `<TAILSCALE_FQDN>`.

Approve the exit node in the Tailscale admin console.

## 6. Configure Ansible

Create the ignored site configuration:

```sh
cp site.example.yml group_vars/all/site.yml
```

Set these values in `group_vars/all/site.yml`:

| Variable | Value |
|---|---|
| `opnsense_api_host` | `<TAILSCALE_FQDN>` |
| `opnsense_lan_address` | `<FIREWALL_LAN_IP>` |
| `opnsense_tailnet_hostname` | `<TAILSCALE_FQDN>` |
| `lab_lan_subnet` | `<LAB_CIDR>` |
| `lab_dhcp_range` | `<DHCP_START>` through `<DHCP_END>` |

Update site-specific firewall destinations if `<UPSTREAM_CIDR>` changes the
required isolation policy.

## 7. Apply Configuration

From `opnsense/`, run:

```sh
mise run syntax
mise run check
mise run apply
```

Ansible configures Tailscale Serve, trusted HTTPS, firewall policy, DHCP, and
DNS. Close the temporary tunnel after the apply succeeds.

## 8. Verify

1. Open `https://<TAILSCALE_FQDN>/` without a certificate warning.
2. Confirm `mise run check` reports zero changes.
3. From a lab client, run `ping -c 3 <UPSTREAM_GATEWAY>` and expect no replies.
4. Run `getent ahostsv4 example.com` and confirm DNS succeeds.
5. Run `curl -I --max-time 10 https://example.com` and confirm HTTPS succeeds.
6. Confirm the firewall appears in `tailscale exit-node list`.

## 9. Back Up

Download an encrypted configuration backup from **System > Configuration >
Backups**. Store its unique password in Bitwarden and keep the backup outside
Git in two reliable locations.
