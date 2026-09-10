package service

import (
	"context"
	"time"

	"duit/internal/domain"
	"duit/internal/store"
)

type BillService struct {
	repo store.BillRepository
}

func NewBillService(repo store.BillRepository) *BillService { return &BillService{repo: repo} }

func (s *BillService) List(ctx context.Context) ([]domain.Bill, error) { return s.repo.List(ctx) }

func (s *BillService) Create(ctx context.Context, b domain.Bill) (domain.Bill, error) {
	if err := b.Validate(); err != nil {
		return domain.Bill{}, err
	}
	return s.repo.Create(ctx, b)
}

func (s *BillService) Update(ctx context.Context, b domain.Bill) error {
	if err := b.Validate(); err != nil {
		return err
	}
	return s.repo.Update(ctx, b)
}

func (s *BillService) MarkPaid(ctx context.Context, id string) error {
	bills, err := s.repo.List(ctx)
	if err != nil {
		return err
	}
	for _, b := range bills {
		if b.ID == id {
			b.Status = domain.BillPaid
			return s.repo.Update(ctx, b)
		}
	}
	return domain.ErrNotFound
}

func (s *BillService) Delete(ctx context.Context, id string) error { return s.repo.Delete(ctx, id) }

// Status buckets a bill against "now" the same way the Bills screen colors
// it: overdue (danger), due within 3 days (warning), otherwise neutral —
// paid always wins. Centralizing this here means the Go tests are the one
// place that pin down the day thresholds; the frontend just renders what
// this returns instead of recomputing its own thresholds.
type Status string

const (
	StatusPaid     Status = "paid"
	StatusOverdue  Status = "overdue"
	StatusDueSoon  Status = "due_soon"
	StatusUpcoming Status = "upcoming"
)

func BillDisplayStatus(b domain.Bill, now time.Time) (Status, int) {
	if b.Status == domain.BillPaid {
		return StatusPaid, 0
	}
	days := int(b.NextDue.Sub(startOfDay(now)).Hours() / 24)
	switch {
	case days < 0:
		return StatusOverdue, days
	case days <= 3:
		return StatusDueSoon, days
	default:
		return StatusUpcoming, days
	}
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// CalendarEntry is one dot on the Bills calendar view for a single day.
type CalendarEntry struct {
	Day   int
	Label string
}

// OccurrencesForMonth expands every bill's cadence into the days it falls on
// within the given month, so navigating the calendar forward/backward shows
// real projected due dates instead of only ever the single stored NextDue.
func OccurrencesForMonth(bills []domain.Bill, year int, month time.Month) map[int][]CalendarEntry {
	from := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)
	out := map[int][]CalendarEntry{}
	for _, b := range bills {
		for _, d := range b.OccurrencesInRange(from, to) {
			out[d.Day()] = append(out[d.Day()], CalendarEntry{Day: d.Day(), Label: b.Name})
		}
	}
	return out
}
