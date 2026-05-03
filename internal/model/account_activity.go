package model

import "time"

// ActivityRow is a unified row representing either a space-scoped audit entry or a
// transaction audit entry. Exactly one of SpaceLog / TxLog is set. Used by both the
// account-level and space-level activity feeds.
type ActivityRow struct {
	SpaceLog *SpaceAuditLogWithActor
	TxLog    *TransactionAuditLogWithActor
}

func (r ActivityRow) Timestamp() time.Time {
	if r.SpaceLog != nil {
		return r.SpaceLog.CreatedAt
	}
	return r.TxLog.CreatedAt
}
