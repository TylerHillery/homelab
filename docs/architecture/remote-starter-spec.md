# remote-starter — Design Draft

> **Status:** working draft, September 29, 2026. This is the current direction we are
> heading, **not a set of decisions**. Anything here can still change — nothing is set in
> stone. Open questions are marked ❓ and collected at the bottom.

## What it is

`remote-starter` is a small, **stateless** REST API agent that runs on each host in the
homelab and exposes service lifecycle operations — start, stop, restart, status — to
HLIMS.

The layering is control plane / data plane:

- **HLIMS** is the control plane: it decides what should happen and presents it in the UI.
- **remote-starter agents** are the data plane: one per host, translating API calls into
  actions against whatever supervisor that host runs (systemd, Docker Compose, SMF,
  OPNsense configd).

"Stateless" is a discipline, not just a description: the agent holds no local database,
keeps no desired-state, caches nothing. Every status read goes live to the underlying
supervisor, so the API can never disagree with reality. The only "state" is the config
file, which is deployment config, not runtime state.

### Non-goals

- **No desired-state reconciliation.** If a service should be running and isn't, noticing
  and acting on that is HLIMS's (or some future automation's) job. The agent just acts.
- **No boot persistence (enable/disable).** Whether a service starts on boot is declarative
  config and stays in Ansible. The API owns *runtime* state only — two sources of truth
  for boot state would be a bug factory.
- **Not a scheduler or orchestrator.** No cron-like scheduling, no cross-host ordering in
  v1. HLIMS can sequence calls itself if it ever needs to.

## Adapters

The adapter pattern is the core of the project — it's what lets one API speak systemd,
Compose, SMF, and configd. Each supervisor gets an implementation of a small
`Supervisor` interface; the HTTP layer never knows which one it's talking to.

```go
type State string // running | stopped | failed | starting | stopping | unknown

type ServiceStatus struct {
    Name    string         `json:"name"`
    Adapter string         `json:"adapter"`
    State   State          `json:"state"`
    Detail  map[string]any `json:"detail,omitempty"` // adapter-native info, passthrough
}

type Supervisor interface {
    List() ([]ServiceStatus, error)
    Status(id string) (ServiceStatus, error)
    Start(id string) error
    Stop(id string) error
    Restart(id string) error
}
```

The `Detail` bag is load-bearing: the abstraction normalizes state for decisions, but
never destroys information (systemd sub-states, container exit codes, SMF FMRI data).
Normalize for decisions, preserve for debugging.

### v1 adapters — chosen to mirror the actual fleet

| Adapter | Host(s) | Mechanism |
|---|---|---|
| **systemd** | Ubuntu minis + Surface Book 3 | ❓ D-Bus (godbus) vs shelling out to `systemctl` |
| **docker compose** | same hosts | ❓ Docker Engine API via compose labels vs shelling out to `docker compose` |
| **SMF** | Helios mini (bare metal) | shell out to `svcadm` / `svcs` |
| **configctl** | OPNsense tower (FreeBSD) | shell out to `configctl` |

Notes on each:

- **systemd** is non-negotiable day one — three of five machines run Ubuntu Server.
- **compose** falls out of the Docker adapter via the `com.docker.compose.project` label;
  no separate adapter needed unless shelling out proves simpler.
- **SMF is why the adapter pattern earns its keep**: the Helios box runs neither systemd
  nor rc.d. Key commands are `svcadm enable|disable|restart <fmri>` and
  `svcs -Ho state <fmri>`. Tentative state mapping: `online`→running,
  `offline`/`disabled`→stopped, `maintenance`→failed, `degraded`→failed — with the detail
  bag carrying the raw "degraded", since degraded means "up but not healthy" and HLIMS
  should notice. `svcadm refresh` (≈ reload) exists here as it does on systemd; see
  "reload" below.
- **configctl — not raw rc.d — is the correct surface on OPNsense.** It's what the GUI's
  Services page talks to, and going around configd risks the running daemon disagreeing
  with OPNsense's config state.

**rc.d deliberately deferred.** The only FreeBSD host is the OPNsense box, where configctl
is the right interface. A generic rc.d adapter solves for a plain-FreeBSD box that doesn't
exist. Revisit if one ever does — or if a manually-installed package on OPNsense turns out
to be invisible to configd (the one legitimate edge case).

