package store_test

import (
	"context"
	"testing"
	"time"

	"duit/internal/domain"
	"duit/internal/store"
)

func mustAccount(t *testing.T, db *store.DB) domain.Account {
	t.Helper()
	a, err := store.NewAccountRepository(db).
		Create(context.Background(), domain.Account{Name: "Checking", Type: domain.AccountChecking})
	if err != nil {
		t.Fatalf("create account fixture: %v", err)
	}
	return a
}

func TestTransactionRepository_FilterByKindAndCategory(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	acc := mustAccount(t, db)
	repo := store.NewTransactionRepository(db)

	seed := []domain.Transaction{
		{AccountID: acc.ID, Date: date(2026, 9, 1), Payee: "Employer", Category: "Salary", AmountMinor: 320000},
		{AccountID: acc.ID, Date: date(2026, 9, 2), Payee: "Albert Heijn", Category: "Groceries", AmountMinor: -8640},
		{AccountID: acc.ID, Date: date(2026, 9, 3), Payee: "Netflix", Category: "Subscriptions", AmountMinor: -1399},
	}
	if _, err := repo.CreateMany(ctx, seed); err != nil {
		t.Fatalf("CreateMany: %v", err)
	}

	expenses, err := repo.List(ctx, store.TransactionFilter{Kind: "expense"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(expenses) != 2 {
		t.Fatalf("expected 2 expenses, got %d", len(expenses))
	}

	groceries, err := repo.List(ctx, store.TransactionFilter{Category: "Groceries"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(groceries) != 1 || groceries[0].Payee != "Albert Heijn" {
		t.Fatalf("expected 1 groceries txn from Albert Heijn, got %+v", groceries)
	}

	matched, err := repo.List(ctx, store.TransactionFilter{Query: "netflix"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(matched) != 1 {
		t.Fatalf("expected case-insensitive-ish substring match to find Netflix, got %d results", len(matched))
	}
}

func TestTransactionRepository_DeleteMany(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	acc := mustAccount(t, db)
	repo := store.NewTransactionRepository(db)

	created, err := repo.CreateMany(ctx, []domain.Transaction{
		{AccountID: acc.ID, Date: date(2026, 9, 1), Payee: "A", Category: "Other", AmountMinor: -100},
		{AccountID: acc.ID, Date: date(2026, 9, 2), Payee: "B", Category: "Other", AmountMinor: -200},
	})
	if err != nil {
		t.Fatalf("CreateMany: %v", err)
	}

	if err := repo.DeleteMany(ctx, []string{created[0].ID, created[1].ID}); err != nil {
		t.Fatalf("DeleteMany: %v", err)
	}
	remaining, _ := repo.List(ctx, store.TransactionFilter{})
	if len(remaining) != 0 {
		t.Fatalf("expected 0 remaining transactions, got %d", len(remaining))
	}
}

func date(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }
