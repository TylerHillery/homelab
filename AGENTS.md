# Repository guidance

## HLIMS is pre-release

HLIMS has **never been released**. Its routes, API, schema, UI, names, and domain
model are provisional and can all be changed when a better design emerges. Do
not preserve backwards compatibility, add redirects or aliases for hypothetical
bookmarks, or keep obsolete behavior merely because it already exists. Discuss
tradeoffs on their merits and make the clean change.

Preserve real inventory and other user data when changing the design. The
private staging database needs a backup and reconciliation when its schema
changes. Before the first official release, revise the existing
`services/hlims/db/migrations/00001_inventory.sql` baseline rather than adding
another migration file.
