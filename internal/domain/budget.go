package domain

import "time"

type BudgetMethod string

const (
	BudgetSimple   BudgetMethod = "simple"
	BudgetEnvelope BudgetMethod = "envelope"
	BudgetOff      BudgetMethod = "off"
)

func (m BudgetMethod) Valid() bool {
	switch m {
	case BudgetSimple, BudgetEnvelope, BudgetOff:
		return true
	}
	return false
}

// BudgetCategory is deliberately the same row shape for both the "simple"
// and "envelope" methods: the methods differ only in how the frontend
// presents AllocatedMinor (a cap vs. an assignment) and in whether
// unspent amounts roll over, not in the stored schema.
type BudgetCategory struct {
	ID             string
	Name           string
	AllocatedMinor int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (c BudgetCategory) Validate() error {
	if c.Name == "" {
		return domainValidation("name", "Pick or type a category.")
	}
	if c.AllocatedMinor <= 0 {
		return domainValidation("allocated", "Enter an amount greater than zero.")
	}
	return nil
}
