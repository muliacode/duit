package domain

import "time"

type CadenceUnit string

const (
	CadenceDay   CadenceUnit = "day"
	CadenceWeek  CadenceUnit = "week"
	CadenceMonth CadenceUnit = "month"
	CadenceYear  CadenceUnit = "year"
)

func (u CadenceUnit) Valid() bool {
	switch u {
	case CadenceDay, CadenceWeek, CadenceMonth, CadenceYear:
		return true
	}
	return false
}

type BillStatus string

const (
	BillUpcoming BillStatus = "upcoming"
	BillPaid     BillStatus = "paid"
)

type Bill struct {
	ID              string
	Name            string
	AmountMinor     int64
	CadenceInterval int
	CadenceUnit     CadenceUnit
	NextDue         time.Time
	AccountID       string
	Status          BillStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (b Bill) Validate() error {
	if b.Name == "" {
		return domainValidation("name", "Give the bill a name.")
	}
	if b.AmountMinor <= 0 {
		return domainValidation("amount", "Enter an amount greater than zero.")
	}
	if b.CadenceInterval <= 0 {
		return domainValidation("cadenceInterval", "Repeat interval must be at least 1.")
	}
	if !b.CadenceUnit.Valid() {
		return domainValidation("cadenceUnit", "Choose a valid repeat unit.")
	}
	return nil
}

// CadenceLabel renders "Monthly" for interval 1, "Every 2 weeks" otherwise —
// pure presentation logic, but it lives next to the data it describes so the
// Go and Vue layers can't drift on the pluralization rule.
func (b Bill) CadenceLabel() string {
	singular := map[CadenceUnit]string{
		CadenceDay:   "Daily",
		CadenceWeek:  "Weekly",
		CadenceMonth: "Monthly",
		CadenceYear:  "Yearly",
	}
	if b.CadenceInterval == 1 {
		if s, ok := singular[b.CadenceUnit]; ok {
			return s
		}
	}
	unit := string(b.CadenceUnit)
	return "Every " + itoa(b.CadenceInterval) + " " + unit + "s"
}

func (b Bill) step(d time.Time) time.Time {
	switch b.CadenceUnit {
	case CadenceDay:
		return d.AddDate(0, 0, b.CadenceInterval)
	case CadenceWeek:
		return d.AddDate(0, 0, 7*b.CadenceInterval)
	case CadenceYear:
		return d.AddDate(b.CadenceInterval, 0, 0)
	default: // month
		return d.AddDate(0, b.CadenceInterval, 0)
	}
}

// OccurrencesInRange walks forward from NextDue and returns every occurrence
// date in [from, to). Guarded to at most 240 steps (20 years of monthly
// cadence) so a misconfigured cadence can never loop forever.
func (b Bill) OccurrencesInRange(from, to time.Time) []time.Time {
	var occ []time.Time
	d := b.NextDue
	for guard := 0; guard < 240 && d.Before(to); guard++ {
		if !d.Before(from) {
			occ = append(occ, d)
		}
		d = b.step(d)
	}
	return occ
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
