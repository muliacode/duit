package main

// DTOs (data transfer objects) are what crosses the Wails JS<->Go boundary.
// They exist so the frontend never depends on internal/domain's shapes
// directly — domain.Account can gain/rename a field without any JSON
// contract changing, and the DTOs are where int64 minor-unit amounts and
// time.Time get turned into plain numbers/ISO strings that TypeScript can
// consume without a Go-specific runtime.
//
// Wails generates TypeScript types from these struct definitions the first
// time you run "wails dev" or "wails generate module" — keep field names
// PascalCase here; Wails' TS output preserves the JSON tag casing you set.

import (
	"time"

	"duit/internal/domain"
	"duit/internal/service"
	"duit/internal/store"
)

type AccountDTO struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Type                string `json:"type"`
	Kind                string `json:"kind"`
	Institution         string `json:"institution"`
	Number              string `json:"number"`
	Currency            string `json:"currency"`
	BalanceMinor        int64  `json:"balanceMinor"`
	CreditLimitMinor    *int64 `json:"creditLimitMinor,omitempty"`
	OriginalAmountMinor *int64 `json:"originalAmountMinor,omitempty"`
	InterestRateBps     *int   `json:"interestRateBps,omitempty"`
	TermMonths          *int   `json:"termMonths,omitempty"`
	MinPaymentMinor     *int64 `json:"minPaymentMinor,omitempty"`
	OpenedAt            string `json:"openedAt,omitempty"`
}

func toAccountDTO(a domain.Account) AccountDTO {
	dto := AccountDTO{
		ID: a.ID, Name: a.Name, Type: string(a.Type), Kind: string(a.Kind()),
		Institution: a.Institution, Number: a.Number, Currency: a.Currency, BalanceMinor: a.BalanceMinor,
		CreditLimitMinor: a.CreditLimitMinor, OriginalAmountMinor: a.OriginalAmountMinor,
		InterestRateBps: a.InterestRateBps, TermMonths: a.TermMonths, MinPaymentMinor: a.MinPaymentMinor,
	}
	if a.OpenedAt != nil {
		dto.OpenedAt = a.OpenedAt.Format("2006-01-02")
	}
	return dto
}

type CreateAccountRequest struct {
	Name                string `json:"name"`
	Type                string `json:"type"`
	Institution         string `json:"institution"`
	Currency            string `json:"currency"`
	BalanceMinor        int64  `json:"balanceMinor"`
	CreditLimitMinor    *int64 `json:"creditLimitMinor,omitempty"`
	OriginalAmountMinor *int64 `json:"originalAmountMinor,omitempty"`
	InterestRateBps     *int   `json:"interestRateBps,omitempty"`
	TermMonths          *int   `json:"termMonths,omitempty"`
	MinPaymentMinor     *int64 `json:"minPaymentMinor,omitempty"`
}

func (r CreateAccountRequest) toDomain() domain.Account {
	return domain.Account{
		Name: r.Name, Type: domain.AccountType(r.Type), Institution: r.Institution, Currency: r.Currency,
		BalanceMinor: r.BalanceMinor, CreditLimitMinor: r.CreditLimitMinor, OriginalAmountMinor: r.OriginalAmountMinor,
		InterestRateBps: r.InterestRateBps, TermMonths: r.TermMonths, MinPaymentMinor: r.MinPaymentMinor,
	}
}

type TransactionDTO struct {
	ID          string `json:"id"`
	AccountID   string `json:"accountId"`
	Date        string `json:"date"`
	Payee       string `json:"payee"`
	Category    string `json:"category"`
	AmountMinor int64  `json:"amountMinor"`
	Note        string `json:"note"`
	ReceiptRef  string `json:"receiptRef,omitempty"`
}

func toTransactionDTO(t domain.Transaction) TransactionDTO {
	return TransactionDTO{
		ID: t.ID, AccountID: t.AccountID, Date: t.Date.Format("2006-01-02"), Payee: t.Payee,
		Category: t.Category, AmountMinor: t.AmountMinor, Note: t.Note, ReceiptRef: t.ReceiptRef,
	}
}

