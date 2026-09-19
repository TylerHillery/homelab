# Secret Management

Status: adopt fnox with age-encrypted Git storage first; retain OpenBao as a
future backend.

## Goal

Give OpenTofu, mise bootstrap, CI, provisioning, and applications one
consistent way to receive secrets without sharing broad credentials between
trust domains or requiring an online secret service during initial recovery.

## Decision

Use fnox as the secret resolution and delivery interface. Initially store
values with fnox's native age provider as ciphertext in committed `fnox.toml`
files. Use SOPS with age only when a structured recovery document or another
consumer specifically requires the SOPS format.

fnox is MIT-licensed and supports local age encryption, profiles, hierarchical
configuration, temporary secret files, command-scoped environment injection,
and remote vault providers. It is not a frontend for SOPS: its age provider
stores independently encrypted values in `fnox.toml`.

Run consumers through an explicit boundary:

```text
fnox age provider -> fnox exec -> OpenTofu, mise, CI, or application command
```

If centralized identity, revocation, PKI, or dynamic credentials become a
concrete need, migrate values to OpenBao and change fnox provider references.
Consumer commands can continue using `fnox exec`.

## Trust Boundaries

Configure separate age providers, recipient sets, files, and automation
identities for each responsibility:

```text
infrastructure/   Cloudflare, DigitalOcean, and other OpenTofu providers
provisioning/     Tailscale enrollment and narrow HLIMS enrollment access
deployment/       mise bootstrap, registry, and deployment credentials
applications/     Per-application and per-environment runtime secrets
recovery/         Offline bootstrap and break-glass material
```

No Machine or automation identity receives access to every boundary. A
provisioned Machine must not be an age recipient for infrastructure-provider,
fnox operator, or reusable Tailscale enrollment credentials.

Use dedicated age identities instead of unencrypted SSH private keys. Keep
private identities outside Git and back them up separately. Removing an age
recipient does not revoke old ciphertext in Git history, so rotate the
underlying secret after a recipient or identity is compromised.

## Integration Model

| Consumer | Access Pattern |
|---|---|
| Interactive OpenTofu | `fnox exec -- tofu plan` with the operator identity |
| OpenTofu in CI | A dedicated CI age identity limited to infrastructure secrets |
| mise bootstrap | `fnox exec -- mise bootstrap` for local secret inputs |
| ZTP enrollment broker | A dedicated provisioning identity with no infrastructure access |
| Applications | Per-workload secrets delivered as environment variables or temporary files |

Set `env = "exec"` so secrets are not injected into the interactive shell by
default, and set `if_missing = "error"` so automation fails closed. Automatic
shell loading is convenient but creates unnecessary ambient exposure for every
process launched from that shell.

Do not place values delivered by fnox into OpenTofu-managed resources unless
the target provider requires it. Values stored in OpenTofu state remain
sensitive data even when marked `sensitive`.

## Remote Bootstrap

`mise bootstrap remote` deliberately does not copy the initiating machine's
environment to SSH targets. An attended run can use `--prompt-secrets`.
Unattended secret-bearing convergence requires fnox, persistent reviewed
configuration, and a narrowly scoped age identity on the target. The Machine
then runs `fnox exec -- mise bootstrap` locally; remote bootstrap remains useful
for installing mise and converging phases that do not need secrets.

Never place decrypted secrets in the bootstrap source archive. A Machine's age
identity may decrypt only that Machine's application or deployment values. It
must not decrypt the credentials used to enroll other Machines.

## Recovery

Git contains the encrypted fnox configuration, but age private identities must
be recoverable independently. Keep offline copies of the operator and recovery
identities and a tested procedure for restoring them.

Use SOPS/age for a structured recovery bundle only when that format is useful,
for example to hold identity-provider recovery data and references to offline
backups. Do not maintain a complete second copy of routine secrets in SOPS.

## Adoption Phases

1. Add fnox through mise and create separate operator and automation age
   identities.
2. Define age providers and recipient sets by trust boundary before importing
   existing Pulumi ESC secrets.
3. Run OpenTofu and local mise bootstrap commands through `fnox exec`.
4. Design per-Machine age enrollment before enabling unattended remote
   templates that contain secrets.
5. Move application secrets one workload at a time and test identity rotation
   and recovery.
6. Evaluate OpenBao only when the limitations of encrypted Git storage create a
   concrete operational or security problem.
