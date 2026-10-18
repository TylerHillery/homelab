#!/usr/bin/env bash
set -euo pipefail

fnox="$1"
bw="$2"
session_config="$3"
secret_config="$4"
shift 4

# The encrypted local session is unlocked with the user's age identity.
# Neither the Bitwarden session nor the Cloudflare token enters argv or the journal.
export BW_SESSION="$("$fnox" --no-daemon --non-interactive -c "$session_config" get BW_SESSION)"
if [[ -z "$BW_SESSION" ]]; then
	echo 'Bitwarden session unavailable; unlock it before starting Caddy' >&2
	exit 1
fi
export PATH="$(dirname "$bw"):$PATH"
exec "$fnox" --no-daemon --non-interactive -c "$secret_config" exec --replace -- \
	/usr/bin/env -u BW_SESSION "$@"
