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
