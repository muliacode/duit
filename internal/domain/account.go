package domain

import "time"

type AccountType string

const (
	AccountChecking   AccountType = "checking"
	AccountSavings    AccountType = "savings"
	AccountCash       AccountType = "cash"
	AccountProperty   AccountType = "property"
	AccountCreditCard AccountType = "credit_card"
	AccountLoan       AccountType = "loan"
	AccountMortgage   AccountType = "mortgage"
)

func (t AccountType) Valid() bool {
	switch t {
	case AccountChecking, AccountSavings, AccountCash, AccountProperty, AccountCreditCard, AccountLoan, AccountMortgage:
		return true
	}
	return false
}

// Label returns the sentence-case display name used throughout the UI.
func (t AccountType) Label() string {
	switch t {
	case AccountChecking:
		return "Checking"
	case AccountSavings:
		return "Savings"
	case AccountCash:
		return "Cash"
	case AccountProperty:
		return "Property"
	case AccountCreditCard:
		return "Credit card"
	case AccountLoan:
		return "Loan"
	case AccountMortgage:
		return "Mortgage"
	default:
		return string(t)
	}
}

type AccountKind string

const (
	KindAsset     AccountKind = "asset"
	KindLiability AccountKind = "liability"
)

// Kind derives asset/liability from the account type, so it is never stored
// redundantly and can never drift out of sync with the type.
func (t AccountType) Kind() AccountKind {
	switch t {
	case AccountCreditCard, AccountLoan, AccountMortgage:
		return KindLiability
	default:
		return KindAsset
	}
}

// Account is the domain entity. All monetary fields are minor units
// (cents) to avoid float rounding errors.
type Account struct {
	ID                  string
	Name                string
	Type                AccountType
	Institution         string
	Number              string
	Currency            string // ISO 4217; empty means "use the app's default currency"
	BalanceMinor        int64
	CreditLimitMinor    *int64 // credit cards only
	OriginalAmountMinor *int64 // loans/mortgages: original principal, for payoff-progress math
	InterestRateBps     *int   // loans/mortgages/credit cards: basis points, e.g. 380 = 3.80% APR
	TermMonths          *int   // mortgages: original term
	MinPaymentMinor     *int64 // loans/mortgages/credit cards: minimum monthly payment, for payoff projections
	OpenedAt            *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (a Account) Kind() AccountKind {
	return a.Type.Kind()
}

func (a Account) Validate() error {
	if a.Name == "" {
		return domainValidation("name", "Give the account a name.")
	}
	if !a.Type.Valid() {
		return domainValidation("type", "Choose a valid account type.")
	}
	return nil
}

func domainValidation(field, msg string) error { return NewValidationError(field, msg) }
