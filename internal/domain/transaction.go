package domain

import "time"

// Transaction amounts are signed minor units: positive = income, negative = expense.
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

func (t Transaction) IsIncome() bool {
	return t.AmountMinor >= 0
}

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
