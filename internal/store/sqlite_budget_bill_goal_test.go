package store_test

import (
	"context"
	"testing"

	"duit/internal/domain"
	"duit/internal/store"
)

func TestBudgetRepository(t *testing.T) {
	ctx := context.Background()
	repo := store.NewBudgetRepository(newTestDB(t))

	c, err := repo.Create(ctx, domain.BudgetCategory{Name: "Groceries", AllocatedMinor: 40000})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := repo.UpdateAllocated(ctx, c.Name, 45000); err != nil {
		t.Fatalf("UpdateAllocated: %v", err)
	}
	list, _ := repo.List(ctx)
	if len(list) != 1 || list[0].AllocatedMinor != 45000 {
		t.Fatalf("expected updated allocation 45000, got %+v", list)
	}
	if err := repo.Delete(ctx, c.Name); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestBillRepository(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	acc := mustAccount(t, db)
	repo := store.NewBillRepository(db)

	b, err := repo.Create(ctx, domain.Bill{
		Name: "Netflix", AmountMinor: 1399, CadenceInterval: 1, CadenceUnit: domain.CadenceMonth,
		NextDue: date(2026, 9, 10), AccountID: acc.ID,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if b.Status != domain.BillUpcoming {
		t.Fatalf("expected default status upcoming, got %s", b.Status)
	}
	b.Status = domain.BillPaid
	if err := repo.Update(ctx, b); err != nil {
		t.Fatalf("Update: %v", err)
	}
	list, _ := repo.List(ctx)
	if len(list) != 1 || list[0].Status != domain.BillPaid {
		t.Fatalf("expected status paid after update, got %+v", list)
	}
}

func TestGoalRepository(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	acc := mustAccount(t, db)
	repo := store.NewGoalRepository(db)

	g, err := repo.Create(
		ctx,
		domain.Goal{Name: "Emergency fund", TargetMinor: 1500000, SavedMinor: 1250000, LinkedAccountID: acc.ID},
	)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if pct := g.ProgressPct(); pct < 83 || pct > 84 {
		t.Fatalf("expected ~83.3%%, got %.2f", pct)
	}
	list, err := repo.List(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("List: %v, %+v", err, list)
	}
	if list[0].LinkedAccountID != acc.ID {
		t.Fatalf("expected linked account to round-trip, got %q", list[0].LinkedAccountID)
	}
}

func TestSettingsRepository_DefaultsAndUpdate(t *testing.T) {
	ctx := context.Background()
	repo := store.NewSettingsRepository(newTestDB(t))

	s, err := repo.Get(ctx)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if s.Currency != "EUR" || s.BudgetMethod != domain.BudgetSimple {
		t.Fatalf("unexpected defaults: %+v", s)
	}

	s.Currency = "USD"
	s.HideAmounts = true
	if err := repo.Update(ctx, s); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := repo.Get(ctx)
	if got.Currency != "USD" || !got.HideAmounts {
		t.Fatalf("expected updated settings to persist, got %+v", got)
	}
}
