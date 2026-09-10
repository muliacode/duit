package service_test

import (
	"testing"
	"time"

	"duit/internal/domain"
	"duit/internal/service"
)

func TestBillDisplayStatus(t *testing.T) {
	now := time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		bill domain.Bill
		want service.Status
	}{
		{"paid always wins", domain.Bill{Status: domain.BillPaid, NextDue: now.AddDate(0, 0, -5)}, service.StatusPaid},
		{"overdue", domain.Bill{Status: domain.BillUpcoming, NextDue: now.AddDate(0, 0, -1)}, service.StatusOverdue},
		{"due today counts as due soon", domain.Bill{Status: domain.BillUpcoming, NextDue: now}, service.StatusDueSoon},
		{
			"due in 2 days is due soon",
			domain.Bill{Status: domain.BillUpcoming, NextDue: now.AddDate(0, 0, 2)},
			service.StatusDueSoon,
		},
		{
			"due in 10 days is upcoming",
			domain.Bill{Status: domain.BillUpcoming, NextDue: now.AddDate(0, 0, 10)},
			service.StatusUpcoming,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, _ := service.BillDisplayStatus(c.bill, now)
			if got != c.want {
				t.Errorf("got %s, want %s", got, c.want)
			}
		})
	}
}

func TestOccurrencesForMonth_BiweeklyAndSemiAnnual(t *testing.T) {
	insurance := domain.Bill{
		Name:            "Car insurance",
		CadenceInterval: 6,
		CadenceUnit:     domain.CadenceMonth,
		NextDue:         mustDate("2026-09-08"),
	}
	cleaning := domain.Bill{
		Name:            "Cleaning",
		CadenceInterval: 2,
		CadenceUnit:     domain.CadenceWeek,
		NextDue:         mustDate("2026-09-13"),
	}

	sep := service.OccurrencesForMonth([]domain.Bill{insurance, cleaning}, 2026, time.September)
	if _, ok := sep[8]; !ok {
		t.Fatalf("expected the insurance's first occurrence on day 8 of September, got %+v", sep)
	}
	if _, ok := sep[13]; !ok {
		t.Fatalf("expected a cleaning occurrence on day 13 of September, got %+v", sep)
	}

	march := service.OccurrencesForMonth([]domain.Bill{insurance}, 2027, time.March)
	if _, ok := march[8]; !ok {
		t.Fatalf("expected the semi-annual insurance to recur on day 8 of March 2027, got %+v", march)
	}

	october := service.OccurrencesForMonth([]domain.Bill{insurance}, 2026, time.October)
	if len(october) != 0 {
		t.Fatalf("expected no insurance occurrence in October (semi-annual, next is March), got %+v", october)
	}
}

func mustDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}
