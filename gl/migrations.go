package gl

import "embed"

// Migrations contains the SQL migration files for the general ledger.
// Composition roots pass this to migration.Run from axon-base.
//
//go:embed migrations/*.sql
var Migrations embed.FS
