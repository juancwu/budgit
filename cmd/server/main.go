package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/app"
	"git.juancwu.dev/juancwu/budgit/internal/config"
	"git.juancwu.dev/juancwu/budgit/internal/routes"
)

// version is set at build time via -ldflags.
var version = "dev"

func main() {
	cfg := config.Load(version)

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

	// Health check bypasses all middleware
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/healthz" {
			if err := a.DB.Ping(); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte("db: unreachable"))
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok" + " - version: " + version))
			return
		}
		handler.ServeHTTP(w, r)
	})

	srv := &http.Server{
		Addr:    cfg.Host + ":" + cfg.Port,
		Handler: finalHandler,
	}

	workerCtx, stopWorker := context.WithCancel(context.Background())
	defer stopWorker()
	go runRecurringWorker(workerCtx, a)

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
		<-sigCh
		slog.Info("shutting down gracefully")
		stopWorker()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	slog.Info("server starting", "version", version, "host", cfg.Host, "port", cfg.Port, "env", cfg.AppEnv, "url", fmt.Sprintf("http://%s:%s", cfg.Host, cfg.Port))

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server failed", "error", err)
		panic(err)
	}
}

// runRecurringWorker materializes due recurring events on a fixed cadence. It
// fires once at startup (catching up anything missed while the server was
// down), then ticks every minute until ctx is cancelled.
func runRecurringWorker(ctx context.Context, a *app.App) {
	tick := func() {
		if err := a.RecurringEventService.ProcessDue(time.Now().UTC()); err != nil {
			slog.Error("recurring event processing failed", "error", err)
		}
	}
	tick()
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			tick()
		}
	}
}