type TransactionFilterRequest struct {
	AccountID string `json:"accountId,omitempty"`
	Category  string `json:"category,omitempty"`
	Query     string `json:"query,omitempty"`
	Kind      string `json:"kind,omitempty"`
	FromDate  string `json:"fromDate,omitempty"`
	ToDate    string `json:"toDate,omitempty"`
	SortBy    string `json:"sortBy,omitempty"`
	SortDesc  bool   `json:"sortDesc,omitempty"`
}

type CreateTransactionRequest struct {
	AccountID   string `json:"accountId"`
	Date        string `json:"date"`
	Payee       string `json:"payee"`
	Category    string `json:"category"`
	IsIncome    bool   `json:"isIncome"`
	AmountMinor int64  `json:"amountMinor"`
	Note        string `json:"note,omitempty"`
}

type SplitLineRequest struct {
	Category    string `json:"category"`
	AmountMinor int64  `json:"amountMinor"`
}

type CreateSplitTransactionRequest struct {
	AccountID string             `json:"accountId"`
	Date      string             `json:"date"`
	Payee     string             `json:"payee"`
	IsIncome  bool               `json:"isIncome"`
	Note      string             `json:"note,omitempty"`
	Splits    []SplitLineRequest `json:"splits"`
}

type BudgetCategoryDTO struct {
	Name           string  `json:"name"`
	AllocatedMinor int64   `json:"allocatedMinor"`
	SpentMinor     int64   `json:"spentMinor"`
	PercentUsed    float64 `json:"percentUsed"`
}

func toBudgetCategoryDTO(c service.CategoryStatus) BudgetCategoryDTO {
	return BudgetCategoryDTO{
		Name:           c.Name,
		AllocatedMinor: c.AllocatedMinor,
		SpentMinor:     c.SpentMinor,
		PercentUsed:    c.PercentUsed,
	}
}

type BillDTO struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	AmountMinor     int64  `json:"amountMinor"`
	CadenceInterval int    `json:"cadenceInterval"`
	CadenceUnit     string `json:"cadenceUnit"`
	CadenceLabel    string `json:"cadenceLabel"`
	NextDue         string `json:"nextDue"`
	AccountID       string `json:"accountId"`
	Status          string `json:"status"`
	DisplayStatus   string `json:"displayStatus"`
	DaysUntil       int    `json:"daysUntil"`
}

func toBillDTO(b domain.Bill, now time.Time) BillDTO {
	displayStatus, days := service.BillDisplayStatus(b, now)
	return BillDTO{
		ID: b.ID, Name: b.Name, AmountMinor: b.AmountMinor, CadenceInterval: b.CadenceInterval,
		CadenceUnit: string(b.CadenceUnit), CadenceLabel: b.CadenceLabel(), NextDue: b.NextDue.Format("2006-01-02"),
		AccountID: b.AccountID, Status: string(b.Status), DisplayStatus: string(displayStatus), DaysUntil: days,
	}
}

type CreateBillRequest struct {
	Name            string `json:"name"`
	AmountMinor     int64  `json:"amountMinor"`
	CadenceInterval int    `json:"cadenceInterval"`
	CadenceUnit     string `json:"cadenceUnit"`
	NextDue         string `json:"nextDue"`
	AccountID       string `json:"accountId"`
}

type GoalDTO struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	TargetMinor     int64   `json:"targetMinor"`
	SavedMinor      int64   `json:"savedMinor"`
	TargetDate      string  `json:"targetDate,omitempty"`
	LinkedAccountID string  `json:"linkedAccountId,omitempty"`
	ProgressPct     float64 `json:"progressPct"`
	Completed       bool    `json:"completed"`
}

