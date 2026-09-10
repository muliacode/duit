package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type SQLiteRuleRepository struct{ db *DB }

func NewRuleRepository(db *DB) *SQLiteRuleRepository { return &SQLiteRuleRepository{db: db} }

func (r *SQLiteRuleRepository) List(ctx context.Context) ([]CategorizationRule, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,match_text,category FROM categorization_rules ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CategorizationRule
	for rows.Next() {
		var cr CategorizationRule
		if err := rows.Scan(&cr.ID, &cr.MatchText, &cr.Category); err != nil {
			return nil, err
		}
		out = append(out, cr)
	}
	return out, rows.Err()
}

func (r *SQLiteRuleRepository) Create(ctx context.Context, cr CategorizationRule) (CategorizationRule, error) {
	if cr.ID == "" {
		cr.ID = uuid.NewString()
	}
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO categorization_rules (id,match_text,category,created_at) VALUES (?,?,?,?)`,
		cr.ID,
		cr.MatchText,
		cr.Category,
		fmtTime(time.Now().UTC()),
	)
	return cr, err
}

func (r *SQLiteRuleRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM categorization_rules WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return checkAffected(res)
}
