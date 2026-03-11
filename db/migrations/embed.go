package migrations

import "embed"

// Files contains the embedded SQL migrations, applied in lexicographic order.
//
//go:embed *.sql
var Files embed.FS
