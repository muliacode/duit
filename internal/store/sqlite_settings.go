package store

import (
	"context"

	"duit/internal/domain"
)

type SQLiteSettingsRepository struct{ db *DB }

func NewSettingsRepository(db *DB) *SQLiteSettingsRepository {
	return &SQLiteSettingsRepository{db: db}
}

func (r *SQLiteSettingsRepository) Get(ctx context.Context) (domain.Settings, error) {
	var s domain.Settings
	var theme, dateFormat, budgetMethod, debtStrategy string
	var hideAmounts, roundUp int
	row := r.db.QueryRowContext(
		ctx,
		`SELECT currency,locale,theme,date_format,hide_amounts,budget_method,debt_strategy,round_up_savings,passcode_hash FROM settings WHERE id = 1`,
	)
	err := row.Scan(
		&s.Currency,
		&s.Locale,
		&theme,
		&dateFormat,
		&hideAmounts,
		&budgetMethod,
		&debtStrategy,
		&roundUp,
		&s.PasscodeHash,
	)
	if err != nil {
		return domain.Settings{}, err
	}
	s.Theme = domain.ThemeChoice(theme)
	s.DateFormat = domain.DateFormat(dateFormat)
	s.BudgetMethod = domain.BudgetMethod(budgetMethod)
	s.DebtStrategy = domain.DebtStrategy(debtStrategy)
	s.HideAmounts = hideAmounts != 0
	s.RoundUpSavings = roundUp != 0
	return s, nil
}

func (r *SQLiteSettingsRepository) Update(ctx context.Context, s domain.Settings) error {
	_, err := r.db.ExecContext(
		ctx,
		`UPDATE settings SET currency=?,locale=?,theme=?,date_format=?,hide_amounts=?,budget_method=?,debt_strategy=?,round_up_savings=?,passcode_hash=? WHERE id=1`,
		s.Currency,
		s.Locale,
		string(s.Theme),
		string(s.DateFormat),
		boolToInt(s.HideAmounts),
		string(s.BudgetMethod),
		string(s.DebtStrategy),
		boolToInt(s.RoundUpSavings),
		s.PasscodeHash,
	)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
