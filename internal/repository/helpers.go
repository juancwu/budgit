package repository

import "github.com/jmoiron/sqlx"

// WithTx runs fn inside a transaction. If fn returns an error, the transaction
// is rolled back; otherwise it is committed.
func WithTx(db *sqlx.DB, fn func(*sqlx.Tx) error) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
