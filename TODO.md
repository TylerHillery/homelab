# TODO

## Planned Projects

- [ ] Build the [Ubuntu development-host Ansible pilot](docs/architecture/provisioning-spec.md)
  on an isolated WSL2 distro, then validate on a mini PC.
- [ ] Migrate OPNsense into the shared Ansible project after Ubuntu verification
  and firewall check/apply parity.
- [ ] Evaluate golden images for faster day-0 setup, then build zero-touch
  provisioning for OS installation, single-use Tailscale enrollment, HLIMS
  registration, and machine convergence.
- [ ] Build `docs/architecture/host-collector-spec.md`
- [ ] Build `docs/architecture/remote-starter-spec.md`
- [ ] Build the centralized observability pilot described in `docs/architecture/observability.md`. (TODO: this needs to be updated for the design choices that came out of `docs/architecture/host-collector-spec.md`)
- [ ] Evaluate Bitwarden Secrets Manager for unattended automation.
