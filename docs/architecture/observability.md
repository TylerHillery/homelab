# Observability

Status: planned pilot.

## Goal

Query logs, traces, metrics, and deployment context from one private system.
Optimize for structured wide events, SQL investigation, bounded storage, and a
small operating footprint.

## Proposed Flow

```text
applications, journald, host metrics
              |
              v
      local OTel Collector
              |
              v
      central OTel gateway
              |
              v
         ClickHouse
              |
              v
           Grafana
```

Host agents retain checkpoints and bounded queues. The gateway normalizes,
filters, batches, and retries. Start with one node; clustering and a message bus
require measured need.

## Event Policy

Emit one structured completion event per request or operation. Include outcome,
duration, service and deployment identity, host, version, trace IDs, normalized
operation, and relevant domain context.

Never place credentials, authorization headers, cookies, or unnecessary
personal data in telemetry. Keep unbounded values out of metric labels.

## Pilot

1. Deploy one gateway, ClickHouse, and Grafana.
2. Add one host agent and one application.
3. Build pipeline-health and application dashboards.
4. Test gateway outage, full queues, restart, retention, and restore.
5. Add traces and test correlation.
6. Evaluate metric storage after logs and traces are stable.

Use insert-only database credentials for collectors and read-only credentials
for dashboards. Bind receivers to private interfaces. Measure volume before
setting retention.

HyperDX, VictoriaMetrics, or analytical lake storage remain alternatives only
after the pilot identifies a concrete gap.
