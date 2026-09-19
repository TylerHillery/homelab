# Upstream fork

HLIMS is a hard fork of
[`tailscale/golink`](https://github.com/tailscale/golink), licensed under the
BSD-3-Clause [`LICENSE`](../LICENSE) retained in the service directory.

Initial fork commit:

```text
3f9300f2b03f29f1a2eb704ff419d849f47aabf4
```

The monorepo Git remote `golink-upstream` records the original project. HLIMS no
longer integrates upstream application changes after pivoting from stored short
links to inventory-backed redirect resolution.

The retained provenance covers the original starting point. HLIMS now provides:

- a stable, versioned JSON API;
- a normalized inventory schema including Products and Assets;
- CRUD APIs for redirect-critical inventory resources;
- canonical and ad hoc inventory-backed redirects;
- generated API clients;
- explicit SQLite migrations and generated queries;
- transport-independent deployment behind Tailscale Serve or another private
  ingress.
