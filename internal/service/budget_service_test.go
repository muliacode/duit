package service_test

import (
	"context"
	"testing"
	"time"

	"duit/internal/domain"
	"duit/internal/service"
	"duit/internal/store"
)

func TestBudgetService_StatusForRange_ExcludesSavingsTransfer(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenInMemory()
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	accRepo := store.NewAccountRepository(db)
	acc, _ := accRepo.Create(ctx, domain.Account{Name: "Checking", Type: domain.AccountChecking})

	budgetRepo := store.NewBudgetRepository(db)
	if _, err := budgetRepo.Create(ctx, domain.BudgetCategory{Name: "Groceries", AllocatedMinor: 40000}); err != nil {
		t.Fatalf("create budget: %v", err)
	}

	txnRepo := store.NewTransactionRepository(db)
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)
	if _, err := txnRepo.CreateMany(ctx, []domain.Transaction{
		{
			AccountID:   acc.ID,
			Date:        from.AddDate(0, 0, 1),
			Payee:       "Albert Heijn",
			Category:    "Groceries",
			AmountMinor: -8640,
		},
		{
			AccountID:   acc.ID,
			Date:        from.AddDate(0, 0, 2),
			Payee:       "Transfer",
			Category:    domain.CategorySavingsTransfer,
			AmountMinor: -50000,
		},
	}); err != nil {
		t.Fatalf("seed transactions: %v", err)
	}

	svc := service.NewBudgetService(budgetRepo, txnRepo)
	statuses, err := svc.StatusForRange(ctx, from, to)
	if err != nil {
		t.Fatalf("StatusForRange: %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("expected 1 category, got %d", len(statuses))
	}
	if statuses[0].SpentMinor != 8640 {
		t.Fatalf("expected spend 8640 (savings transfer excluded), got %d", statuses[0].SpentMinor)
	}
	if statuses[0].PercentUsed < 21 || statuses[0].PercentUsed > 22 {
		t.Fatalf("expected ~21.6%%, got %.2f", statuses[0].PercentUsed)
	}
}

func TestBudgetService_AdjustAllocation_FloorsAtZero(t *testing.T) {
	ctx := context.Background()
	db, _ := store.OpenInMemory()
	defer db.Close()
	budgetRepo := store.NewBudgetRepository(db)
	budgetRepo.Create(ctx, domain.BudgetCategory{Name: "Fun", AllocatedMinor: 500})

	svc := service.NewBudgetService(budgetRepo, store.NewTransactionRepository(db))
	if err := svc.AdjustAllocation(ctx, "Fun", -10000); err != nil {
		t.Fatalf("AdjustAllocation: %v", err)
	}
	cats, _ := budgetRepo.List(ctx)
	if cats[0].AllocatedMinor != 0 {
		t.Fatalf("expected allocation floored at 0, got %d", cats[0].AllocatedMinor)
	}
}

func TestUnassignedMinor(t *testing.T) {
	cats := []domain.BudgetCategory{{AllocatedMinor: 40000}, {AllocatedMinor: 7000}}
	if got := service.UnassignedMinor(365000, cats); got != 318000 {
		t.Fatalf("expected 318000 unassigned, got %d", got)
	}
	if got := service.UnassignedMinor(10000, cats); got != -37000 {
		t.Fatalf("expected negative (over-assigned) result of -37000, got %d", got)
	}
}
