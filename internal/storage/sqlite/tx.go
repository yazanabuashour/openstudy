package sqlite

import (
	"database/sql"
	"fmt"
)

func rollbackUnlessCommitted(tx *sql.Tx) {
	_ = tx.Rollback()
}

func wrapDatabaseError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}
