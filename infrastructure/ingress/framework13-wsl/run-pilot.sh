#!/usr/bin/env bash
set -euo pipefail

credential_file="${CREDENTIALS_DIRECTORY:?}/cloudflare-token"
if [[ ! -s "$credential_file" ]]; then
	echo 'Missing Caddy Cloudflare credential' >&2
	exit 1
fi

CF_API_TOKEN="$(cat "$credential_file")"
export CF_API_TOKEN

config_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
exec "$HOME/.local/bin/caddy-homelab" run --config "$config_dir/Caddyfile" --adapter caddyfile
