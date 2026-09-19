# Deployment Model

HLIMS integrates with deployment systems without replacing them. The terms
below distinguish inventory, execution, placement, and network exposure so that
one tool is not expected to own every layer.

## Useful Terminology

These tools operate at different layers:

| Layer | Examples | Responsibility |
|---|---|---|
| Service manager | systemd | Starts, stops, restarts, and supervises host processes |
| Container runtime | Docker Engine, Podman | Runs individual containers |
| Single-host orchestrator | Docker Compose | Defines and coordinates related containers on one Machine |
| Fleet scheduler | Nomad, Kubernetes, Docker Swarm | Selects Machines and schedules workloads across them |
| Machine converger | mise bootstrap, Ansible | Applies host, systemd, firewall, and Compose configuration |
| Deployment controller | Komodo, Coolify, Kamal | Delivers new application versions |
| Ingress publisher | Caddy, Tailscale Serve, Traefik | Makes a backend reachable at a stable URL |
| Inventory and catalog | HLIMS | Records Machines, Services, Instances, endpoints, and relationships |

Systemd and Docker supervise workloads already assigned to a Machine. Docker
Compose coordinates containers on one Machine. Nomad, Kubernetes, and Docker
Swarm are schedulers because they can choose placement across a fleet.

The layers can be combined without conflating their responsibilities. For
example, Komodo can deploy a Docker Compose application to a Machine cataloged
by HLIMS, while Caddy publishes its public endpoint and Tailscale Serve
publishes a separate Tailnet endpoint.

## Initial Boundaries

HLIMS should initially recognize two workload formats:

```text
compose
systemd
```

Deployment providers are a separate concern:

```text
komodo
mise
ansible
manual
```

The distinction allows mise or Ansible to manage either Compose or systemd
while Komodo specializes in Compose. mise is the initial Machine converger;
Ansible remains a fallback while the approach in
[Machine Management](machine-management.md) is piloted. A single container can
still use a small Compose definition instead of introducing a separate
raw-Docker deployment model.

HLIMS remains authoritative for identity, placement, relationships, and
reachable endpoints. Deployment controllers remain authoritative for execution,
runtime status, logs, and release operations.
