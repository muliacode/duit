package service

import (
	"context"
	"time"

	"duit/internal/domain"
	"duit/internal/store"
)

type BudgetService struct {
	repo store.BudgetRepository
	txns store.TransactionRepository
}

func NewBudgetService(repo store.BudgetRepository, txns store.TransactionRepository) *BudgetService {
	return &BudgetService{repo: repo, txns: txns}
}

// CategoryStatus joins a budget category with its month-to-date spend —
// the repository only stores the allocation, never a cached "spent" value,
// so it can never go stale relative to the transaction log.
type CategoryStatus struct {
	domain.BudgetCategory
	SpentMinor  int64
	PercentUsed float64 // may exceed 100 when over budget
}

// StatusForRange returns every category with spend computed over
// [from, to). Passing the current month gives "this month's budgets";
// passing a past month re-derives that month's status on demand — nothing
// is ever pre-aggregated and cached, so there is nothing that can drift.
func (s *BudgetService) StatusForRange(ctx context.Context, from, to time.Time) ([]CategoryStatus, error) {
	cats, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	spent, err := s.spendByCategory(ctx, from, to)
	if err != nil {
		return nil, err
	}
	out := make([]CategoryStatus, 0, len(cats))
	for _, c := range cats {
		sp := spent[c.Name]
		pct := 0.0
		if c.AllocatedMinor > 0 {
			pct = float64(sp) / float64(c.AllocatedMinor) * 100
		}
		out = append(out, CategoryStatus{BudgetCategory: c, SpentMinor: sp, PercentUsed: pct})
	}
	return out, nil
}

func (s *BudgetService) spendByCategory(ctx context.Context, from, to time.Time) (map[string]int64, error) {
	txns, err := s.txns.List(
		ctx,
		store.TransactionFilter{Kind: "expense", FromDate: from.Format("2006-01-02"), ToDate: to.Format("2006-01-02")},
	)
	if err != nil {
		return nil, err
	}
	out := map[string]int64{}
	for _, t := range txns {
		if t.Category == domain.CategorySavingsTransfer {
			continue
		}
		out[t.Category] += -t.AmountMinor
	}
	return out, nil
}

func (s *BudgetService) Create(ctx context.Context, c domain.BudgetCategory) (domain.BudgetCategory, error) {
	if err := c.Validate(); err != nil {
		return domain.BudgetCategory{}, err
	}
	return s.repo.Create(ctx, c)
}

func (s *BudgetService) Delete(ctx context.Context, name string) error {
	return s.repo.Delete(ctx, name)
}

// AdjustAllocation applies a signed delta (used by the Envelope view's +/-
// controls) and floors at zero so an envelope can never go negative.
func (s *BudgetService) AdjustAllocation(ctx context.Context, name string, deltaMinor int64) error {
	cats, err := s.repo.List(ctx)
	if err != nil {
		return err
	}
	for _, c := range cats {
		if c.Name == name {
			next := c.AllocatedMinor + deltaMinor
			if next < 0 {
				next = 0
			}
			return s.repo.UpdateAllocated(ctx, name, next)
		}
	}
	return domain.ErrNotFound
}

// UnassignedMinor is the envelope-method "ready to assign" figure: income
// not yet allocated to any envelope. Negative means over-assigned.
func UnassignedMinor(incomeMinor int64, categories []domain.BudgetCategory) int64 {
	var allocated int64
	for _, c := range categories {
		allocated += c.AllocatedMinor
	}
	return incomeMinor - allocated
}
