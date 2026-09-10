package service_test

import (
	"testing"

	"duit/internal/domain"
	"duit/internal/service"
)

func TestPayoffMonths_ZeroBalance(t *testing.T) {
	svc := service.DebtService{}
	if got := svc.PayoffMonths(0, 1990, 2500, 0); got != 0 {
		t.Fatalf("expected 0 months for zero balance, got %d", got)
	}
}

func TestPayoffMonths_NeverPaysOff(t *testing.T) {
	svc := service.DebtService{}
	// 19.9% APR on 100000 minor units, payment smaller than the interest accrued
	got := svc.PayoffMonths(100000, 1990, 100, 0)
	if got != -1 {
		t.Fatalf("expected -1 (never pays off) when payment doesn't cover interest, got %d", got)
	}
}

func TestPayoffMonths_ExtraPaymentShortensPayoff(t *testing.T) {
	svc := service.DebtService{}
	base := svc.PayoffMonths(820000, 520, 22000, 0) // car loan-ish figures
	withExtra := svc.PayoffMonths(820000, 520, 22000, 15000)
	if base == -1 || withExtra == -1 {
		t.Fatalf("expected both to pay off, got base=%d withExtra=%d", base, withExtra)
	}
	if withExtra >= base {
		t.Fatalf("expected extra payment to shorten payoff: base=%d withExtra=%d", base, withExtra)
	}
}

func TestRegularPayment_MatchesKnownAmortization(t *testing.T) {
	svc := service.DebtService{}
	// 255,000.00 principal, 3.80% APR, 360 months — a real mortgage payment
	// for these figures is approximately 1,186.00/month.
	payment := svc.RegularPayment(25500000, 380, 360)
	if payment < 118000 || payment > 119500 {
		t.Fatalf("expected regular payment near 118,600-119,000 minor units, got %d", payment)
	}
}

func TestAllocateExtra_Avalanche_PicksHighestRate(t *testing.T) {
	svc := service.DebtService{}
	rateA, rateB := 1990, 520
	debts := []domain.Account{
		{ID: "cc", BalanceMinor: 41230, InterestRateBps: &rateA},
		{ID: "car", BalanceMinor: 820000, InterestRateBps: &rateB},
	}
	alloc := svc.AllocateExtra(domain.StrategyAvalanche, debts, 15000, nil)
	byID := map[string]int64{}
	for _, a := range alloc {
		byID[a.AccountID] = a.ExtraMinor
	}
	if byID["cc"] != 15000 || byID["car"] != 0 {
		t.Fatalf("expected avalanche to send all extra to the highest-APR debt (cc), got %+v", byID)
	}
}

func TestAllocateExtra_Snowball_PicksSmallestBalance(t *testing.T) {
	svc := service.DebtService{}
	rateA, rateB := 1990, 520
	debts := []domain.Account{
		{ID: "cc", BalanceMinor: 41230, InterestRateBps: &rateA},
		{ID: "car", BalanceMinor: 820000, InterestRateBps: &rateB},
	}
	alloc := svc.AllocateExtra(domain.StrategySnowball, debts, 15000, nil)
	byID := map[string]int64{}
	for _, a := range alloc {
		byID[a.AccountID] = a.ExtraMinor
	}
	if byID["cc"] != 15000 || byID["car"] != 0 {
		t.Fatalf("expected snowball to send all extra to the smallest-balance debt (cc), got %+v", byID)
	}
}

func TestAllocateExtra_Equal_Splits(t *testing.T) {
	svc := service.DebtService{}
	debts := []domain.Account{{ID: "a"}, {ID: "b"}}
	alloc := svc.AllocateExtra(domain.StrategyEqual, debts, 10000, nil)
	for _, a := range alloc {
		if a.ExtraMinor != 5000 {
			t.Fatalf("expected an even 50/50 split, got %+v", alloc)
		}
	}
}

func TestAllocateExtra_Custom_PassesThrough(t *testing.T) {
	svc := service.DebtService{}
	debts := []domain.Account{{ID: "a"}, {ID: "b"}}
	custom := map[string]int64{"a": 3000, "b": 7000}
	alloc := svc.AllocateExtra(domain.StrategyCustom, debts, 0, custom)
	byID := map[string]int64{}
	for _, a := range alloc {
		byID[a.AccountID] = a.ExtraMinor
	}
	if byID["a"] != 3000 || byID["b"] != 7000 {
		t.Fatalf("expected custom amounts to pass through unchanged, got %+v", byID)
	}
}
