package service

import "duit/internal/domain"

// DebtService is pure calculation — no repository, no I/O — which is why
// its tests (debt_service_test.go) need no database at all. It intentionally
// does not know about accounts or persistence: callers pass in the balance,
// APR, and payment figures already read from an Account.
type DebtService struct{}

func NewDebtService() *DebtService { return &DebtService{} }

// PayoffMonths simulates a standard amortized payoff: each month, interest
// accrues at aprBps/10000/12, then (minPaymentMinor+extraMinor) is applied.
// Returns -1 if the payment never covers the interest (balance would grow
// forever) — callers should render that as "never at this rate", not 0.
// Capped at 600 months (50 years) as a hard safety bound.
func (DebtService) PayoffMonths(balanceMinor int64, aprBps int, minPaymentMinor, extraMinor int64) int {
	if balanceMinor <= 0 {
		return 0
	}
	monthlyRate := float64(aprBps) / 10000.0 / 12.0
	payment := float64(minPaymentMinor + extraMinor)
	bal := float64(balanceMinor)

	if payment <= bal*monthlyRate {
		return -1
	}
	months := 0
	for bal > 0 && months < 600 {
		interest := bal * monthlyRate
		p := payment
		if p > bal+interest {
			p = bal + interest
		}
		bal = bal + interest - p
		months++
	}
	return months
}

// RegularPayment computes the standard fixed monthly payment for a fully
// amortizing loan of the given original principal, APR, and term — used to
// derive a mortgage's "regular payment" when the account doesn't store one
// explicitly.
func (DebtService) RegularPayment(originalMinor int64, aprBps, termMonths int) int64 {
	if termMonths <= 0 || originalMinor <= 0 {
		return 0
	}
	r := float64(aprBps) / 10000.0 / 12.0
	if r == 0 {
		return originalMinor / int64(termMonths)
	}
	p := float64(originalMinor) * r / (1 - powNeg(1+r, termMonths))
	return int64(p)
}

func powNeg(base float64, exp int) float64 {
	// (1+r)^-n computed as 1 / (1+r)^n to avoid math.Pow's negative-exponent
	// edge cases with a tiny loop — term lengths are at most a few hundred months.
	result := 1.0
	for i := 0; i < exp; i++ {
		result *= base
	}
	return 1 / result
}

type DebtStrategyAllocation struct {
	AccountID  string
	ExtraMinor int64
}

// AllocateExtra splits a pool of extra payment across debts per strategy:
//   - avalanche: all of it to the highest-APR debt
//   - snowball:  all of it to the smallest-balance debt
//   - equal:     split evenly
//   - custom:    caller-supplied per-debt amounts (passed through as-is)
//
// This only decides THIS month's allocation (matching the mockup's
// behavior) — it does not simulate redirecting the extra to the next debt
// once the first is paid off. See the README for that known simplification.
func (DebtService) AllocateExtra(
	strategy domain.DebtStrategy,
	debts []domain.Account,
	extraPoolMinor int64,
	custom map[string]int64,
) []DebtStrategyAllocation {
	if len(debts) == 0 {
		return nil
	}
	switch strategy {
	case domain.StrategyCustom:
		out := make([]DebtStrategyAllocation, 0, len(debts))
		for _, d := range debts {
			out = append(out, DebtStrategyAllocation{AccountID: d.ID, ExtraMinor: custom[d.ID]})
		}
		return out
	case domain.StrategyEqual:
		share := extraPoolMinor / int64(len(debts))
		out := make([]DebtStrategyAllocation, 0, len(debts))
		for _, d := range debts {
			out = append(out, DebtStrategyAllocation{AccountID: d.ID, ExtraMinor: share})
		}
		return out
	case domain.StrategySnowball:
		target := debts[0]
		for _, d := range debts {
			if d.BalanceMinor < target.BalanceMinor {
				target = d
			}
		}
		return allocateAllTo(debts, target.ID, extraPoolMinor)
	default: // avalanche
		target := debts[0]
		for _, d := range debts {
			if rate(d) > rate(target) {
				target = d
			}
		}
		return allocateAllTo(debts, target.ID, extraPoolMinor)
	}
}

func rate(a domain.Account) int {
	if a.InterestRateBps == nil {
		return 0
	}
	return *a.InterestRateBps
}

func allocateAllTo(debts []domain.Account, id string, amount int64) []DebtStrategyAllocation {
	out := make([]DebtStrategyAllocation, 0, len(debts))
	for _, d := range debts {
		extra := int64(0)
		if d.ID == id {
			extra = amount
		}
		out = append(out, DebtStrategyAllocation{AccountID: d.ID, ExtraMinor: extra})
	}
	return out
}