### Later adapters (not v1)

- **k3s** (`kubectl rollout restart` on deployments) — the obvious next adapter given the
  3-node cluster.
- supervisord, s6/runit — only if a host ever needs them.
- **reload**: systemd, SMF, and supervisord support it; Docker doesn't. If exposed at all,
  unsupported adapters return 501. Probably v2.

### The hard part, named honestly

State normalization across supervisors is where adapters go to die. Each supervisor has
its own vocabulary — systemd's ~10 states, Docker's `restarting` crash loops, SMF's
`degraded` — and the mapping table *is* the product. Expect this to be fiddly; it's the
real work of the project.

## API design — Pitchfork as inspiration, not template

jdx's **Pitchfork** (process manager for local development) has a clean HTTP API and is a
good starting skeleton: it showed us the shape of action endpoints, idempotent no-ops,
and rich status objects. We take its bones; the dev-tool-specific organs stay behind.
Nothing here is copied — it's a jumping-off point, and we diverge wherever our problem
differs.

### What we take from Pitchfork

- **Action endpoints**: `POST /v1/services/{name}/start|stop|restart`. Same shape.
- **`noop: true` on idempotent no-ops**: `{ok: true, noop: true}` distinguishes "I started
  it" from "it was already running" — better than a bare 200. (We simplify Pitchfork's
  contract: always HTTP 200 on a handled request, `ok` true when the adapter did what was
  asked, `noop` saying whether anything changed. Real HTTP statuses — 404 unknown service,
  500 adapter failure — for the rest. Pitchfork's "200 with `ok: false`" is an API smell
  we don't inherit.)
- **Rich status objects**: pid, uptime, cpu/memory where cheap (gopsutil + adapter detail).
  HLIMS wants to display this.
- **Log tailing as NDJSON** (`GET /v1/services/{name}/logs/tail`, v2): the operator loop is
  "restart it, then watch the logs to confirm it came up healthy" — streaming that through
  the same API is genuinely valuable. Each adapter fetches logs differently (journalctl,
  `docker logs`, …), so this is v2, but don't lose it.
- **`GET /v1/stats`** (v2, cheap): host-level stats for an HLIMS overview page.

### What we deliberately leave behind

- **Namespaces, projects, worktrees, groups, proxies** — Pitchfork needs these because it
  discovers dev processes from project directories. Our services come from a static config
  allowlist; we don't have the problem these endpoints solve. Don't import the complexity.
- **enable/disable** — boot persistence stays in Ansible (see non-goals).
- **Process tree** — dead weight here; pid in the detail bag suffices.
- **Qualified IDs** (`namespace/name` URL-encoded as one path segment) — our names are
  flat, unique config keys, so `{name}` is a plain path segment and the adapter rides along
  as a field. Simpler, no encoding dance.

### Endpoint sketch (v1)

- `GET /v1/services` → all configured services, live state
- `GET /v1/services/{name}` → one service
- `POST /v1/services/{name}/start|stop|restart` → `{ok: true, noop: bool}`

## API-first contract strategy

The OpenAPI spec is the contract; code is generated from it — never the reverse.

- `openapi.yaml` lives in the monorepo where both HLIMS and remote-starter can reach it.
  It is the single source of truth.
- **oapi-codegen** generates the server stubs, the client, and the shared types from the
  spec. (Stronger than client-only generation: the server can't drift from the contract
  because it's generated from it.)
- Workflow: edit the YAML → `go generate ./...` → everything updates. CI regenerates and
  fails on `git diff --exit-code` so nobody merges an API change without regenerating.
- **Generated client + thin hand-written wrapper** is the expected pattern. Generated code
  is correct but not ergonomic (no retries, no nice error types) — the wrapper is where
  the humanity lives.
- This makes the API language-agnostic: whatever language HLIMS is written in, it
  generates its client from the same YAML. Rewriting remote-starter in another language
  later doesn't move the contract. (Precedent: this is how Oxide does it — Nexus's API is
  an OpenAPI spec with generated clients via progenitor.)

❓ Keep the spec minimal on day one. Five endpoints should take an hour to spec, not a
week. The value compounds as the API grows; the adapters are where the real work is.

## Language: Go

Decision (lightly held): **Go**.

- Time to a working v1 is an evening or two; stdlib `net/http` plus a small router is the
  whole server.
