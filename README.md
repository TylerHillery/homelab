# Overview

This project sets up monitoring for my homelab.

## Metrics Collection

Each PC will run an [OTel Collector](https://opentelemetry.io/docs/collector/) as a [systemd-managed binary service](https://opentelemetry.io/docs/collector/install/binary/linux/#automatic-service-configuration) using the [host metrics receiver](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/hostmetricsreceiver). Each collector will use the [OTLP gRPC exporter](https://github.com/open-telemetry/opentelemetry-collector/tree/main/exporter/otlpexporter) to send metrics to a central deployed [gateway collector](https://opentelemetry.io/docs/collector/deployment/gateway/).

The gateway collector acts as a central aggregation point for all other collectors and is deployed on a single PC. The gateway collector will then fan out the data to multiple exporters:

1. [ClickHouse Exported:](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/clickhouseexporter)

    ClickHouse is self-hosted on one of the homelab PCs and serves as the long-term metrics datastore. [Grafana](https://grafana.com/grafana/plugins/grafana-clickhouse-datasource/) is also self-hosted and configured to query ClickHouse for internal visualization and analysis.

2. OTLP gRPC Exporter:

    Metrics are exported via OTLP gRPC to another OTel collector running as a sidecar alongside a backend API service. The side car collector will send metrics to the backend API service via HTTP. Both the sidecar collector and the backend API are deployed as Docker containers on a VPS. The backend API exposes a Server-Sent Events (SSE) endpoint, which streams metrics in real time to a SPA-based frontend served at `minilab.tylertries.com`.

```mermaid
flowchart LR
    subgraph Homelab["Homelab Metrics Collection"]
        subgraph Node1["PC #1"]
            H1[Host Metrics]
            C1[OTel Collector<br/>hostmetrics]
            H1 --> C1
        end

        subgraph Node2["PC #2"]
            H2[Host Metrics]
            C2[OTel Collector<br/>hostmetrics]
            H2 --> C2
        end

        subgraph Node3["PC #3"]
            H3[Host Metrics]
            C3[OTel Collector<br/>hostmetrics]
            H3 --> C3
        end

        subgraph Gateway["Gateway Collector"]
            G[OTel Collector<br/>Gateway]
        end

        C1 -->|OTLP gRPC| G
        C2 -->|OTLP gRPC| G
        C3 -->|OTLP gRPC| G

        subgraph Storage["Storage & Visualization"]
            CH[ClickHouse]
            Grafana[Grafana]
        end

        G -->|ClickHouse Exporter| CH
        CH --> Grafana
    end

    subgraph Cloud["VPS / Public Streaming Path"]
        Sidecar[OTel Collector<br/>Sidecar]
        Caddy[Caddy<br/>TLS + Reverse Proxy]
        API[Backend API<br/>SSE]

        G -->|OTLP gRPC| Sidecar
        Sidecar -->|HTTP| API
        API <--> Caddy

    end

```

## Hardware

HP ProDesk 600 G1 Mini x3

- Intel(R) Core(TM) i5-4590T CPU @ 2.00GHz
- 8GB RAM
- 256GB SSD