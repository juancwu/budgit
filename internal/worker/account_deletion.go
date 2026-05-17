package worker

import (
	"context"
	"log/slog"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/service"
)

// AccountDeletionWorker periodically drains the account deletion queue. It
// also exposes a trigger channel so the HTTP handler can wake the worker
// immediately after enqueueing a new request. On startup the worker runs one
// pass synchronously so requests that were in-flight when the server went
// down are resumed before the first new request arrives.
type AccountDeletionWorker struct {
	userService *service.UserService
	interval    time.Duration
	trigger     chan struct{}
}

func NewAccountDeletionWorker(userService *service.UserService, interval time.Duration) *AccountDeletionWorker {
	w := &AccountDeletionWorker{
		userService: userService,
		interval:    interval,
		trigger:     make(chan struct{}, 1),
	}
	userService.SetDeletionTrigger(w.trigger)
	return w
}

// Start runs an initial pass to resume any work in-flight from a previous
// boot, then loops until ctx is cancelled, processing whenever the ticker
// fires or a trigger arrives.
func (w *AccountDeletionWorker) Start(ctx context.Context) {
	slog.Info("account deletion worker starting", "interval", w.interval)

	// Resume work from before the last restart.
	if n := w.userService.ProcessPendingDeletions(); n > 0 {
		slog.Info("account deletion worker resumed pending work on startup", "processed", n)
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("account deletion worker shutting down")
			return
		case <-ticker.C:
			w.userService.ProcessPendingDeletions()
		case <-w.trigger:
			w.userService.ProcessPendingDeletions()
		}
	}
}
