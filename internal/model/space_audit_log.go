package model

import "time"

type SpaceAuditAction string

const (
	SpaceAuditActionRenamed         SpaceAuditAction = "space.renamed"
	SpaceAuditActionDeleted         SpaceAuditAction = "space.deleted"
	SpaceAuditActionMemberInvited   SpaceAuditAction = "member.invited"
	SpaceAuditActionMemberJoined    SpaceAuditAction = "member.joined"
	SpaceAuditActionMemberRemoved   SpaceAuditAction = "member.removed"
	SpaceAuditActionInviteCancelled SpaceAuditAction = "invite.cancelled"
	SpaceAuditActionAccountCreated  SpaceAuditAction = "account.created"
	SpaceAuditActionAccountRenamed  SpaceAuditAction = "account.renamed"
	SpaceAuditActionAccountDeleted  SpaceAuditAction = "account.deleted"
)

type SpaceAuditLog struct {
	ID           string           `db:"id"`
	SpaceID      string           `db:"space_id"`
	ActorID      *string          `db:"actor_id"`
	Action       SpaceAuditAction `db:"action"`
	TargetUserID *string          `db:"target_user_id"`
	TargetEmail  *string          `db:"target_email"`
	Metadata     []byte           `db:"metadata"`
	CreatedAt    time.Time        `db:"created_at"`
}

type SpaceAuditLogWithActor struct {
	SpaceAuditLog
	ActorName       *string `db:"actor_name"`
	ActorEmail      *string `db:"actor_email"`
	TargetUserName  *string `db:"target_user_name"`
	TargetUserEmail *string `db:"target_user_email"`
}
