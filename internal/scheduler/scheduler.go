package scheduler

import (
	"context"
	"log/slog"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/service"
)

type Scheduler struct {
	recurringService        *service.RecurringExpenseService
	recurringDepositService *service.RecurringDepositService
	interval                time.Duration
}

func New(recurringService *service.RecurringExpenseService, recurringDepositService *service.RecurringDepositService) *Scheduler {
	return &Scheduler{
		recurringService:        recurringService,
		recurringDepositService: recurringDepositService,
		interval:                1 * time.Hour,
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
	now := time.Now()

	slog.Info("scheduler: processing due recurring expenses")
	if err := s.recurringService.ProcessDueRecurrences(now); err != nil {
		slog.Error("scheduler: failed to process recurring expenses", "error", err)
	}

	slog.Info("scheduler: processing due recurring deposits")
	if err := s.recurringDepositService.ProcessDueRecurrences(now); err != nil {
		slog.Error("scheduler: failed to process recurring deposits", "error", err)
	}
}
