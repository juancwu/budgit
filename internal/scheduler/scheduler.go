package scheduler

import (
	"context"
	"log/slog"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/service"
)

type Scheduler struct {
	recurringService *service.RecurringExpenseService
	interval         time.Duration
}

func New(recurringService *service.RecurringExpenseService) *Scheduler {
	return &Scheduler{
		recurringService: recurringService,
		interval:         1 * time.Hour,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	// Run immediately on startup to catch up missed recurrences
	s.run()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("scheduler stopping")
			return
		case <-ticker.C:
			s.run()
		}
	}
}

func (s *Scheduler) run() {
	slog.Info("scheduler: processing due recurring expenses")
	now := time.Now()
	if err := s.recurringService.ProcessDueRecurrences(now); err != nil {
		slog.Error("scheduler: failed to process recurring expenses", "error", err)
	}
}
