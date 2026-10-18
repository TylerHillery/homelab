# Infrastructure as Code Layout

Status: current conventions for OpenTofu roots; adapt when real operational
boundaries change. Ansible host configuration lives in
[`infrastructure/ansible/`](../../infrastructure/ansible/), while application-owned
configuration can live beside its service.

## Choose a root by ownership

An OpenTofu **root** is a directory with its own state, credentials, review,
apply, and recovery procedure. Group resources that share an owner and lifecycle
and should normally be planned and applied together. Split when resources must
exist independently, require different privileges or operators, or have a
different failure/restore boundary. A root need not be one provider, one service,
one machine, or one resource. Conversely, using the same provider is not a
reason to combine unrelated resources into a single state.

For example, keep records governed as one DNS zone together rather than making
an application root for every record. Keep storage that holds other roots' state
in a separate bootstrap root: it must survive those roots' destruction and
cannot depend on the backend it creates. An application's infrastructure can
stay beside that application when it has an independent lifecycle. Minimize
cross-root state dependencies; prefer provider data lookups or explicit,
non-secret inputs where those boundaries are stable.

Keep each root's state private and independently recoverable. Review plans for
the affected root, verify the result, and avoid concurrent applies unless the
backend's locking has been tested. Treat state and saved plans as sensitive even
when source configuration contains no secrets. Back up state before changing a
backend, and verify the migrated state against live resources before applying.
Provisioning a remote state store does not itself back up previous versions.

## Reuse only after a pattern emerges

Start with straightforward root configuration and provider-pinned dependencies.
Keep stable, non-secret values such as account IDs and resource names in the
root's configuration; OpenTofu variables are useful for real per-environment or
caller-supplied inputs, not mandatory for every literal. Use mise for pinned
tools and repeatable commands, and fnox for scoped runtime secrets. Do not put
secrets in tracked configuration or export them by hand on every run.

Extract a module when multiple real consumers need the same *behavior*, inputs,
and maintenance policy, not simply to avoid repeating a few resource blocks.
Avoid prematurely generic modules and a single all-providers root. Document each
root's ownership, credentials, backend, bootstrap steps, and recovery in its
README. Use separate scoped credentials for provider operations and state
storage when they have different permissions.

Current roots: [`cloudflare-dns`](../../infrastructure/opentofu/cloudflare-dns/)
manages DNS records, while [`state-bootstrap`](../../infrastructure/opentofu/state-bootstrap/)
manages the state bucket with independent local state. The DNS root uses that
bucket as its remote backend. The bootstrap root retains separate local state
because it cannot rely on the storage it creates.
