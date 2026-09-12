# Upstream fork

HLIMS is a hard fork of
[`tailscale/golink`](https://github.com/tailscale/golink), licensed under the
BSD-3-Clause [`LICENSE`](../LICENSE) retained in the service directory.

Initial fork commit:

```text
3f9300f2b03f29f1a2eb704ff419d849f47aabf4
```

The monorepo Git remote `golink-upstream` tracks the original project. Upstream
changes are reviewed and selectively integrated; they are not merged
automatically.

HLIMS intends to preserve short-link compatibility while adding:

- a stable, versioned JSON API;
- Device, Machine, Service, and Instance resources;
- generated API clients;
- a CLI and API-backed console;
- explicit SQLite migrations and generated queries.
