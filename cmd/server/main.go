package main

import (
	"fmt"
	"log/slog"
	"net/http"

	"git.juancwu.dev/juancwu/budgit/internal/app"
	"git.juancwu.dev/juancwu/budgit/internal/config"
	"git.juancwu.dev/juancwu/budgit/internal/routes"
)

func main() {
	cfg := config.Load()

	a, err := app.New(cfg)
	if err != nil {
		slog.Error("failed to initialize app", "error", err)
		panic(err)
	}
	defer func() {
		err := a.Close()
		if err != nil {
			slog.Error("failed to close app", "error", err)
		}
	}()

	handler := routes.SetupRoutes(a)
	slog.Info("server starting", "host", cfg.Host, "port", cfg.Port, "env", cfg.AppEnv, "url", fmt.Sprintf("http://%s:%s", cfg.Host, cfg.Port))

	err = http.ListenAndServe(":"+cfg.Port, handler)
	if err != nil {
		slog.Error("server failed", "error", err)
		panic(err)
	}
}
