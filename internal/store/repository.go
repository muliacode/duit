package store

import (
	"context"

	"duit/internal/domain"
)

// These interfaces are what internal/service depends on. Keeping them here
// (next to the SQLite implementations) rather than in internal/domain is a
// deliberate choice: domain stays free of any persistence concern, and a
// service test can implement a tiny in-memory fake of one of these
// interfaces without importing database/sql at all.

type AccountRepository interface {
	List(ctx context.Context) ([]domain.Account, error)
	Get(ctx context.Context, id string) (domain.Account, error)
	Create(ctx context.Context, a domain.Account) (domain.Account, error)
	Update(ctx context.Context, a domain.Account) error
	Delete(ctx context.Context, id string) error
}

type TransactionRepository interface {
	List(ctx context.Context, f TransactionFilter) ([]domain.Transaction, error)
	Get(ctx context.Context, id string) (domain.Transaction, error)
	Create(ctx context.Context, t domain.Transaction) (domain.Transaction, error)
	CreateMany(ctx context.Context, ts []domain.Transaction) ([]domain.Transaction, error)
	Update(ctx context.Context, t domain.Transaction) error
	Delete(ctx context.Context, id string) error
	DeleteMany(ctx context.Context, ids []string) error
}

// TransactionFilter mirrors the Transactions screen's filter bar exactly —
// zero values mean "no filter" for that dimension.
type TransactionFilter struct {
	AccountID string
	Category  string
	Query     string // matches payee or category, case-insensitive substring
	Kind      string // "income" | "expense" | "" (both)
	FromDate  string // inclusive, YYYY-MM-DD
	ToDate    string // exclusive, YYYY-MM-DD
	SortBy    string // "date" | "payee" | "category" | "amount" (default: date)
	SortDesc  bool
}

type BudgetRepository interface {
	List(ctx context.Context) ([]domain.BudgetCategory, error)
	Create(ctx context.Context, c domain.BudgetCategory) (domain.BudgetCategory, error)
	UpdateAllocated(ctx context.Context, name string, allocatedMinor int64) error
	Delete(ctx context.Context, name string) error
}

type BillRepository interface {
	List(ctx context.Context) ([]domain.Bill, error)
	Create(ctx context.Context, b domain.Bill) (domain.Bill, error)
	Update(ctx context.Context, b domain.Bill) error
	Delete(ctx context.Context, id string) error
}

type GoalRepository interface {
	List(ctx context.Context) ([]domain.Goal, error)
	Create(ctx context.Context, g domain.Goal) (domain.Goal, error)
	Update(ctx context.Context, g domain.Goal) error
	Delete(ctx context.Context, id string) error
}

type SettingsRepository interface {
	Get(ctx context.Context) (domain.Settings, error)
	Update(ctx context.Context, s domain.Settings) error
}

type CategorizationRule struct {
	ID        string
	MatchText string
	Category  string
}

type RuleRepository interface {
	List(ctx context.Context) ([]CategorizationRule, error)
	Create(ctx context.Context, r CategorizationRule) (CategorizationRule, error)
	Delete(ctx context.Context, id string) error
}
