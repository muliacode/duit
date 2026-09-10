package domain

import "time"

type Goal struct {
	ID              string
	Name            string
	TargetMinor     int64
	SavedMinor      int64
	TargetDate      *time.Time
	LinkedAccountID string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (g Goal) ProgressPct() float64 {
	if g.TargetMinor <= 0 {
		return 0
	}
	pct := float64(g.SavedMinor) / float64(g.TargetMinor) * 100
	if pct > 100 {
		return 100
	}
	return pct
}

func (g Goal) Completed() bool { return g.TargetMinor > 0 && g.SavedMinor >= g.TargetMinor }

func (g Goal) Validate() error {
	if g.Name == "" {
		return domainValidation("name", "Give the goal a name.")
	}
	if g.TargetMinor <= 0 {
		return domainValidation("target", "Enter a target amount greater than zero.")
	}
	return nil
}