func toGoalDTO(g domain.Goal) GoalDTO {
	dto := GoalDTO{
		ID: g.ID, Name: g.Name, TargetMinor: g.TargetMinor, SavedMinor: g.SavedMinor,
		LinkedAccountID: g.LinkedAccountID, ProgressPct: g.ProgressPct(), Completed: g.Completed(),
	}
	if g.TargetDate != nil {
		dto.TargetDate = g.TargetDate.Format("2006-01-02")
	}
	return dto
}

type CreateGoalRequest struct {
	Name            string `json:"name"`
	TargetMinor     int64  `json:"targetMinor"`
	SavedMinor      int64  `json:"savedMinor,omitempty"`
	TargetDate      string `json:"targetDate,omitempty"`
	LinkedAccountID string `json:"linkedAccountId,omitempty"`
}

// SettingsDTO deliberately omits PasscodeHash — the frontend only ever
// needs to know whether a passcode is set, never the hash itself.
type SettingsDTO struct {
	Currency       string `json:"currency"`
	Locale         string `json:"locale"`
	Theme          string `json:"theme"`
	DateFormat     string `json:"dateFormat"`
	HideAmounts    bool   `json:"hideAmounts"`
	BudgetMethod   string `json:"budgetMethod"`
	DebtStrategy   string `json:"debtStrategy"`
	RoundUpSavings bool   `json:"roundUpSavings"`
	HasPasscode    bool   `json:"hasPasscode"`
}

func toSettingsDTO(s domain.Settings) SettingsDTO {
	return SettingsDTO{
		Currency: s.Currency, Locale: s.Locale, Theme: string(s.Theme), DateFormat: string(s.DateFormat),
		HideAmounts: s.HideAmounts, BudgetMethod: string(s.BudgetMethod), DebtStrategy: string(s.DebtStrategy),
		RoundUpSavings: s.RoundUpSavings, HasPasscode: s.PasscodeHash != "",
	}
}

type UpdateSettingsRequest struct {
	Currency       string `json:"currency"`
	Locale         string `json:"locale"`
	Theme          string `json:"theme"`
	DateFormat     string `json:"dateFormat"`
	HideAmounts    bool   `json:"hideAmounts"`
	BudgetMethod   string `json:"budgetMethod"`
	DebtStrategy   string `json:"debtStrategy"`
	RoundUpSavings bool   `json:"roundUpSavings"`
}

type MonthPointDTO struct {
	Label        string `json:"label"` // e.g. "Sep 2026"
	IncomeMinor  int64  `json:"incomeMinor"`
	ExpenseMinor int64  `json:"expenseMinor"`
}

func toMonthPointDTO(p service.MonthPoint) MonthPointDTO {
	return MonthPointDTO{
		Label:        p.Month.String()[:3] + " " + itoa(p.Year),
		IncomeMinor:  p.IncomeMinor,
		ExpenseMinor: p.ExpenseMinor,
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := [4]byte{}
	i := 4
	for n > 0 && i > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	return string(digits[i:])
}

type CategoryTrendDTO struct {
	Category       string `json:"category"`
	ThisMonthMinor int64  `json:"thisMonthMinor"`
	LastMonthMinor int64  `json:"lastMonthMinor"`
}

type PayeeSummaryDTO struct {
	Payee      string `json:"payee"`
	Count      int    `json:"count"`
	TotalMinor int64  `json:"totalMinor"`
}

type CategorizationRuleDTO struct {
	ID        string `json:"id"`
	MatchText string `json:"matchText"`
	Category  string `json:"category"`
}

func toCategorizationRuleDTO(r store.CategorizationRule) CategorizationRuleDTO {
	return CategorizationRuleDTO{ID: r.ID, MatchText: r.MatchText, Category: r.Category}
}

type DebtPayoffDTO struct {
	AccountID    string `json:"accountId"`
	ExtraMinor   int64  `json:"extraMinor"`
	PayoffMonths int    `json:"payoffMonths"` // -1 means "never at this rate"
}
