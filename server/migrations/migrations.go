// Package migrations holds the SQL migrations, applied in file name order.
// Files are append-only: never edit one that has shipped, add the next number.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
