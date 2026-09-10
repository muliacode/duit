package store

import (
	"context"
	"time"

	"duit/internal/domain"

	"github.com/google/uuid"
)

type SQLiteBillRepository struct{ db *DB }

func NewBillRepository(db *DB) *SQLiteBillRepository { return &SQLiteBillRepository{db: db} }

const billColumns = `id,name,amount_minor,cadence_interval,cadence_unit,next_due,account_id,status,created_at,updated_at`

func (r *SQLiteBillRepository) List(ctx context.Context) ([]domain.Bill, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+billColumns+` FROM bills ORDER BY next_due`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Bill
	for rows.Next() {
		b, err := scanBill(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *SQLiteBillRepository) Create(ctx context.Context, b domain.Bill) (domain.Bill, error) {
	if b.ID == "" {
		b.ID = uuid.NewString()
	}
	if b.Status == "" {
		b.Status = domain.BillUpcoming
	}
	now := time.Now().UTC()
	b.CreatedAt, b.UpdatedAt = now, now
	_, err := r.db.ExecContext(ctx, `INSERT INTO bills (`+billColumns+`) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		b.ID, b.Name, b.AmountMinor, b.CadenceInterval, string(b.CadenceUnit), b.NextDue.Format("2006-01-02"),
		b.AccountID, string(b.Status), fmtTime(b.CreatedAt), fmtTime(b.UpdatedAt))
	return b, err
}

func (r *SQLiteBillRepository) Update(ctx context.Context, b domain.Bill) error {
	b.UpdatedAt = time.Now().UTC()
	res, err := r.db.ExecContext(
		ctx,
		`UPDATE bills SET name=?,amount_minor=?,cadence_interval=?,cadence_unit=?,next_due=?,account_id=?,status=?,updated_at=? WHERE id=?`,
		b.Name,
		b.AmountMinor,
		b.CadenceInterval,
		string(b.CadenceUnit),
		b.NextDue.Format("2006-01-02"),
		b.AccountID,
		string(b.Status),
		fmtTime(b.UpdatedAt),
		b.ID,
	)
	if err != nil {
		return err
	}
	return checkAffected(res)
}

func (r *SQLiteBillRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM bills WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return checkAffected(res)
}

func scanBill(s scanner) (domain.Bill, error) {
	var b domain.Bill
	var cadenceUnit, nextDue, status, createdAt, updatedAt string
	if err := s.Scan(
		&b.ID,
		&b.Name,
		&b.AmountMinor,
		&b.CadenceInterval,
		&cadenceUnit,
		&nextDue,
		&b.AccountID,
		&status,
		&createdAt,
		&updatedAt,
	); err != nil {
		return domain.Bill{}, err
	}
	b.CadenceUnit = domain.CadenceUnit(cadenceUnit)
	b.Status = domain.BillStatus(status)
	b.NextDue, _ = time.Parse("2006-01-02", nextDue)
	b.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	b.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return b, nil
}
