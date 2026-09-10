package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"duit/internal/domain"

	"github.com/google/uuid"
)

type SQLiteTransactionRepository struct{ db *DB }

func NewTransactionRepository(db *DB) *SQLiteTransactionRepository {
	return &SQLiteTransactionRepository{db: db}
}

const txnColumns = `id,account_id,date,payee,category,amount_minor,note,receipt_ref,created_at,updated_at`

func (r *SQLiteTransactionRepository) List(ctx context.Context, f TransactionFilter) ([]domain.Transaction, error) {
	q := strings.Builder{}
	q.WriteString("SELECT " + txnColumns + " FROM transactions WHERE 1=1")
	var args []any

	if f.AccountID != "" {
		q.WriteString(" AND account_id = ?")
		args = append(args, f.AccountID)
	}
	if f.Category != "" {
		q.WriteString(" AND category = ?")
		args = append(args, f.Category)
	}
	if f.Query != "" {
		q.WriteString(" AND (payee LIKE ? OR category LIKE ?)")
		like := "%" + f.Query + "%"
		args = append(args, like, like)
	}
	switch f.Kind {
	case "income":
		q.WriteString(" AND amount_minor >= 0")
	case "expense":
		q.WriteString(" AND amount_minor < 0")
	}
	if f.FromDate != "" {
		q.WriteString(" AND date >= ?")
		args = append(args, f.FromDate)
	}
	if f.ToDate != "" {
		q.WriteString(" AND date < ?")
		args = append(args, f.ToDate)
	}

	sortCol := map[string]string{"date": "date", "payee": "payee", "category": "category", "amount": "amount_minor"}[f.SortBy]
	if sortCol == "" {
		sortCol = "date"
	}
	dir := "DESC"
	if !f.SortDesc {
		dir = "ASC"
	}
	fmt.Fprintf(&q, " ORDER BY %s %s, id %s", sortCol, dir, dir)

	rows, err := r.db.QueryContext(ctx, q.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Transaction
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *SQLiteTransactionRepository) Get(ctx context.Context, id string) (domain.Transaction, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+txnColumns+` FROM transactions WHERE id = ?`, id)
	t, err := scanTransaction(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Transaction{}, domain.ErrNotFound
	}
	return t, err
}

func (r *SQLiteTransactionRepository) Create(ctx context.Context, t domain.Transaction) (domain.Transaction, error) {
	created, err := r.CreateMany(ctx, []domain.Transaction{t})
	if err != nil {
		return domain.Transaction{}, err
	}
	return created[0], nil
}

// CreateMany backs both a single "add transaction" and a split transaction
// (several categories, one save action) with one code path — a split is
// simply CreateMany with len(ts) > 1sharing date/payee/account.
func (r *SQLiteTransactionRepository) CreateMany(
	ctx context.Context,
	ts []domain.Transaction,
) ([]domain.Transaction, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	now := time.Now().UTC()
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO transactions (`+txnColumns+`) VALUES (?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	for i := range ts {
		if ts[i].ID == "" {
			ts[i].ID = uuid.NewString()
		}
		ts[i].CreatedAt, ts[i].UpdatedAt = now, now
		if _, err := stmt.ExecContext(ctx, ts[i].ID, ts[i].AccountID, ts[i].Date.Format("2006-01-02"),
			ts[i].Payee, ts[i].Category, ts[i].AmountMinor, ts[i].Note, ts[i].ReceiptRef,
			fmtTime(ts[i].CreatedAt), fmtTime(ts[i].UpdatedAt)); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return ts, nil
}

func (r *SQLiteTransactionRepository) Update(ctx context.Context, t domain.Transaction) error {
	t.UpdatedAt = time.Now().UTC()
	res, err := r.db.ExecContext(
		ctx,
		`UPDATE transactions SET account_id=?,date=?,payee=?,category=?,amount_minor=?,note=?,receipt_ref=?,updated_at=? WHERE id=?`,
		t.AccountID,
		t.Date.Format("2006-01-02"),
		t.Payee,
		t.Category,
		t.AmountMinor,
		t.Note,
		t.ReceiptRef,
		fmtTime(t.UpdatedAt),
		t.ID,
	)
	if err != nil {
		return err
	}
	return checkAffected(res)
}

func (r *SQLiteTransactionRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM transactions WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return checkAffected(res)
}

func (r *SQLiteTransactionRepository) DeleteMany(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	_, err := r.db.ExecContext(ctx, "DELETE FROM transactions WHERE id IN ("+placeholders+")", args...)
	return err
}

func scanTransaction(s scanner) (domain.Transaction, error) {
	var t domain.Transaction
	var date, createdAt, updatedAt string
	if err := s.Scan(
		&t.ID,
		&t.AccountID,
		&date,
		&t.Payee,
		&t.Category,
		&t.AmountMinor,
		&t.Note,
		&t.ReceiptRef,
		&createdAt,
		&updatedAt,
	); err != nil {
		return domain.Transaction{}, err
	}
	t.Date, _ = time.Parse("2006-01-02", date)
	t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return t, nil
}
