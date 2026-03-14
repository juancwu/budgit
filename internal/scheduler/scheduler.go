package scheduler

import (
	"context"
	"log/slog"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/service"
)

type Scheduler struct {
	recurringService        *service.RecurringExpenseService
	recurringReceiptService *service.RecurringReceiptService
	interval                time.Duration
}

func New(recurringService *service.RecurringExpenseService, recurringReceiptService *service.RecurringReceiptService) *Scheduler {
	return &Scheduler{
		recurringService:        recurringService,
		recurringReceiptService: recurringReceiptService,
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

	slog.Info("scheduler: processing due recurring receipts")
	if err := s.recurringReceiptService.ProcessDueRecurrences(now); err != nil {
		slog.Error("scheduler: failed to process recurring receipts", "error", err)
	}
}
