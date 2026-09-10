package service

import (
	"context"
	"sort"
	"time"

	"duit/internal/domain"
	"duit/internal/store"
)

type ReportService struct {
	txns store.TransactionRepository
}

func NewReportService(txns store.TransactionRepository) *ReportService {
	return &ReportService{txns: txns}
}

type MonthPoint struct {
	Year         int
	Month        time.Month
	IncomeMinor  int64
	ExpenseMinor int64
}

// MonthlyTrend returns exactly n months ending with the month containing
// "now", each derived live from stored transactions — including months with
// zero transactions, which correctly show as zero. This directly fixes a
// bug in the earlier HTML prototype, which showed a hardcoded six-month
// history even when the account had no data at all; here there is nothing
// to hardcode because every point is a real query.
func (s *ReportService) MonthlyTrend(ctx context.Context, now time.Time, n int) ([]MonthPoint, error) {
	points := make([]MonthPoint, n)
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -(n - 1), 0)
	for i := 0; i < n; i++ {
		monthStart := start.AddDate(0, i, 0)
		monthEnd := monthStart.AddDate(0, 1, 0)
		txns, err := s.txns.List(
			ctx,
			store.TransactionFilter{FromDate: monthStart.Format("2006-01-02"), ToDate: monthEnd.Format("2006-01-02")},
		)
		if err != nil {
			return nil, err
		}
		p := MonthPoint{Year: monthStart.Year(), Month: monthStart.Month()}
		for _, t := range txns {
			if t.Category == domain.CategorySavingsTransfer {
				continue
			}
			if t.AmountMinor >= 0 {
				p.IncomeMinor += t.AmountMinor
			} else {
				p.ExpenseMinor += -t.AmountMinor
			}
		}
		points[i] = p
	}
	return points, nil
}

// Forecast projects next month's income/expense as the trailing average —
// a simple, explainable model appropriate for a personal finance app
// (no ML, no external data source to justify anything fancier).
func Forecast(points []MonthPoint) (incomeMinor, expenseMinor int64) {
	if len(points) == 0 {
		return 0, 0
	}
	var incomeSum, expenseSum int64
	for _, p := range points {
		incomeSum += p.IncomeMinor
		expenseSum += p.ExpenseMinor
	}
	return incomeSum / int64(len(points)), expenseSum / int64(len(points))
}

type CategoryTrendRow struct {
	Category       string
	ThisMonthMinor int64
	LastMonthMinor int64
}

// CategoryTrend compares this month's spend per category against last
// month's, both computed live from transactions, sorted by this month's
// spend descending (biggest categories first, matching the mockup).
func (s *ReportService) CategoryTrend(ctx context.Context, now time.Time) ([]CategoryTrendRow, error) {
	thisStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	thisEnd := thisStart.AddDate(0, 1, 0)
	lastStart := thisStart.AddDate(0, -1, 0)

	thisTxns, err := s.txns.List(
		ctx,
		store.TransactionFilter{
			Kind:     "expense",
			FromDate: thisStart.Format("2006-01-02"),
			ToDate:   thisEnd.Format("2006-01-02"),
		},
	)
	if err != nil {
		return nil, err
	}
	lastTxns, err := s.txns.List(
		ctx,
		store.TransactionFilter{
			Kind:     "expense",
			FromDate: lastStart.Format("2006-01-02"),
			ToDate:   thisStart.Format("2006-01-02"),
		},
	)
	if err != nil {
		return nil, err
	}

	thisSums := sumByCategory(thisTxns)
	lastSums := sumByCategory(lastTxns)

	seen := map[string]bool{}
	var order []string
	for cat := range thisSums {
		if !seen[cat] {
			seen[cat] = true
			order = append(order, cat)
		}
	}
	for cat := range lastSums {
		if !seen[cat] {
			seen[cat] = true
			order = append(order, cat)
		}
	}

	out := make([]CategoryTrendRow, 0, len(order))
	for _, cat := range order {
		out = append(out, CategoryTrendRow{Category: cat, ThisMonthMinor: thisSums[cat], LastMonthMinor: lastSums[cat]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ThisMonthMinor > out[j].ThisMonthMinor })
	return out, nil
}

func sumByCategory(txns []domain.Transaction) map[string]int64 {
	out := map[string]int64{}
	for _, t := range txns {
		if t.Category == domain.CategorySavingsTransfer {
			continue
		}
		out[t.Category] += -t.AmountMinor
	}
	return out
}

// SavingsRatePct is (income-expense)/income * 100 for the period. Returns 0
// (not NaN/Inf, which JSON cannot encode) when income is zero or negative.
func SavingsRatePct(incomeMinor, expenseMinor int64) float64 {
	if incomeMinor <= 0 {
		return 0
	}
	return float64(incomeMinor-expenseMinor) / float64(incomeMinor) * 100
}

// RoundUpTotalMinor sums, across every expense transaction, the gap between
// its amount and the next whole currency unit — the "spare change" a
// round-up-to-savings feature would sweep. minorUnitsPerWhole is 100 for a
// currency with 2 decimal places (EUR, USD).
func RoundUpTotalMinor(txns []domain.Transaction, minorUnitsPerWhole int64) int64 {
	var total int64
	for _, t := range txns {
		if t.AmountMinor >= 0 || t.Category == domain.CategorySavingsTransfer {
			continue
		}
		abs := -t.AmountMinor
		remainder := abs % minorUnitsPerWhole
		if remainder > 0 {
			total += minorUnitsPerWhole - remainder
		}
	}
	return total
}