- The Docker client library is first-class (Docker itself is Go) — the day-one adapter
  swims with the current.
- Single static binary, `GOOS=linux go build`, scp to each host. Trivial deploys.
- The project's complexity budget belongs in the adapter matrix and state normalization,
  not in the language. Go gets out of the way.

Rust was considered and deferred: its strengths (zero-cost abstractions, fearless
concurrency) don't match this project's hard problems, and tokio + lifetimes would tax
momentum on a side project. Rust reps are better spent on a future project where they pay
off — and because the API is spec-first, a later rewrite wouldn't disturb HLIMS anyway.

## Auth & security posture

The agent is root-equivalent: anyone who can call it owns the box. The posture is layered,
cheapest strong layer first:

1. **Tailscale is the primary boundary** ("ingress as auth"). The agent binds to the
   Tailscale interface only; tailnet ACLs (tracked as code) restrict which nodes can reach
   the port. This isn't flat-LAN security theater — every connection is WireGuard-encrypted
   (which moots the usual plaintext-token-over-HTTP warning) and identity-bound per node.
2. **Bearer token header anyway** (~10 lines, Pitchfork-style). The ACL answers "which
   *node* may connect"; the token answers "which *process* on that node." Nearly free
   defense in depth, already in place if anything ever sits past the tailnet.
3. **The config allowlist is the blast-radius limiter.** Even with API access, only listed
   services are controllable. This is inherent to the design — no extra code — and it's the
   highest-leverage control: it turns "API compromise = full root" into "API compromise =
   can bounce postgres."
4. **Audit-log every mutating call**: who, what, when. (Destination TBD — local file is
   fine for v1.)

⚠️ **Router caution**: remote-starter on the OPNsense box manages the network services of
the router itself. Only list services that are safe to bounce remotely (Unbound: fine).
Anything that could kill the LAN interface means a walk to the HP tower with a keyboard.
❓ Do we want a per-service `readonly` (status-only) flag in config? Probably yes.

## Config sketch

```toml
[server]
listen = "100.x.y.z:8080"   # Tailscale IP only
token  = "..."               # or env var REMOTE_STARTER_TOKEN

[[services]]
name    = "postgres"
adapter = "systemd"
unit    = "postgresql.service"

[[services]]
name    = "link-library"
adapter = "compose"
project = "link-library"     # matched via com.docker.compose.project label

[[services]]
name    = "unbound"
adapter  = "configctl"
service  = "unbound"
```

Notes: the allowlist doubles as the security boundary (see above). A single Go binary
carries all adapters compiled in; config selects per host. The agent on the Helios box
never touches the systemd adapter — dead code there, and that's fine. No plugin system
needed for v1, just a `map[string]Supervisor` registry.

## v1 scope

- Four adapters: systemd, compose, SMF (`svcadm`), configctl
- Endpoints: list, get, start, stop, restart (with `noop` semantics)
- TOML config, bearer token, Tailscale-only bind
- Audit log (local file)
- Test seam: the `Supervisor` interface — unit tests use a fake supervisor and never need
  root; the Docker adapter can be integration-tested for real

## v2 ideas (not committed)

- Log tailing (NDJSON stream)
- `GET /v1/stats` host overview
- k3s adapter (`rollout restart`)
- mTLS in addition to / instead of bearer token
- `reload` endpoint (501 where unsupported)
- Per-service `readonly` flag (maybe v1 — see router caution)

## Open questions ❓

1. HLIMS's language — mostly moot thanks to codegen, but affects whether the thin client
   wrapper can be shared.
2. Compose adapter: Docker Engine API via compose labels, or shell out to `docker compose`?
   (Leaning Engine API — the Go client is first-class — but shelling out is faster to build.)
3. systemd adapter: D-Bus via godbus, or shell out to `systemctl` for v1? (Shelling out is
   the pragmatic v1; D-Bus is the "real" way.)
4. Sync-with-timeout duration for start/stop/restart calls — 30s? per-service override?
5. Audit log destination and retention.
6. Does HLIMS grow automation on top later ("restart when the health check fails") with
   remote-starter staying dumb? Assumed yes — that's the layering we want — but it should
   be said out loud before anyone builds it.
7. Per-service `readonly` flag for v1? (Router caution above says probably.)
