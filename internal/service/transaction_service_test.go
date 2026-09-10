package service_test

import (
	"context"
	"testing"
	"time"

	"duit/internal/domain"
	"duit/internal/service"
	"duit/internal/store"
)

func TestTransactionService_CreateSplit_SkipsZeroAndFlipsSign(t *testing.T) {
	ctx := context.Background()
	db, _ := store.OpenInMemory()
	defer db.Close()
	acc, _ := store.NewAccountRepository(db).Create(ctx, domain.Account{Name: "Checking", Type: domain.AccountChecking})
	txnRepo := store.NewTransactionRepository(db)
	svc := service.NewTransactionService(txnRepo, store.NewBillRepository(db))

	created, err := svc.CreateSplit(ctx, acc.ID, "Zalando", time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC),
		[]domain.Split{{Category: "Shopping", AmountMinor: 8000}, {Category: "Other", AmountMinor: 0}}, false, "")
	if err != nil {
		t.Fatalf("CreateSplit: %v", err)
	}
	if len(created) != 1 {
		t.Fatalf("expected the zero-amount split to be skipped, got %d rows", len(created))
	}
	if created[0].AmountMinor != -8000 {
		t.Fatalf("expected expense split to be negative, got %d", created[0].AmountMinor)
	}
}

func TestTransactionService_CreateSplit_RejectsAllZero(t *testing.T) {
	ctx := context.Background()
	db, _ := store.OpenInMemory()
	defer db.Close()
	acc, _ := store.NewAccountRepository(db).Create(ctx, domain.Account{Name: "Checking", Type: domain.AccountChecking})
	svc := service.NewTransactionService(store.NewTransactionRepository(db), store.NewBillRepository(db))

	_, err := svc.CreateSplit(
		ctx,
		acc.ID,
		"Payee",
		time.Now(),
		[]domain.Split{{Category: "Other", AmountMinor: 0}},
		false,
		"",
	)
	if err == nil {
		t.Fatal("expected a validation error when every split is zero")
	}
}

func TestTransactionService_DetectRecurring_IgnoresAlreadyTrackedBills(t *testing.T) {
	ctx := context.Background()
	db, _ := store.OpenInMemory()
	defer db.Close()
	acc, _ := store.NewAccountRepository(db).Create(ctx, domain.Account{Name: "Checking", Type: domain.AccountChecking})
	txnRepo := store.NewTransactionRepository(db)
	billRepo := store.NewBillRepository(db)

	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	txnRepo.CreateMany(ctx, []domain.Transaction{
		{AccountID: acc.ID, Date: since, Payee: "Netflix", Category: "Subscriptions", AmountMinor: -1399},
		{
			AccountID:   acc.ID,
			Date:        since.AddDate(0, 1, 0),
			Payee:       "Netflix",
			Category:    "Subscriptions",
			AmountMinor: -1399,
		},
		{AccountID: acc.ID, Date: since, Payee: "Albert Heijn", Category: "Groceries", AmountMinor: -5000},
		{
			AccountID:   acc.ID,
			Date:        since.AddDate(0, 0, 3),
			Payee:       "Albert Heijn",
			Category:    "Groceries",
			AmountMinor: -3000,
		},
	})
	billRepo.Create(
		ctx,
		domain.Bill{
			Name:            "Netflix",
			AmountMinor:     1399,
			CadenceInterval: 1,
			CadenceUnit:     domain.CadenceMonth,
			NextDue:         since,
			AccountID:       acc.ID,
		},
	)

	svc := service.NewTransactionService(txnRepo, billRepo)
	candidate, err := svc.DetectRecurring(ctx, since)
	if err != nil {
		t.Fatalf("DetectRecurring: %v", err)
	}
	if candidate == nil {
		t.Fatal("expected a candidate (Albert Heijn), got nil")
	}
	if candidate.Payee != "Albert Heijn" {
		t.Fatalf("expected Albert Heijn (Netflix is already a tracked bill), got %q", candidate.Payee)
	}
	if candidate.AverageMinor != 4000 {
		t.Fatalf("expected average 4000, got %d", candidate.AverageMinor)
	}
}

func TestTopPayees_SortsDescendingAndExcludesTransfers(t *testing.T) {
	txns := []domain.Transaction{
		{Payee: "A", AmountMinor: -1000, Category: "Other"},
		{Payee: "B", AmountMinor: -5000, Category: "Other"},
		{Payee: "B", AmountMinor: -5000, Category: "Other"},
		{Payee: "Savings", AmountMinor: -20000, Category: domain.CategorySavingsTransfer},
	}
	top := service.TopPayees(txns, 5)
	if len(top) != 2 {
		t.Fatalf("expected 2 payees (transfer excluded), got %d", len(top))
	}
	if top[0].Payee != "B" || top[0].TotalMinor != 10000 || top[0].Count != 2 {
		t.Fatalf("expected B first with total 10000 count 2, got %+v", top[0])
	}
}
