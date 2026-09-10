package store

import (
	"context"
	"time"

	"duit/internal/domain"

	"github.com/google/uuid"
)

type SQLiteGoalRepository struct{ db *DB }

func NewGoalRepository(db *DB) *SQLiteGoalRepository { return &SQLiteGoalRepository{db: db} }

func (r *SQLiteGoalRepository) List(ctx context.Context) ([]domain.Goal, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id,name,target_minor,saved_minor,target_date,linked_account_id,created_at,updated_at FROM goals ORDER BY created_at`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Goal
	for rows.Next() {
		g, err := scanGoal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (r *SQLiteGoalRepository) Create(ctx context.Context, g domain.Goal) (domain.Goal, error) {
	if g.ID == "" {
		g.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	g.CreatedAt, g.UpdatedAt = now, now
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO goals (id,name,target_minor,saved_minor,target_date,linked_account_id,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)`,
		g.ID,
		g.Name,
		g.TargetMinor,
		g.SavedMinor,
		dateOrNil(g.TargetDate),
		nullIfEmpty(g.LinkedAccountID),
		fmtTime(g.CreatedAt),
		fmtTime(g.UpdatedAt),
	)
	return g, err
}

func (r *SQLiteGoalRepository) Update(ctx context.Context, g domain.Goal) error {
	g.UpdatedAt = time.Now().UTC()
	res, err := r.db.ExecContext(
		ctx,
		`UPDATE goals SET name=?,target_minor=?,saved_minor=?,target_date=?,linked_account_id=?,updated_at=? WHERE id=?`,
		g.Name,
		g.TargetMinor,
		g.SavedMinor,
		dateOrNil(g.TargetDate),
		nullIfEmpty(g.LinkedAccountID),
		fmtTime(g.UpdatedAt),
		g.ID,
	)
	if err != nil {
		return err
	}
	return checkAffected(res)
}

func (r *SQLiteGoalRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM goals WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return checkAffected(res)
}

func scanGoal(s scanner) (domain.Goal, error) {
	var g domain.Goal
	var targetDate, linkedAccountID, createdAt, updatedAt string
	if err := s.Scan(
		&g.ID,
		&g.Name,
		&g.TargetMinor,
		&g.SavedMinor,
		&nullString{&targetDate},
		&nullString{&linkedAccountID},
		&createdAt,
		&updatedAt,
	); err != nil {
		return domain.Goal{}, err
	}
	if targetDate != "" {
		t, _ := time.Parse("2006-01-02", targetDate)
		g.TargetDate = &t
	}
	g.LinkedAccountID = linkedAccountID
	g.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	g.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return g, nil
}

func dateOrNil(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format("2006-01-02")
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
