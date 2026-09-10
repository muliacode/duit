package store

import (
	"database/sql/driver"
	"fmt"
	"time"

	"duit/internal/domain"
)

func fmtTime(t time.Time) string { return t.Format(time.RFC3339) }

func timePtrToStr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}

func strToTimePtr(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}

func checkAffected(res interface{ RowsAffected() (int64, error) }) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// nullString scans a NULL column into an empty string instead of requiring
// callers to juggle sql.NullString everywhere a text column is optional.
type nullString struct{ dest *string }

func (n *nullString) Scan(value any) error {
	if value == nil {
		*n.dest = ""
		return nil
	}
	switch v := value.(type) {
	case string:
		*n.dest = v
	case []byte:
		*n.dest = string(v)
	default:
		return fmt.Errorf("nullString: unsupported type %T", value)
	}
	return nil
}

var _ driver.Valuer = (*nullString)(nil)

func (n *nullString) Value() (driver.Value, error) { return *n.dest, nil }
