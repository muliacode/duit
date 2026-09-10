package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"duit/internal/domain"

	"github.com/google/uuid"
)

type SQLiteAccountRepository struct{ db *DB }

func NewAccountRepository(db *DB) *SQLiteAccountRepository { return &SQLiteAccountRepository{db: db} }

const accountColumns = `id,name,type,institution,number,currency,balance_minor,credit_limit_minor,original_amount_minor,interest_rate_bps,term_months,min_payment_minor,opened_at,created_at,updated_at`

func (r *SQLiteAccountRepository) List(ctx context.Context) ([]domain.Account, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+accountColumns+` FROM accounts ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Account
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *SQLiteAccountRepository) Get(ctx context.Context, id string) (domain.Account, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+accountColumns+` FROM accounts WHERE id = ?`, id)
	a, err := scanAccount(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Account{}, domain.ErrNotFound
	}
	return a, err
}

func (r *SQLiteAccountRepository) Create(ctx context.Context, a domain.Account) (domain.Account, error) {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	a.CreatedAt, a.UpdatedAt = now, now
	_, err := r.db.ExecContext(ctx, `INSERT INTO accounts (`+accountColumns+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		a.ID, a.Name, string(a.Type), a.Institution, a.Number, a.Currency, a.BalanceMinor,
		a.CreditLimitMinor, a.OriginalAmountMinor, a.InterestRateBps, a.TermMonths, a.MinPaymentMinor,
		timePtrToStr(a.OpenedAt), fmtTime(a.CreatedAt), fmtTime(a.UpdatedAt))
	return a, err
}

func (r *SQLiteAccountRepository) Update(ctx context.Context, a domain.Account) error {
	a.UpdatedAt = time.Now().UTC()
	res, err := r.db.ExecContext(
		ctx,
		`UPDATE accounts SET name=?,type=?,institution=?,number=?,currency=?,balance_minor=?,credit_limit_minor=?,original_amount_minor=?,interest_rate_bps=?,term_months=?,min_payment_minor=?,opened_at=?,updated_at=? WHERE id=?`,
		a.Name,
		string(a.Type),
		a.Institution,
		a.Number,
		a.Currency,
		a.BalanceMinor,
		a.CreditLimitMinor,
		a.OriginalAmountMinor,
		a.InterestRateBps,
		a.TermMonths,
		a.MinPaymentMinor,
		timePtrToStr(a.OpenedAt),
		fmtTime(a.UpdatedAt),
		a.ID,
	)
	if err != nil {
		return err
	}
	return checkAffected(res)
}

func (r *SQLiteAccountRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM accounts WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return checkAffected(res)
}

// scanner is satisfied by both *sql.Row and *sql.Rows, so scan* helpers work
// for both List (many rows) and Get (one row) without duplicating the column
// list in two places.
type scanner interface{ Scan(dest ...any) error }

func scanAccount(s scanner) (domain.Account, error) {
	var a domain.Account
	var typ, openedAt, createdAt, updatedAt string
	if err := s.Scan(&a.ID, &a.Name, &typ, &a.Institution, &a.Number, &a.Currency, &a.BalanceMinor,
		&a.CreditLimitMinor, &a.OriginalAmountMinor, &a.InterestRateBps, &a.TermMonths, &a.MinPaymentMinor,
		&nullString{&openedAt}, &createdAt, &updatedAt); err != nil {
		return domain.Account{}, err
	}
	a.Type = domain.AccountType(typ)
	a.OpenedAt = strToTimePtr(openedAt)
	a.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	a.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return a, nil
}
