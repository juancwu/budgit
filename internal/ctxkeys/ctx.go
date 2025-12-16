package ctxkeys

import (
	"context"

	"git.juancwu.dev/juancwu/budgething/internal/model"
)

const (
	UserKey string = "user"
)

func User(ctx context.Context) *model.User {
	user, _ := ctx.Value(UserKey).(*model.User)
	return user
}
