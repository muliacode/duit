package domain

import "time"

// CategorySavingsTransfer is the one category name treated as an internal
// transfer rather than spending — excluded from "spend this month", budget
// consumption, and reporting totals. Centralized here so every layer that
// needs the exclusion (budget/report services) references one constant
// instead of repeating the string.
const CategorySavingsTransfer = "Savings transfer"

// Transaction amounts are signed minor units: positive = income, negative = expense.
// This single-sign convention (rather than a separate "kind" enum plus an
// unsigned amount) is the one source of truth for direction, so a bug can
// never show an expense as income or vice versa.
type Transaction struct {
	ID          string
	AccountID   string
	Date        time.Time
	Payee       string
	Category    string
	AmountMinor int64
	Note        string
	ReceiptRef  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (t Transaction) IsIncome() bool { return t.AmountMinor >= 0 }

func (t Transaction) Validate() error {
	if t.AccountID == "" {
		return domainValidation("accountId", "Pick an account.")
	}
	if t.Category == "" {
		return domainValidation("category", "Pick or type a category.")
	}
	if t.AmountMinor == 0 {
		return domainValidation("amount", "Enter an amount greater than zero.")
	}
	return nil
}

// Split represents one line of a split transaction as submitted by the UI.
// The service layer turns a SplitRequest into N Transaction rows that share
// a date/payee/account but carry different categories and amounts.
type Split struct {
	Category    string
	AmountMinor int64
}
