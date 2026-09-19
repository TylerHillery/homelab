# Observability

Status: planned as a separate centralized observability project.

## Goal

Provide one place to query logs, traces, metrics, and deployment context from
every server and application. Optimize for wide structured events, SQL,
high-cardinality investigation, Grafana dashboards, and a small homelab
operating footprint.

## Recommended Pilot

Start with upstream OpenTelemetry Collectors, one single-node ClickHouse
database, and Grafana. Do not begin with the complete LGTM or ClickStack suite.

```text
Applications -> local OTel Collector agent --+
journald -----> local OTel Collector agent ---+-> central OTel gateway
host metrics -> local OTel Collector agent ---+          |
                                                        v
                                                   ClickHouse
                                                        |
                                                     Grafana
```

The central gateway provides normalization, filtering, batching, retry, and a
persistent sending queue. Each host agent keeps source checkpoints and a
bounded persistent queue so a central outage does not immediately lose data or
fill host disks. Kafka and ClickHouse clustering are not justified initially.

Use the standard OpenTelemetry ClickHouse exporter tables first. The official
Grafana ClickHouse datasource understands the conventional log and trace
schemas and still permits arbitrary SQL. Pin Collector versions and manage the
ClickHouse DDL explicitly after initial schema creation because exporter schema
changes can require migrations.

## Wide Events

Emit one context-rich structured event per request or service hop at completion
rather than many disconnected log strings. Events should include stable names
and enough dimensions to investigate unknown failures:

```text
timestamp, event_name, outcome, duration_ms
service.name, service.version, deployment.environment.name
host.name, instance_id, commit_hash
request_id, trace_id, span_id
operation, normalized route, status_code
domain-specific context needed to explain the outcome
structured error type, code, and message
```

Use consistent field names across services. Preserve high-cardinality IDs on
logs and spans, but never add request IDs, user IDs, raw URLs, or other
unbounded values as metric dimensions. Filter credentials, authorization
headers, cookies, and unnecessary personal data before data reaches a durable
queue.

## Why Direct ClickHouse

- Wide structured records and high-cardinality event fields fit its columnar
  storage model.
- SQL is the primary query interface.
- Grafana has an official ClickHouse datasource.
- ClickStack and HyperDX can be evaluated later against compatible data.
- ClickHouse can support separate future analytical datasets without forcing
  telemetry and general data to share tables, users, or retention policies.

Full ClickStack adds HyperDX, its Collector distribution, and MongoDB-backed UI
state. That may be worthwhile if its event-search and correlation workflow is
better than Grafana, but it should be demonstrated before adding those services
and backup surfaces.

The ClickHouse exporter currently has lower stability for metrics than for the
database itself. Trial metrics after logs and traces. If SQL over OTel metric
tables is too awkward, use single-node VictoriaMetrics for metrics while
retaining ClickHouse for wide logs and traces rather than adopting all of LGTM.

## DuckLake Evaluation

DuckLake 1.0 is production-ready as a lakehouse format, but the current
observability path is experimental:

- Quack, DuckDB's client/server protocol, is beta.
- `duckdb-otlp` can lose accepted rows that have not reached a durable commit.
- Canard Stack is an experimental query-only server with partial Prometheus,
  Loki, and Tempo APIs.
- Canard lacks full PromQL, LogQL, TraceQL, histogram query support, and
  full-text indexing.
- A dependable multi-client DuckLake adds PostgreSQL catalog metadata, object
  storage, retention deletion, snapshot expiry, and file compaction.

Keep DuckLake and Canard as a later learning experiment, sampled archive, or
secondary analytical store. Do not make them the only live telemetry backend
until ingestion durability and Grafana compatibility mature.

## Operations

- Bind application OTLP receivers to loopback and the gateway to its Tailnet
  address or another private authenticated network.
- Give the Collector an insert-only ClickHouse account and Grafana a read-only
  account with query limits.
- Start with a short measured retention period, such as 14 days for logs and
  traces, before choosing long-term policy.
- Monitor Collector queue depth, dropped records, export failures, ClickHouse
  rejected inserts, part counts, disk use, and query memory.
- Back up schemas, users, Collector configuration, Grafana provisioning, and
  dashboards. Treat raw telemetry as replaceable unless a specific dataset has
  stronger retention requirements.
- Test a central outage, Collector restart, full queue, and restore before
  onboarding every host.

## Adoption Phases

1. Deploy ClickHouse, Grafana, and one central Collector gateway.
2. Add one canary host agent with host metrics, selected journald units, and
   Collector self-telemetry.
3. Ingest one application's wide structured events and build fleet,
   application, and pipeline-health dashboards.
4. Stop ClickHouse and restart Collectors to verify queue durability and disk
   bounds.
5. Add traces and validate correlation through shared trace and request IDs.
6. Trial metric storage and decide whether ClickHouse or VictoriaMetrics is the
   better long-term metric backend.
7. Evaluate HyperDX against the existing ClickHouse data and DuckLake as a
   separate analytical experiment.
