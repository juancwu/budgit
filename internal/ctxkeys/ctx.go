package ctxkeys

import (
	"context"

	"git.juancwu.dev/juancwu/budgit/internal/config"
	"git.juancwu.dev/juancwu/budgit/internal/model"
)

const (
	UserKey      string = "user"
	ProfileKey   string = "profile"
	URLPathKey   string = "url_path"
	ConfigKey    string = "config"
	CSRFTokenKey string = "csrf_token"
)

func User(ctx context.Context) *model.User {
	user, _ := ctx.Value(UserKey).(*model.User)
	return user
}

func WithUser(ctx context.Context, user *model.User) context.Context {
	return context.WithValue(ctx, UserKey, user)
}

func Profile(ctx context.Context) *model.Profile {
	profile, _ := ctx.Value(ProfileKey).(*model.Profile)
	return profile
}

func WithProfile(ctx context.Context, profile *model.Profile) context.Context {
	return context.WithValue(ctx, ProfileKey, profile)
}

func URLPath(ctx context.Context) string {
	path, _ := ctx.Value(URLPathKey).(string)
	return path
}

func WithURLPath(ctx context.Context, urlPath string) context.Context {
	return context.WithValue(ctx, URLPathKey, urlPath)
}

func Config(ctx context.Context) *config.Config {
	cfg, _ := ctx.Value(ConfigKey).(*config.Config)
	return cfg
}

func WithConfig(ctx context.Context, cfg *config.Config) context.Context {
	return context.WithValue(ctx, ConfigKey, cfg)
}

func CSRFToken(ctx context.Context) string {
	token, _ := ctx.Value(CSRFTokenKey).(string)
	return token
}

func WithCSRFToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, CSRFTokenKey, token)
}
