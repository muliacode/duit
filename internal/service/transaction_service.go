package service

import (
	"context"
	"strings"
	"time"

	"duit/internal/domain"
	"duit/internal/store"
)

type TransactionService struct {
	repo  store.TransactionRepository
	bills store.BillRepository
}

func NewTransactionService(repo store.TransactionRepository, bills store.BillRepository) *TransactionService {
	return &TransactionService{repo: repo, bills: bills}
}

func (s *TransactionService) List(ctx context.Context, f store.TransactionFilter) ([]domain.Transaction, error) {
	return s.repo.List(ctx, f)
}

func (s *TransactionService) Create(ctx context.Context, t domain.Transaction) (domain.Transaction, error) {
	if err := t.Validate(); err != nil {
		return domain.Transaction{}, err
	}
	return s.repo.Create(ctx, t)
}

// CreateSplit writes one row per split, all sharing date/payee/account/note —
// this is the only code path for both "split into multiple categories" and
// (trivially, with one split) a plain single-category transaction, so the
// two never drift into different validation or persistence behavior.
func (s *TransactionService) CreateSplit(
	ctx context.Context,
	accountID, payee string,
	date time.Time,
	splits []domain.Split,
	isIncome bool,
	note string,
) ([]domain.Transaction, error) {
	if accountID == "" {
		return nil, domain.NewValidationError("accountId", "Pick an account.")
	}
	txns := make([]domain.Transaction, 0, len(splits))
	for _, sp := range splits {
		if sp.AmountMinor <= 0 {
			continue
		}
		amt := sp.AmountMinor
		if !isIncome {
			amt = -amt
		}
		txns = append(
			txns,
			domain.Transaction{
				AccountID:   accountID,
				Date:        date,
				Payee:       payee,
				Category:    sp.Category,
				AmountMinor: amt,
				Note:        note,
			},
		)
	}
	if len(txns) == 0 {
		return nil, domain.NewValidationError("splits", "Enter at least one split amount greater than zero.")
	}
	return s.repo.CreateMany(ctx, txns)
}

func (s *TransactionService) Update(ctx context.Context, t domain.Transaction) error {
	if err := t.Validate(); err != nil {
		return err
	}
	return s.repo.Update(ctx, t)
}

func (s *TransactionService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *TransactionService) DeleteMany(ctx context.Context, ids []string) error {
	return s.repo.DeleteMany(ctx, ids)
}

// RecurringCandidate is a payee whose spending pattern looks like a bill
// that hasn't been tracked as one yet.
type RecurringCandidate struct {
	Payee        string
	Occurrences  int
	AverageMinor int64
}

// DetectRecurring flags the first payee (by transaction order) with 2+
// expense transactions since "since" that isn't already tracked as a bill.
// Returns (nil, nil) when nothing qualifies — callers should treat that as
// "no suggestion", not an error.
func (s *TransactionService) DetectRecurring(ctx context.Context, since time.Time) (*RecurringCandidate, error) {
	txns, err := s.repo.List(ctx, store.TransactionFilter{Kind: "expense", FromDate: since.Format("2006-01-02")})
	if err != nil {
		return nil, err
	}
	bills, err := s.bills.List(ctx)
	if err != nil {
		return nil, err
	}
	billNames := make(map[string]bool, len(bills))
	for _, b := range bills {
		billNames[strings.ToLower(b.Name)] = true
	}

	counts := map[string]int{}
	sums := map[string]int64{}
	var order []string
	for _, t := range txns {
		if counts[t.Payee] == 0 {
			order = append(order, t.Payee)
		}
		counts[t.Payee]++
		sums[t.Payee] += -t.AmountMinor
	}
	for _, payee := range order {
		if counts[payee] >= 2 && !billNames[strings.ToLower(payee)] {
			return &RecurringCandidate{
				Payee:        payee,
				Occurrences:  counts[payee],
				AverageMinor: sums[payee] / int64(counts[payee]),
			}, nil
		}
	}
	return nil, nil
}

// PayeeSummary is one row of the "top payees" report: total expense spend
// and how many transactions made it up.
type PayeeSummary struct {
	Payee      string
	Count      int
	TotalMinor int64
}

// TopPayees returns the top n payees by total expense in the given
// transaction set, excluding internal transfers — used by the Reports screen.
func TopPayees(txns []domain.Transaction, n int) []PayeeSummary {
	type agg struct {
		count int
		total int64
	}
	sums := map[string]*agg{}
	var order []string
	for _, t := range txns {
		if t.AmountMinor >= 0 || t.Category == domain.CategorySavingsTransfer {
			continue
		}
		a, ok := sums[t.Payee]
		if !ok {
			a = &agg{}
			sums[t.Payee] = a
			order = append(order, t.Payee)
		}
		a.count++
		a.total += -t.AmountMinor
	}
	out := make([]PayeeSummary, 0, len(order))
	for _, p := range order {
		out = append(out, PayeeSummary{Payee: p, Count: sums[p].count, TotalMinor: sums[p].total})
	}
	// Simple insertion sort by TotalMinor desc — payee lists are small (tens, not thousands),
	// so this stays readable instead of pulling in sort.Slice for a handful of items.
	for i := 1; i < len(out); i++ {
		j := i
		for j > 0 && out[j-1].TotalMinor < out[j].TotalMinor {
			out[j-1], out[j] = out[j], out[j-1]
			j--
		}
	}
	if len(out) > n {
		out = out[:n]
	}
	return out
}
