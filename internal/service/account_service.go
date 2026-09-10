// Package service holds all business logic: validation beyond simple field
// checks, calculations (payoff projections, forecasts, net worth), and
// orchestration across repositories. The Wails App struct (see app.go) is a
// thin adapter that only translates between these services and the
// JSON-serializable DTOs Wails exposes to the frontend — it must never
// contain business logic itself, so that logic stays unit-testable without
// spinning up Wails.
package service

import (
	"context"

	"duit/internal/domain"
	"duit/internal/store"
)

type AccountService struct {
	repo store.AccountRepository
}

func NewAccountService(repo store.AccountRepository) *AccountService {
	return &AccountService{repo: repo}
}

func (s *AccountService) List(ctx context.Context) ([]domain.Account, error) { return s.repo.List(ctx) }

func (s *AccountService) Get(ctx context.Context, id string) (domain.Account, error) {
	return s.repo.Get(ctx, id)
}

func (s *AccountService) Create(ctx context.Context, a domain.Account) (domain.Account, error) {
	if err := a.Validate(); err != nil {
		return domain.Account{}, err
	}
	return s.repo.Create(ctx, a)
}

func (s *AccountService) Update(ctx context.Context, a domain.Account) error {
	if err := a.Validate(); err != nil {
		return err
	}
	return s.repo.Update(ctx, a)
}

func (s *AccountService) Delete(ctx context.Context, id string) error { return s.repo.Delete(ctx, id) }

// NetWorthMinor sums assets minus liabilities. Accounts held in a currency
// other than the app's base currency are summed without conversion — there
// is no exchange-rate source in this local-only app. See the handoff README
// ("Known simplifications") before treating this as accurate for
// multi-currency users.
func NetWorthMinor(accounts []domain.Account) int64 {
	var total int64
	for _, a := range accounts {
		if a.Kind() == domain.KindAsset {
			total += a.BalanceMinor
		} else {
			total -= a.BalanceMinor
		}
	}
	return total
}
