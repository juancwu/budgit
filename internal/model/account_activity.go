package model

import "time"

// AccountActivityRow is a unified row representing either an account-scoped space
// audit entry or a transaction audit entry that belongs to the account. Exactly one
// of SpaceLog / TxLog is set.
type AccountActivityRow struct {
	SpaceLog *SpaceAuditLogWithActor
	TxLog    *TransactionAuditLogWithActor
}

func (r AccountActivityRow) Timestamp() time.Time {
	if r.SpaceLog != nil {
		return r.SpaceLog.CreatedAt
	}
	return r.TxLog.CreatedAt
}
