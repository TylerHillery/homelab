# HLIMS Collector → Ingestion Pipeline — Spec

**Status:** brainstorming — capturing ideas and direction. Last major pass:
2026-09-30.
Nothing below is decided unless marked **[DECIDED]**. **[OPEN]** = undecided
(lean noted where one exists). **[IDEA]** = worth exploring, no lean yet.
**Author:** Tyler Hillery (with Muse)

## 1. The pipeline at a glance

The shape Tyler landed on Sept 30 (direction, details open):

```
collector (per host; adapter pattern; one binary per OS)
  → local file spool → Fluent Bit → central NATS JetStream (CAPTURES stream)
  → transform consumer (stateless, bounded parallel)
  → central JetStream (RECORDS stream)
  → publisher (per-host lanes, watermark)
  → HLIMS API (idempotent UPSERT, observed columns only)
  → SQLite entities
```

Two durable streams, two custom consumers, one stateless transform, one
stateful publisher. Raw history tiers to a blob store; audit failures
quarantine; nothing is ever silently dropped.

```mermaid
flowchart LR
    C[collector<br/>one binary per OS<br/>entity × OS adapters] -- "envelope (host_id, run_id)<br/>→ local file spool" --> SP[local file spool<br/>durable, one JSON file per run]
    SP -- "Fluent Bit tails spool<br/>→ central NATS JetStream" --> S1[(CAPTURES stream<br/>hlimes.capture.raw)]
    S1 -- "transform consumer<br/>pull, bounded parallel<br/>explicit ack" --> T[transform<br/>stateless pure function<br/>raw→staging→intermediate→final<br/>Write-Audit-Publish gates]
    T -- "audit failure<br/>reason in headers" --> Q[(quarantine stream)]
    T -- "publish hlimes.records.&lt;host_id&gt;<br/>Msg-Id host:run" --> S2[(RECORDS stream)]
    S2 -- "publisher<br/>per-host serial lanes<br/>watermark check" --> P[publisher]
    P -- "POST idempotent UPSERT<br/>observed columns only" --> H[HLIMS API]
    H --> G[(SQLite entities)]
    S1 -.-> A1[archiver → blob store<br/>[IDEA] Benthos fit]
    S2 -.-> A2[archiver → blob store]
```

## 2. Core concepts

### 2.1 Run as the unit of identity

- Every artifact in the flow is keyed by **(host_id, run_id)**: envelope,
  stream messages, quarantine entries, HLIMS lineage, progress events.
