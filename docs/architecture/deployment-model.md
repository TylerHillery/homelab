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

## HLIMS Inventory Model

A **Deployment** is an optional Machine-local Docker Compose project grouping
related Instances. It records a stable project name, absolute working directory,
and ordered Compose file paths (including overrides), not whether the project
is currently running. Each container-backed member Instance records its Compose
service key.
A static site built into the same project may also be a member without being a
container: mark its Instance `static_content` and record its relative content
path, then link it to the exact server Instance with a `served_by` relationship.
Do not invent a Compose service key or backend listener port for static files;
the URL's external port still belongs on their verified Instance Endpoint.
Systemd is a service manager on the Machine, not a Compose-style Deployment.
An Instance records its own unit name, system/user scope, and the login name
for a user-scoped manager. Multiple units from the same scope and user appear
together under that Machine's **systemd units** section; there is no fake
Deployment per unit. A real `.target` could be modeled in the future if its
actual `Wants`, `Requires`, or `PartOf` relationships warrant it. Sharing a
manager does not imply shared restart behavior.

An Instance can belong to **zero or one** Compose Deployment *or* record one
systemd unit, not both. A Deployment belongs to one Machine. Instances with
neither retain their existing Service, name, Machine, and endpoint; a managed
Instance does not need a fake Deployment.
The same Service can have multiple independent Instances and Instances in
several Compose projects. In the Services console each Deployment shows *only*
its own Instances; independent Instances remain in their own group. The
Machine console shows each Deployment on its host with only its member services;
independent Instances keep the Machine's original Service grouping. The global
Service catalog still lists Instances, grouped by Compose Deployment for clarity.

An explicit Instance Dependency records that one exact Instance **uses** another
(for example, an API using a PostgreSQL Instance in a different Compose
project). It does not imply that Compose `depends_on` crosses project boundaries,
that systemd orders either unit, or that the provider is healthy. Do not store
database passwords or ad hoc start/stop commands in this inventory.
The distinct `served_by` relation records the actual server for a static-content
Instance; it does not claim a separate container or systemd unit.

HLIMS does not infer live state from an Instance row or a Compose file. The
console says “not checked” until an explicit read-only runtime observation
exists. A future status/control integration must select the correct Machine,
manager, scope, and account before issuing commands.
