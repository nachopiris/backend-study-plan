// Package migration implements the online backfill step of the
// customers.name -> customers.full_name expand/contract migration, between
// migrations/002_expand.sql and migrations/003_contract.sql.
package migration

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Backfill copies name into full_name for every customer row where
// full_name is still NULL — rows written before migrations/002_expand.sql's
// trigger existed, which the trigger never ran for and never will (it only
// fires on new writes, not retroactively).
//
// Backfill must process at most batchSize rows per batch, each batch in its
// own statement/transaction, instead of a single UPDATE touching every
// matching row. A single UPDATE over a large, live table holds its lock for
// as long as the whole statement takes and generates one huge burst of WAL —
// exactly what an online migration is meant to avoid; many small batches
// keep the table available to the running application throughout the
// backfill.
//
// Backfill returns the number of batches it ran (0 if there was nothing to
// backfill) and stops as soon as a batch updates zero rows. On any error, it
// returns the number of batches successfully committed so far alongside the
// error.
func Backfill(ctx context.Context, pool *pgxpool.Pool, batchSize int) (int, error) {
	return 0, errors.New("not implemented")
}
