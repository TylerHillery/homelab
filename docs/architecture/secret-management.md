# Secret Management

Status: current for interactive operations; unattended automation is planned.

## Current Model

Bitwarden Password Manager stores operator-managed secrets. fnox resolves them
and injects them only into the command that needs them.

```text
Bitwarden -> fnox -> command environment
```

The repository's `fnox.toml` contains provider references, not Bitwarden secret
values. The ignored `fnox.local.toml` contains the age recipient and encrypted
`BW_SESSION` bootstrap. The private age identity stays outside Git and can be
restored from Bitwarden.

Root mise tasks provide the operator workflow:

```sh
mise run bw:bootstrap-age
mise run bw:unlock
mise run fnox:exec <command>
```

## Rules

- Keep secrets out of Git, command arguments, logs, and generated plans.
- Use `env = "exec"`; do not inject secrets into every interactive shell.
- Fail when required values are missing.
- Use the narrowest credential supported by each system.
- Keep personal and organization vaults separate.
- Rotate credentials after exposure; encryption does not revoke old copies.
- Treat OpenTofu state as sensitive when providers persist secret values.

## Boundaries

Interactive operator credentials must not become reusable machine credentials.
Provisioning claims, application secrets, infrastructure credentials, and
recovery material require separate access scopes when automated.

## Recovery

Back up the age identity and Bitwarden recovery material independently. Test
restoration on a new operator device. Encrypted service backups remain outside
Git.

## Unattended Automation

Password Manager sessions are operator-oriented. Before enabling unattended
provisioning or deployment, evaluate Bitwarden Secrets Manager or another
backend that provides scoped machine identities, expiration, rotation, and
audit. Consumer commands should continue using fnox so the backend can change
without changing every workflow.
