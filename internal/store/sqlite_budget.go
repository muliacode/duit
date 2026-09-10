package store

import (
	"context"
	"time"

	"duit/internal/domain"

	"github.com/google/uuid"
)

type SQLiteBudgetRepository struct{ db *DB }

func NewBudgetRepository(db *DB) *SQLiteBudgetRepository { return &SQLiteBudgetRepository{db: db} }

func (r *SQLiteBudgetRepository) List(ctx context.Context) ([]domain.BudgetCategory, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id,name,allocated_minor,created_at,updated_at FROM budget_categories ORDER BY created_at`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.BudgetCategory
	for rows.Next() {
		var c domain.BudgetCategory
		var createdAt, updatedAt string
		if err := rows.Scan(&c.ID, &c.Name, &c.AllocatedMinor, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *SQLiteBudgetRepository) Create(ctx context.Context, c domain.BudgetCategory) (domain.BudgetCategory, error) {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	c.CreatedAt, c.UpdatedAt = now, now
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO budget_categories (id,name,allocated_minor,created_at,updated_at) VALUES (?,?,?,?,?)`,
		c.ID,
		c.Name,
		c.AllocatedMinor,
		fmtTime(c.CreatedAt),
		fmtTime(c.UpdatedAt),
	)
	return c, err
}

func (r *SQLiteBudgetRepository) UpdateAllocated(ctx context.Context, name string, allocatedMinor int64) error {
	res, err := r.db.ExecContext(ctx, `UPDATE budget_categories SET allocated_minor=?, updated_at=? WHERE name=?`,
		allocatedMinor, fmtTime(time.Now().UTC()), name)
	if err != nil {
		return err
	}
	return checkAffected(res)
}

func (r *SQLiteBudgetRepository) Delete(ctx context.Context, name string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM budget_categories WHERE name = ?`, name)
	if err != nil {
		return err
	}
	return checkAffected(res)
}
