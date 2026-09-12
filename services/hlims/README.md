# HLIMS

HLIMS is the Home Lab Information Management System. It is based on Tailscale's
golink while an API-first catalog is introduced incrementally.

## Development

HLIMS owns its tool versions and tasks in this directory's `mise.toml`. The web
interface uses the directly maintained `static/base.css`; it has no Node.js,
package-manager, or CSS build dependency.

The server-rendered interface progressively enhances forms with a vendored,
pinned copy of HTMX 4.0.0. Native HTML form actions remain the fallback when
JavaScript is unavailable.

```bash
mise install
mise run check
mise run dev
```

Open <http://localhost:8090> while `mise run dev` is running. Air rebuilds the
embedded assets and refreshes the browser when Go, HTML, CSS, or JavaScript
files change. Use `mise run serve` when live reload is not needed.

From the monorepo root, use the namespaced form such as
`mise //services/hlims:test`.

## Upstream golink documentation

- [Fork point and upstream integration policy](docs/upstream.md)
- [Original golink documentation](docs/upstream-golink.md)
