package app

import (
	"context"
	"fmt"

	"ledger/internal/cache"
	"ledger/internal/db"
	"ledger/internal/repository/pg"
	"ledger/internal/service"
)

// NewService инициализирует все инфраструктурные зависимости и собирает сервис.
func NewService(ctx context.Context) (service.Service, func() error, error) {
	database, err := db.New(ctx, db.Config{})
	if err != nil {
		return nil, nil, err
	}

	cacheClient, err := cache.New(ctx)
	if err != nil {
		_ = db.Close(database)
		return nil, nil, err
	}

	budgetRepo := pg.NewBudgetRepository(database)
	expenseRepo := pg.NewExpenseRepository(database)

	svc := service.New(budgetRepo, expenseRepo, cacheClient, nil)

	closeFn := func() error {
		var closingErr error
		if err := cacheClient.Close(); err != nil {
			closingErr = fmt.Errorf("close redis: %w", err)
		}
		if err := db.Close(database); err != nil {
			if closingErr != nil {
				closingErr = fmt.Errorf("%v; close db: %w", closingErr, err)
			} else {
				closingErr = fmt.Errorf("close db: %w", err)
			}
		}
		return closingErr
	}

	return svc, closeFn, nil
}