- **run_id is a per-host monotonic sequence, minted only by the host
  collector** (Tyler's position Sept 30). Ordering is per host, never
  global. run_ids order *observations*; only the observer can mint that
  order.
- A run captures host dimensional details at one point in time. Two
  back-to-back runs with identical state are two distinct runs.
- Dedup is on run_id (delivery), never on content.
- **[DECIDED] Always insert captures**, even when nothing changed (hash the
  canonicalized capture, store a `changed` boolean). The heartbeat —
  "hasn't checked in for 30 days" — is a feature.
- **[DECIDED] Per-row lineage:** (`run_id`, `rule_name`, `rule_version`).
  You can always answer "why does HLIMS think this?"

### 2.2 Parallelism model

- The entire pipeline runs in parallel across run IDs and hosts. No ordering
  is needed between runs — except at the publisher's per-host commit.
- The per-host high-water mark is one mechanism doing two jobs: idempotency
  (redelivered runs skip) and ordering (older runs that finish late skip).

### 2.3 Invocation-agnostic transform

- `transform(host_id, run_id, raw_capture) -> final_records` is a pure
  function of its inputs — no transport imports, no persisted state.
- The NATS consumer, a manual CLI run, and tests all call the same
  entrypoint. What invoked it is not its business.

## 3. Collector

**[DECIDED]** One binary per OS from a single codebase — Ubuntu,
FreeBSD/OPNsense, illumos/Helios. No interpreter or runtime dependency on
the host.

**[DECIDED]** Adapter pattern: **entity collectors** (cpu, memory, disk, nic,
os, system) × **OS adapters** (linux, freebsd, illumos). Schema defined once
per entity; adapters fill it. Same philosophy as the Ansible roles: uniform
interface, OS-specific implementation. The envelope and transport code is
shared.

The collector's job, per Tyler's Sept 30 outline: **collect, assign
(host_id, run_id), emit the envelope.** It does not schedule itself and
does not choose the destination — it collects and writes the capture to
the local spool; the shipper worries about everything downstream of that.

- **[OPEN] Language — leaning Rust.** Tyler's reasons: likely C interop for
  low-level collection, small binary, memory efficiency. Go and stdlib-only
  Python remain options. Nothing chosen.
- **[OPEN] What each adapter uses — unevaluated.** osquery is one candidate
  for Linux/FreeBSD (adapter shells out to `osqueryi --json`, no Thrift/SDK;
  dependency never leaks past the adapter boundary). **No illumos support**
  — the Helios box needs a hand-rolled adapter regardless, which is what the
  adapter interface is for. FreeBSD table coverage is thinner than Linux;
  verify on OPNsense before depending on it. Tyler doesn't know this space
  yet; a broad survey is still owed.
- **[OPEN] Envelope contract** (proposed; versioned, never broken once set):
  `{host_id, run_id, captured_at, collector_version, entities: {...}}`.
  Schema recommendation (Tyler asked Sept 30; assistant's take, not
  decided): **CloudEvents for the envelope, JSON payload, JSON Schema
  validation** — not protobuf. CloudEvents is the CNCF standard for
  exactly this (event envelope + typed arbitrary payload), has a NATS
  protocol binding and multi-language SDKs, and the envelope maps ~1:1
  (`id`←run_id, `source`←host_id, `type`←e.g.
  `com.hlmis.inventory.run.v1`, `time`←captured_at, `data`←entities).
  Protobuf/Buf evaluated and set aside for the edge format: the entities
  payload is open-world (per-adapter facts, constantly evolving) — a
  closed .proto would force `Struct`/`Any` or endless schema churn — and
  binary encoding kills spool debuggability (`cat` the spool is a feature
  in a homelab). Buf only matters if protobuf is chosen; it isn't.
  OTel LogRecord also set aside *as the edge format*: the run_id-as-identity
  mapping is lossy and it couples the entity model to OTel's release
  train — but a downstream transformer can already map envelope →
  LogRecord if inventory ever needs to appear in the o11y stack, so keep
  the edge format his own and translate at the boundary.
- **[IDEA] Schema options surveyed** (Sept 30 — discussed, none decided;
  the direction above is the assistant's recommendation, not Tyler's
  decision):
  - **CloudEvents** (leading): CNCF envelope standard (`id`, `source`,
    `type`, `time`, `data`), NATS protocol binding, SDKs in every target
    language. Envelope maps ~1:1 to the proposed contract. To read:
    https://cloudevents.io/ (also in §15).
  - **JSON Schema**: validate the envelope at stage boundaries. No
    toolchain, no codegen to keep in sync across three OS targets — a
    library call per stage. Pairs with the CloudEvents direction (validate
    the JSON binding).
  - **Protobuf + Buf** (set aside for the edge format): closed schema
    fights the open-world `entities` payload (would churn per adapter or
    degenerate to `Struct`/`Any`); binary encoding kills
    `cat`-the-spool debuggability; protoc/Buf toolchain tax for a
    one-person, three-target build. Buf only matters if protobuf is
    chosen — it isn't.
  - **OTel LogRecord** (set aside as the edge format): run_id-as-identity
    maps lossily, and it couples the entity model to OTel's release train.
    Viable as a *downstream translation* (envelope → LogRecord in a
    transformer) if runs ever need to appear in the o11y stack.
  - **Collector output as "structured log"** (framing note): the run file
    is shaped like a structured log record (timestamped JSON, full
    context), but treat it as an *event*, not a log line — logs are for
    humans grepping, events are for machines deriving. Practical
    consequence is the same: JSON, validated, never a bare string.
- **[DECIDED] Delivery — per-host edge shape** (Tyler, Sept 30): two
  processes per host. (1) **Collector** — NATS-ignorant; collects, assigns
  `(host_id, run_id)`, and durably writes each capture to a local file
  spool. Knows nothing about brokers, shippers, or central's address.
  (2) **Shipper** — tails the spool and publishes to central NATS
  JetStream. Knows nothing about collection logic. The spool directory
  format (one JSON envelope per file, naming/ordering convention) is the
  contract between them — either side is replaceable without touching the
  other. If the shipper dies or central is down, the spool piles up and
  drains on recovery; durability is the spool's job, not the transport's.
- **Shipper: Fluent Bit — current direction** (Tyler, Sept 30: "settled on
  for now" — leaning, not locked), consolidated with the future metrics
  agent: one agent, two pipelines — file spool → NATS (inventory,
  at-least-once) + host metrics → ClickHouse/Prometheus (loss-tolerant).
  Inventory rides as opaque file/log events; no data-model fight. Why
  Fluent Bit leads: C codebase with FreeBSD ports (plausibly the only
  candidate across Ubuntu/FreeBSD — **illumos build unverified**); tiny
  footprint suits the OPNsense box; NATS output exists and is documented
  (**JetStream publish-ack semantics not yet verified against primary
  docs** — do not assume at-least-once until checked). Gaps: no native
  ClickHouse output (bridge via `http` output to ClickHouse's HTTP insert
  interface). Evaluated and set aside: NATS leaf per host (see §12); MQTT
  bridge; Vector (best NATS support on paper incl. `Nats-Msg-Id` templating,
  but no official FreeBSD and Tyler's maintenance doubt unresolved); OTel
  Collector + contributing NATS (give-back play, not the critical path —
  revisit as a standalone itch-scratcher after the pipeline runs); Benthos
  as shipper (fine tool, but redundant once Fluent Bit covers the job).
- **[OPEN] Delivery verification**: Fluent Bit packaging on OPNsense
  specifically (ports vs plugin); illumos/Helios — none of the shippers
  likely build there, so the Helios box probably keeps a custom
  `collector ship` fallback (per-host variance is already in the design);
  exact spool file naming/rotation/retention convention.
- **[OPEN] Execution model:** one-shot via external cron/systemd timer vs
  long-running daemon. Scheduling stays external either way — never in
  collector code.
- **[IDEA] Manual trigger without in-code scheduling.** One binary, two
  modes: `collector run` (one-shot, cron-invoked) and `collector serve`
  (tiny HTTP daemon; `POST /collect`, `GET /status`). Both call the same
  collect function. Auth posture: Tailscale-bind + token, same as
  remote-starter — and remote-starter ("run this binary over HTTP") may
  ultimately own this job; connect the two designs before building either.

Prior art (survey still owed): osquery (candidate), Chef Ohai / Puppet
Facter / Ansible `setup` (fact architectures worth studying), OTel
Collector (evaluated — wrong fit: metrics/logs/traces ≠ entities; keep as
possible monitoring sidecar), Duckservability (DuckDB-over-/proc PoC,
Linux-only, fun prior art, not a candidate).

## 4. Transport: NATS

Edge shape (Tyler, Sept 30): collector → local file spool → Fluent Bit →
central JetStream. No per-host NATS leaf — evaluated and set aside (see
§12); edge durability is the spool's job.

- **Central JetStream owns the durable streams.** Stages never chat
  *through* NATS mid-computation — streams are the durable entrances
  between stages ("NATS is the door, not the hallway").
- **Two streams to start** [OPEN names]: `CAPTURES` (raw) and `RECORDS`
  (transformed). Subjects (proposed): `hlimes.capture.raw`,
  `hlimes.records.<host_id>`, plus a quarantine subject. See §13 on the
  `hlimes`/`hlims` spelling.
- **NATS subjects are the registry.** The publisher picks the typed subject;
  one small dedicated consumer per subject (fetch → invoke → ack). No router
  or dispatcher process — the broker's subject matching is the dispatch.
  (An earlier dispatcher-with-handler-registry sketch was superseded Sept
  30 per Tyler's pushback.)
- **Msg-Id `<host_id>:<run_id>`** on every publish; the JetStream dedup
  window absorbs crash-republish duplicates.
- **[IDEA] Tiered history.** JetStream holds recent (hot); an archiver
  consumer per stream copies aged messages to a blob store (cold —
  S3-compatible/MinIO or plain files). Mechanism: interest-based retention —
  a message stays in the stream until *both* the stage consumer and the
  archiver have acked. The archiver is the natural Benthos/Redpanda Connect
  fit: pure plumbing, zero custom logic, a few lines of YAML.
- **[IDEA] Metrics share the bus, not the pipeline** (Tyler, Sept 30 —
  inventory = slowly-changing dimensions, metrics = facts). NATS as the
  spine and Fluent Bit as the edge agent extend naturally: Fluent Bit's
  native cpu/mem/disk/net inputs publish to `hlims.metrics.>` subjects
  alongside inventory, with a separate NATS → ClickHouse consumer doing
  downsampling/retention. But metrics should *not* ride the file spool —
  a missed 10s CPU sample is worthless (nobody replays it), while a lost
  inventory run loses unrecoverable history; the spool's durability exists
  for data you can't re-observe. Likewise the transform → publisher →
  HLIMS path is entity-shaped (watermarks, idempotent UPSERT) and doesn't
  apply — metrics want last-write-wins per timestamp, out-of-order is
  normal, TSDBs handle it natively. Same transport, different workload.
- **[IDEA] The fabric is payload-agnostic; OTel rides it as OTLP**
  (Tyler, Sept 30). Nothing in the transport constrains payload shape —
  the inventory pipeline is the first application on a general fabric, not
  the fabric itself. OTel data (traces/metrics/logs) can ride NATS as
  OTLP-encoded payloads on `otel.>` subjects; Fluent Bit already speaks
  OTel natively, and NATS-receiver components exist for feeding OTel
  Collectors centrally (Synadia's, grepplabs'). Two caveats: keep the OTLP
  encoding — inventing a custom trace envelope forks you off the ecosystem
  (Tempo/Jaeger/ClickHouse interop is the point); and don't file-spool
  spans — volume makes per-span files insane, use NATS/Fluent Bit buffering
  or the Collector's own disk queue instead. Honest note: OTel's native
  topology (SDK → Collector → backend) is well-trodden; routing through
  NATS buys buffering and a single bus, but the fabric doesn't demand that
  everything ride it.
- **[IDEA] The shared investment is event schema, not transport** (Tyler,
  Sept 30 — structured-logging / wide-events philosophy, Charity Majors).
  The fabric is the easy part; the durable common work across inventory,
  metrics, and future OTel is the DTO/event schema, structured-logging
  patterns, and naming conventions (envelope fields, subject hierarchy,
  field naming). Adopt OTel semantic conventions where they apply
  (telemetry naming); inventory keeps its custom envelope because OTel has
  no entity model. Cardinality correction (Tyler, Sept 30 — he was right):
  the cardinality fear is Prometheus-shaped (in-memory series index per
  label set), not universal. On a columnar store like ClickHouse,
  high-cardinality columns are cheap — that's the economic foundation wide
  events rest on. So: wide metric events → ClickHouse as the analytics
  store (derive aggregates at query time; aggregation at write time is a
  one-way trip), Prometheus kept as the scrape/alerting layer, not the
  analytics DB. Note the pipeline already *is* this architecture: raw
  captures are wide events (unique `run_id`, ordered per host, no
  aggregation at write time) and stages derive from that singular source —
  o11y 2.0 for inventory, just not called that.

## 5. Transform consumer

One consumer to start: consumes raw captures, applies normalization and
cleanup rules, writes the completed state back to NATS. The transform
function itself is completely stateless.

- **Stages [DECIDED]:** `raw → staging → intermediate → final` (Tyler's
  names — no medallion jargon), with **Write-Audit-Publish gates
  [DECIDED]** between stages. At each gate: write the stage → run audit
  queries → promote only if they pass. Audits are dbt-test-shaped: a query
  that must return zero rows. The gate *is* the mechanism.
  - *raw* — the capture payload, verbatim.
  - *staging* — raw shredded into typed tables; envelope validated.
  - *intermediate* — syntactic normalization (lowercase, strip, vendor
    maps) plus entity resolution against the gold catalog.
  - *final* — fully resolved entity set, referential integrity checked,
    validated against the HLIMS ingest schema.
- **[OPEN] DuckDB as the transform engine — Tyler's lean Sept 30:**
  stateless (in-memory/temp per run), *not* a persisted database. The
  statelessness deletes the whole DuckDB write-contention story. Only raw
  captures are durable, and that's the tiered store's job (§4), not
  DuckDB's. (This revisits the earlier "DuckDB = history" line — see §9.)
- **Consumer shape:** pull subscription on the captures stream; fetch
  batches (~2× max_parallel); each message goes to a bounded worker pool
  (**max_parallel configurable, e.g. 3–5** — the semaphore doubles as fetch
  backpressure). Per message: transform → publish records with
  Msg-Id `<host_id>:<run_id>` → **ack raw only after the records publish
  lands**. Transient errors → NAK with backoff. Permanent errors or
  max-deliver exceeded → publish to quarantine + ack (manual DLQ — stop the
  redelivery loop).
- **Audit failure → quarantine, not deletion.** The capture is marked
  quarantined with the reason; raw is preserved; nothing is silently
  dropped. Replay is a *new* run over the same raw capture, linked by
  lineage — not a paused workflow resuming.
- [OPEN] Normalization SQL framework: versioned views vs SQLMesh vs plain
  versioned functions.

## 6. Publisher

Owns the HLIMS commit. The only stateful component in the pipeline — and
its state is tiny: per-host max published run_id (the high-water mark).

- **Parallel across hosts; exactly one in-flight publish per host.**
  Per-host serial lanes inside the one publisher process (created on
  demand; lane count naturally bounded by host count). **The lane is the
  mutex; the watermark SQLite is crash recovery, not the lock.** (Why the
  lane must be the mutex: check-then-POST-then-bump is not atomic across
  the network — two concurrent same-host publishes could both pass the
  check, both POST, and race the watermark bump, leaving the watermark
  behind HLIMS's actual state. One in-flight publish per host eliminates
  the race by construction.)
- **Per lane, serially:** watermark check → stale (`run_id ≤ watermark`)?
  ack + skip : POST HLIMS → bump watermark → ack.
- **Crash ordering:** bump the watermark *after* the POST succeeds, never
  before. Crash between POST and bump → redelivery → benign idempotent
  re-POST. The reverse order could skip a run HLIMS never received.
- **Watermark store:** SQLite on the central host (`host_id →
  last_published_run_id`), loaded at startup. Caveat: all publishers must
  share the one store.
- **POST failures** → NAK with backoff. HLIMS down just buffers in the
  stream; transforms upstream are unaffected. This is the decoupling payoff
  of the records stream.
- The watermark lives in the publisher — **not** in HLIMS (see §7). It is
  pipeline-specific state: "newest observation I've applied."
- **[IDEA] Observed vs managed attributes.** External writers (manual
  annotations, labels, RMA notes) need a place to write without the
  pipeline clobbering them every N minutes. Split entity attributes into
  *observed* (pipeline-owned: what the collector saw) and *managed*
  (human/external-owned). The pipeline's UPSERT is **column-scoped to
  observed attributes only**; managed attributes have their own write path.
  Disjoint write sets = no conflict, no watermark involvement.

## 7. HLIMS boundary

- **[DECIDED] SQLite is the source of truth for entities.**
- **[DECIDED] HLIMS has no concept of run IDs** (Tyler's boundary, Sept 30)
  — pure entities, idempotent bulk UPSERT of normalized records, agnostic
  to how data arrives. It validates through its existing triggers and
  constraints; the pipeline never writes around them.
- **HLIMS takes no DuckDB dependency.** It sees HTTP POSTs from a client,
  like any other.
- **[DECIDED] One writer per fact.** The provisioner owns intent (profile,
  playbook, hostname — from the MAC registry). HLIMS owns facts (what the
  device reports). HLIMS may mirror the assigned profile read-only; it
  never originates it.
- [OPEN] Exact idempotent UPSERT contract (bulk shape, conflict targets).

## 8. Consumers as programs (not a framework)

- **No shared consumer framework.** Each consumer is an independent program
  in whatever language fits it (transform: whatever runs DuckDB best;
  publisher: whatever does HTTP + SQLite cleanly). What's shared is the
  *contract*, not code: envelope schema, subject naming, ack discipline
  (transform→publish→ack; NAK backoff; quarantine on max-deliver), Msg-Id
  dedup convention — documented once, implemented per consumer. Rule of
  three: extract shared code if/when a third consumer starts copying
  ack-handling.
- **If that day comes, the tool-shaped answer is a NATS consumer framework,
  not a workflow engine.** E.g. FastStream (Python): `@subscriber` over
  NATS JetStream with concurrency, retries, and DLQ, where the handler
  stays a plain function. Temporal/Hatchet/Restate/DBOS-class tools are
  *not* the answer here — they replace NATS as the substrate rather than
  framing consumers on top of it. Different architecture, not a consumer
  framework. (Set aside with the DAG-UI demotion — see §12.)
- **[IDEA] If the shared bits get extracted, the shape is a lightweight
  library, not a framework** (Tyler's cut, Sept 30): library = he calls it
  from his own `main()` (composable primitives: bounded pool,
  ack-after-success wrapper, per-host lane dispatcher, watermark store);
  framework = it calls him (decorators, it owns the loop); orchestrator =
  separate infrastructure (its own server cluster). The deciding axis is
  infrastructure: the library needs nothing beyond the NATS he already
  runs. Don't build it upfront — let it emerge from real duplication.
- **[IDEA] Declarative frontend as the eventual shape** (Tyler's TOML
  sketch, Sept 30): the library grows a config frontend — one file
  declaring each consumer (subject, stream, parallelism model, retry/DLQ
  policy, handler import path) plus a small per-language runner
  (`hlimes-flow run consumers.toml`) that supervises them. Ordering stays:
  consumers first, library second, TOML third.
- **Prior art: Benthos / Redpanda Connect.** *Processors* are the prior art
  for the pure-function task abstraction (per-message function, unaware of
  surrounding parallelism); their input/pipeline/output placement answers
  "where does the function run," where this design answers it with streams
  as stage boundaries. Benthos also has javascript/python code processors
  and a full `http_client` output (retries, backpressure, dynamic URLs) —
  but those cover the POST, the easy 10% of the publisher. The other 90%
  is the invariant (per-host mutual exclusion + watermark
  check-then-POST-then-bump), which needs durable per-key state no
  processor or output owns. Trade-off triangle, recorded: watermark-in-HLIMS
  would make the publisher stateless and Benthos-shaped — Tyler priced that
  and chose HLIMS purity, so the custom publisher is the conscious cost of
  that boundary. Honest carve-out: the *archiver* (JetStream → blob store)
  is where Benthos genuinely fits — pure plumbing, no code.

## 9. Working invariants

- **[DECIDED] SQLite is the source of truth for entities.**
- **[OPEN — was DECIDED, reconsidered Sept 30]** "DuckDB is the source of
  truth for history": Tyler now leans stateless — DuckDB as a pure
  transform engine (in-memory/temp per run), with only the *raw captures*
  needing durable persistence. Where raw history lives: the tiered store
  (JetStream hot → blob cold, §4) is the current direction; one-file-per-
  capture and DuckDB files remain fallbacks. The invariant that stands
  regardless: the queue (delivery) never becomes the history store by
  accident.
- **[DECIDED] Deterministic collector.** No LLM on the device. LLM is
  server-side, suggestion-only, human-approved, ratcheted into rules.
- **[DECIDED] Dependency discipline:** the collector stays thin (no DuckDB,
  no queue client); the queue knows nothing about DuckDB; the DuckDB job
  knows nothing about the queue; HLIMS takes no new dependencies.
- **[DECIDED] HLIMS has no run_id concept.** Pipeline bookkeeping
  (watermarks, run tracking) lives in the ingestion layer, never in HLIMS.
- **[OPEN] Identity cascade (proposed)** for matching captures to devices:
  TPM EK hash (v4) → `system_uuid` → (`product_id`, `serial_number`) →
  `os_machine_id` → MAC set. Surrogate key assigned on first sight.

## 10. Quarantine

Mechanics (settled direction): audit-gate *data* failures are routed by the
transform itself to the quarantine subject (reason in headers) and the
original message is acked; *infrastructure* failures go unacked → redelivery
with backoff. Raw is always preserved. Replay is a new run.

**[IDEA]** What would make it "quarantine" rather than just a DLQ is the
human lifecycle — and none of this is decided: how the human finds out
(ntfy? dashboard? logs?), whether a server-side LLM proposes resolutions
("these 3 unrecognized devices look like the same SSD — merge?"), how the
judgment gets codified into a deterministic rule, how replay is triggered.
Constraint that stands regardless: the LLM never runs on the device and
never writes directly.

## 11. Open questions

1. Collector language: Rust (lean) vs Go vs stdlib-only Python.
2. What each OS adapter uses: osquery vs alternatives — the broad survey is
   still owed (Linux, FreeBSD/OPNsense, illumos).
3. Delivery: Fluent Bit is the current direction (Tyler, Sept 30 —
   "settled on for now"); NATS leaf evaluated and set aside (§12).
   Remaining: Fluent Bit packaging on OPNsense; Helios likely needs the
   custom `collector ship` fallback; spool naming/rotation/retention
   convention.
4. Collector execution model: one-shot via external scheduler vs daemon.
5. Manual trigger mechanism: `collector serve` vs remote-starter ownership.
6. Stream/subject names; envelope contract (version it before first use).
7. Raw-history home if not the tiered NATS store: one file per capture?
   DuckDB files after all?
8. Blob store choice: MinIO (S3-compatible, self-hosted) vs plain files.
9. Transform consumer language; publisher language (independent choices).
10. `max_parallel` and other consumer tuning (fetch batch, NAK backoff,
    max-deliver threshold) — measure before tuning.
11. Normalization SQL framework: versioned views vs SQLMesh vs plain
    versioned functions.
12. HLIMS ingest API: exact idempotent UPSERT contract (conflict targets,
    column scoping for observed vs managed).
13. Identity cascade for device matching.
14. Live DAG/run visualization: wanted, not required — revisit only if
    consumer-lag dashboards prove insufficient.

## 12. Set aside (not deleted)

- **NATS leaf per host.** Evaluated Sept 30, set aside: once the local file
  spool exists, the leaf's durability value is marginal — the spool already
  owns edge durability, and a leaf would be a second moving part per host
  buying nothing the spool doesn't. Revisit only if the spool+shipper shape
  proves inadequate.
- **Temporal / Hatchet / Argo (+ Kestra, Windmill, Prefect).** Evaluated
  Sept 30 against: OSI-approved license (hard requirement), event-driven
  per-push jobs, live DAG UI, language-agnostic workers. Shortlist was
  Temporal vs Hatchet vs Argo. Set aside when the DAG UI was demoted from
  required to wanted: without it, this is a staged-queue problem, not a
  workflow problem — no human pauses, no branching beyond pass/quarantine —
  and the NATS-native design runs on already-committed infrastructure with
  zero new systems to operate. Revisit only if observability needs outgrow
  consumer-lag dashboards.
- **Excluded on license (not OSI open source):** Inngest (SSPL), Restate
  (BSL), n8n (Sustainable Use License).
- **Celery / Dramatiq / RQ:** zero-learning-curve fallback if a consumer
  ever degenerates to "just run this Python function." A retreat, not a
  plan.

## 13. Naming

- `hlimes` vs `hlims` — the spec currently uses both (`hlimes.capture.raw`,
  `hlims-ingest`). Normalize to one before any name becomes contractual
  (subject names, stream names, consumer names).

## 14. Non-goals (for now)

- Automated phoenix rebuilds — manual on-command first; automate after
  three boring manual runs.
- Cron-scheduled pipeline runs — event-driven per push.
- LLM inference anywhere on the device.
- Building the generic DAG platform beyond per-subject consumers.
- Generalizing for hypothetical future job types (provisioning, rebuilds)
  before the second job type arrives.

## 15. External resources

Methodology — logging, metrics, collection philosophy (Tyler's canon;
the pipeline's event design follows these):

- Charity Majors, "Live Your Best Life With Structured Events" —
  arbitrarily wide structured events, unique and ordered, no aggregation
  at write time; unique IDs are the highest-value data; aggregation is a
  one-way trip. https://charity.wtf/p/live-your-best-life-with-structured-events
- Charity Majors, "Have you heard? Clickhouse is winning the
  observability wars!" — columnar storage changes the cardinality
  economics; store one source of truth, derive everything from it; a
  better engine can't beat bad (three-pillar) architecture.
  https://charity.wtf/p/have-you-heard-clickhouse-is-winning
- https://loggingsucks.com/ — structured logging advocacy; Tyler's
  reference.
- https://cloudevents.io/ — CNCF event-envelope standard; recommended
  for the run envelope (NATS binding, multi-language SDKs). Spec to read
  before finalizing the envelope contract.

Systems to study:

- Oxide **oximeter** (https://github.com/oxidecomputer/omicron/tree/main/oximeter) —
  Oxide's metrics collection system, used by Omicron, runs on illumos.
  Research target for the Helios metrics story — especially relevant
  since Fluent Bit likely doesn't build on illumos and the Helios box may
  need its own metrics path.
- https://github.com/statsite/statsite statusite metrics agg server
- https://github.com/lyft/statsrelay consitsent hasing relay for statsd
- https://github.com/bitdriftlabs/pulse O11y proxy for very large metrics infra. Idea from statsite and statsrelay. Calls out the below three projects for now having
    - agrgregation (dropping a pod label to dervice service level aggregate metrics)
    - clustering (consistent hasing and routing at the aggregation tier)
    - automated blocking/elision based on control-palen drive configuraiton
- https://github.com/open-telemetry/opentelemetry-collector
- https://github.com/fluent/fluent-bit
- https://github.com/vectordotdev/vector
