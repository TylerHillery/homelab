// Package migrations provides the embedded HLIMS database migrations.
package migrations

import "embed"

// Files contains every SQL migration applied during HLIMS startup.
//
//go:embed *.sql
var Files embed.FS
