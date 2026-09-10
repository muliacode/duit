package service_test

import (
	"context"
	"testing"
	"time"

	"duit/internal/domain"
	"duit/internal/service"
	"duit/internal/store"
)

func TestReportService_MonthlyTrend_ZeroForMonthsWithNoData(t *testing.T) {
	ctx := context.Background()
	db, _ := store.OpenInMemory()
	defer db.Close()
	accRepo := store.NewAccountRepository(db)
	acc, _ := accRepo.Create(ctx, domain.Account{Name: "Checking", Type: domain.AccountChecking})
	txnRepo := store.NewTransactionRepository(db)
	txnRepo.CreateMany(ctx, []domain.Transaction{
		{
			AccountID:   acc.ID,
			Date:        time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
			Payee:       "Employer",
			Category:    "Salary",
			AmountMinor: 320000,
		},
		{
			AccountID:   acc.ID,
			Date:        time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC),
			Payee:       "Shop",
			Category:    "Groceries",
			AmountMinor: -8000,
		},
	})

	svc := service.NewReportService(txnRepo)
	points, err := svc.MonthlyTrend(ctx, time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC), 6)
	if err != nil {
		t.Fatalf("MonthlyTrend: %v", err)
	}
	if len(points) != 6 {
		t.Fatalf("expected 6 points, got %d", len(points))
	}
	// This is the exact bug the earlier HTML prototype had: months with zero
	// transactions must show as zero, never as fabricated demo numbers.
	for i := 0; i < 5; i++ {
		if points[i].IncomeMinor != 0 || points[i].ExpenseMinor != 0 {
			t.Fatalf("expected month %d (no data) to be zero, got %+v", i, points[i])
		}
	}
	last := points[5]
	if last.IncomeMinor != 320000 || last.ExpenseMinor != 8000 {
		t.Fatalf("expected September to reflect seeded transactions, got %+v", last)
	}
}

func TestForecast_AveragesTrend(t *testing.T) {
	points := []service.MonthPoint{{IncomeMinor: 100, ExpenseMinor: 40}, {IncomeMinor: 200, ExpenseMinor: 60}}
	income, expense := service.Forecast(points)
	if income != 150 || expense != 50 {
		t.Fatalf("expected averages (150, 50), got (%d, %d)", income, expense)
	}
}

func TestSavingsRatePct(t *testing.T) {
	if got := service.SavingsRatePct(0, 100); got != 0 {
		t.Fatalf("expected 0 for zero income (no divide-by-zero), got %.2f", got)
	}
	if got := service.SavingsRatePct(1000, 800); got < 19.9 || got > 20.1 {
		t.Fatalf("expected ~20%%, got %.2f", got)
	}
}

func TestRoundUpTotalMinor(t *testing.T) {
	txns := []domain.Transaction{
		{AmountMinor: -8640, Category: "Groceries"},                    // 86.40 -> rounds up 0.60 -> 60 minor
		{AmountMinor: -1399, Category: "Subscriptions"},                // 13.99 -> rounds up 0.01 -> 1 minor
		{AmountMinor: -5000, Category: domain.CategorySavingsTransfer}, // excluded
		{AmountMinor: 320000, Category: "Salary"},                      // income, excluded
	}
	if got := service.RoundUpTotalMinor(txns, 100); got != 61 {
		t.Fatalf("expected round-up total 61 minor units, got %d", got)
	}
}
