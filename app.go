package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"duit/internal/domain"
	"duit/internal/service"
	"duit/internal/store"
)

// App is the Wails-bound struct: every exported method on it becomes
// callable from the frontend as an async function. It is intentionally a
// thin adapter — parse/validate the request DTO, call exactly one service
// method, map the result to a response DTO. Any "what does this button do"
// logic belongs in internal/service, not here, so it can be unit tested
// without Wails running.
type App struct {
	ctx context.Context
	db  *store.DB

	accounts     *service.AccountService
	transactions *service.TransactionService
	budgets      *service.BudgetService
	bills        *service.BillService
	debt         *service.DebtService
	goals        *service.GoalService
	reports      *service.ReportService
	settings     *service.SettingsService
	rules        store.RuleRepository
}

func NewApp() *App { return &App{} }

// startup wires the database and every service. Called once by Wails after
// the window is created. If the database can't be opened, the app can't
// function, so we panic rather than silently running with nil services.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	appDir := filepath.Join(dir, "duit")
	if err := os.MkdirAll(appDir, 0o700); err != nil {
		panic(fmt.Errorf("create app data directory: %w", err))
	}

	db, err := store.Open(filepath.Join(appDir, "duit.db"))
	if err != nil {
		panic(fmt.Errorf("open database: %w", err))
	}
	a.db = db

	accountRepo := store.NewAccountRepository(db)
	txnRepo := store.NewTransactionRepository(db)
	billRepo := store.NewBillRepository(db)

	a.accounts = service.NewAccountService(accountRepo)
	a.transactions = service.NewTransactionService(txnRepo, billRepo)
	a.budgets = service.NewBudgetService(store.NewBudgetRepository(db), txnRepo)
	a.bills = service.NewBillService(billRepo)
	a.debt = service.NewDebtService()
	a.goals = service.NewGoalService(store.NewGoalRepository(db))
	a.reports = service.NewReportService(txnRepo)
	a.settings = service.NewSettingsService(store.NewSettingsRepository(db))
	a.rules = store.NewRuleRepository(db)
}

func (a *App) shutdown(ctx context.Context) {
	if a.db != nil {
		a.db.Close()
	}
}

// ---------- Accounts ----------

func (a *App) ListAccounts() ([]AccountDTO, error) {
	accs, err := a.accounts.List(a.ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AccountDTO, len(accs))
	for i, acc := range accs {
		out[i] = toAccountDTO(acc)
	}
	return out, nil
}

func (a *App) CreateAccount(req CreateAccountRequest) (AccountDTO, error) {
	acc, err := a.accounts.Create(a.ctx, req.toDomain())
	if err != nil {
		return AccountDTO{}, err
	}
	return toAccountDTO(acc), nil
}

func (a *App) DeleteAccount(id string) error { return a.accounts.Delete(a.ctx, id) }

func (a *App) NetWorthMinor() (int64, error) {
	accs, err := a.accounts.List(a.ctx)
	if err != nil {
		return 0, err
	}
	return service.NetWorthMinor(accs), nil
}

// ---------- Transactions ----------

func (a *App) ListTransactions(f TransactionFilterRequest) ([]TransactionDTO, error) {
	txns, err := a.transactions.List(a.ctx, store.TransactionFilter{
		AccountID: f.AccountID, Category: f.Category, Query: f.Query, Kind: f.Kind,
		FromDate: f.FromDate, ToDate: f.ToDate, SortBy: f.SortBy, SortDesc: f.SortDesc,
	})
	if err != nil {
		return nil, err
	}
	out := make([]TransactionDTO, len(txns))
	for i, t := range txns {
		out[i] = toTransactionDTO(t)
	}
	return out, nil
}

func (a *App) CreateTransaction(req CreateTransactionRequest) (TransactionDTO, error) {
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return TransactionDTO{}, domain.NewValidationError("date", "Enter a valid date.")
	}
	amt := req.AmountMinor
	if !req.IsIncome {
		amt = -amt
	}
	t, err := a.transactions.Create(a.ctx, domain.Transaction{
		AccountID:   req.AccountID,
		Date:        date,
		Payee:       req.Payee,
		Category:    req.Category,
		AmountMinor: amt,
		Note:        req.Note,
	})
	if err != nil {
		return TransactionDTO{}, err
	}
	return toTransactionDTO(t), nil
}

func (a *App) CreateSplitTransaction(req CreateSplitTransactionRequest) ([]TransactionDTO, error) {
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, domain.NewValidationError("date", "Enter a valid date.")
	}
	splits := make([]domain.Split, len(req.Splits))
	for i, sp := range req.Splits {
		splits[i] = domain.Split{Category: sp.Category, AmountMinor: sp.AmountMinor}
	}
	created, err := a.transactions.CreateSplit(a.ctx, req.AccountID, req.Payee, date, splits, req.IsIncome, req.Note)
	if err != nil {
		return nil, err
	}
	out := make([]TransactionDTO, len(created))
	for i, t := range created {
		out[i] = toTransactionDTO(t)
	}
	return out, nil
}

func (a *App) UpdateTransaction(id string, req CreateTransactionRequest) (TransactionDTO, error) {
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return TransactionDTO{}, domain.NewValidationError("date", "Enter a valid date.")
	}
	amt := req.AmountMinor
	if !req.IsIncome {
		amt = -amt
	}
	t := domain.Transaction{
		ID:          id,
		AccountID:   req.AccountID,
		Date:        date,
		Payee:       req.Payee,
		Category:    req.Category,
		AmountMinor: amt,
		Note:        req.Note,
	}
	if err := a.transactions.Update(a.ctx, t); err != nil {
		return TransactionDTO{}, err
	}
	return toTransactionDTO(t), nil
}

func (a *App) DeleteTransaction(id string) error     { return a.transactions.Delete(a.ctx, id) }
func (a *App) DeleteTransactions(ids []string) error { return a.transactions.DeleteMany(a.ctx, ids) }

type RecurringCandidateDTO struct {
	Payee        string `json:"payee"`
	Occurrences  int    `json:"occurrences"`
	AverageMinor int64  `json:"averageMinor"`
}

// DetectRecurringTransaction looks at the last 60 days. Returns a nil
// pointer (serialized as JSON null) when nothing looks recurring.
func (a *App) DetectRecurringTransaction() (*RecurringCandidateDTO, error) {
	candidate, err := a.transactions.DetectRecurring(a.ctx, time.Now().AddDate(0, 0, -60))
	if err != nil {
		return nil, err
	}
	if candidate == nil {
		return nil, nil
	}
	return &RecurringCandidateDTO{
		Payee:        candidate.Payee,
		Occurrences:  candidate.Occurrences,
		AverageMinor: candidate.AverageMinor,
	}, nil
}

// ImportTransactionsCSV parses a "Date,Description,Amount[,Category]" CSV
// (Date as YYYY-MM-DD, Amount as a signed decimal e.g. -12.50) and creates
// one transaction per valid row, skipping the header and any malformed
// rows. Returns how many rows were imported.
func (a *App) ImportTransactionsCSV(accountID string, csvContent string) (int, error) {
	reader := csv.NewReader(strings.NewReader(csvContent))
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		return 0, fmt.Errorf("parse CSV: %w", err)
	}
	if len(rows) < 2 {
		return 0, nil
	}
	imported := 0
	for _, row := range rows[1:] {
		if len(row) < 3 {
			continue
		}
		date, err := time.Parse("2006-01-02", strings.TrimSpace(row[0]))
		if err != nil {
			continue
		}
		amount, err := strconv.ParseFloat(strings.TrimSpace(row[2]), 64)
		if err != nil {
			continue
		}
		category := "Other"
		if len(row) >= 4 && strings.TrimSpace(row[3]) != "" {
			category = strings.TrimSpace(row[3])
		}
		_, err = a.transactions.Create(a.ctx, domain.Transaction{
			AccountID: accountID, Date: date, Payee: strings.TrimSpace(row[1]), Category: category,
			AmountMinor: int64(amount * 100),
		})
		if err != nil {
			continue
		}
		imported++
	}
	return imported, nil
}

// ---------- Budgets ----------

func (a *App) ListBudgetCategories() ([]BudgetCategoryDTO, error) {
	now := time.Now()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)
	statuses, err := a.budgets.StatusForRange(a.ctx, from, to)
	if err != nil {
		return nil, err
	}
	out := make([]BudgetCategoryDTO, len(statuses))
	for i, s := range statuses {
		out[i] = toBudgetCategoryDTO(s)
	}
	return out, nil
}

func (a *App) CreateBudgetCategory(name string, allocatedMinor int64) (BudgetCategoryDTO, error) {
	c, err := a.budgets.Create(a.ctx, domain.BudgetCategory{Name: name, AllocatedMinor: allocatedMinor})
	if err != nil {
		return BudgetCategoryDTO{}, err
	}
	return BudgetCategoryDTO{Name: c.Name, AllocatedMinor: c.AllocatedMinor}, nil
}

func (a *App) DeleteBudgetCategory(name string) error { return a.budgets.Delete(a.ctx, name) }

func (a *App) AdjustBudgetAllocation(name string, deltaMinor int64) error {
	return a.budgets.AdjustAllocation(a.ctx, name, deltaMinor)
}

// UnassignedMinor is the envelope-method "ready to assign" figure.
func (a *App) UnassignedMinor() (int64, error) {
	now := time.Now()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)
	incomeTxns, err := a.transactions.List(
		a.ctx,
		store.TransactionFilter{Kind: "income", FromDate: from.Format("2006-01-02"), ToDate: to.Format("2006-01-02")},
	)
	if err != nil {
		return 0, err
	}
	var income int64
	for _, t := range incomeTxns {
		income += t.AmountMinor
	}
	cats, err := a.ListBudgetCategories()
	if err != nil {
		return 0, err
	}
	var allocated int64
	for _, c := range cats {
		allocated += c.AllocatedMinor
	}
	return income - allocated, nil
}

// ---------- Bills ----------

func (a *App) ListBills() ([]BillDTO, error) {
	bills, err := a.bills.List(a.ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	out := make([]BillDTO, len(bills))
	for i, b := range bills {
		out[i] = toBillDTO(b, now)
	}
	return out, nil
}

func (a *App) CreateBill(req CreateBillRequest) (BillDTO, error) {
	due, err := time.Parse("2006-01-02", req.NextDue)
	if err != nil {
		return BillDTO{}, domain.NewValidationError("nextDue", "Enter a valid date.")
	}
	b, err := a.bills.Create(a.ctx, domain.Bill{
		Name: req.Name, AmountMinor: req.AmountMinor, CadenceInterval: req.CadenceInterval,
		CadenceUnit: domain.CadenceUnit(req.CadenceUnit), NextDue: due, AccountID: req.AccountID,
	})
	if err != nil {
		return BillDTO{}, err
	}
	return toBillDTO(b, time.Now()), nil
}

func (a *App) MarkBillPaid(id string) error { return a.bills.MarkPaid(a.ctx, id) }
func (a *App) DeleteBill(id string) error   { return a.bills.Delete(a.ctx, id) }

// BillCalendar returns, for the given year/1-indexed month, a map of
// day-of-month -> bill names occurring that day (accounting for each bill's
// own cadence, not just its single stored NextDue).
func (a *App) BillCalendar(year int, month int) (map[string][]string, error) {
	bills, err := a.bills.List(a.ctx)
	if err != nil {
		return nil, err
	}
	occ := service.OccurrencesForMonth(bills, year, time.Month(month))
	out := make(map[string][]string, len(occ))
	for day, entries := range occ {
		labels := make([]string, len(entries))
		for i, e := range entries {
			labels[i] = e.Label
		}
		out[strconv.Itoa(day)] = labels
	}
	return out, nil
}

// ---------- Goals ----------

func (a *App) ListGoals() ([]GoalDTO, error) {
	goals, err := a.goals.List(a.ctx)
	if err != nil {
		return nil, err
	}
	out := make([]GoalDTO, len(goals))
	for i, g := range goals {
		out[i] = toGoalDTO(g)
	}
	return out, nil
}

func (a *App) CreateGoal(req CreateGoalRequest) (GoalDTO, error) {
	g := domain.Goal{
		Name:            req.Name,
		TargetMinor:     req.TargetMinor,
		SavedMinor:      req.SavedMinor,
		LinkedAccountID: req.LinkedAccountID,
	}
	if req.TargetDate != "" {
		if d, err := time.Parse("2006-01-02", req.TargetDate); err == nil {
			g.TargetDate = &d
		}
	}
	created, err := a.goals.Create(a.ctx, g)
	if err != nil {
		return GoalDTO{}, err
	}
	return toGoalDTO(created), nil
}

func (a *App) DeleteGoal(id string) error { return a.goals.Delete(a.ctx, id) }

// ---------- Debt ----------

func (a *App) DebtPayoffProjection(
	strategy string,
	extraPoolMinor int64,
	custom map[string]int64,
) ([]DebtPayoffDTO, error) {
	accs, err := a.accounts.List(a.ctx)
	if err != nil {
		return nil, err
	}
	var debts []domain.Account
	for _, acc := range accs {
		if acc.Kind() == domain.KindLiability && acc.Type != domain.AccountMortgage {
			debts = append(debts, acc)
		}
	}
	allocations := a.debt.AllocateExtra(domain.DebtStrategy(strategy), debts, extraPoolMinor, custom)
	byID := make(map[string]domain.Account, len(debts))
	for _, d := range debts {
		byID[d.ID] = d
	}
	out := make([]DebtPayoffDTO, 0, len(allocations))
	for _, alloc := range allocations {
		d := byID[alloc.AccountID]
		var minPayment int64
		if d.MinPaymentMinor != nil {
			minPayment = *d.MinPaymentMinor
		}
		var rateBps int
		if d.InterestRateBps != nil {
			rateBps = *d.InterestRateBps
		}
		months := a.debt.PayoffMonths(d.BalanceMinor, rateBps, minPayment, alloc.ExtraMinor)
		out = append(out, DebtPayoffDTO{AccountID: d.ID, ExtraMinor: alloc.ExtraMinor, PayoffMonths: months})
	}
	return out, nil
}

// MortgageProjection is separate from DebtPayoffProjection because a
// mortgage's "regular payment" is derived from its term/rate/principal
// (RegularPayment) rather than stored as an explicit minimum, and a
// mortgage is never part of the avalanche/snowball/equal/custom strategy
// pool (paying it off is a decades-long, deliberately separate decision).
func (a *App) MortgageProjection(accountID string, extraMinor int64) (DebtPayoffDTO, error) {
	acc, err := a.accounts.Get(a.ctx, accountID)
	if err != nil {
		return DebtPayoffDTO{}, err
	}
	var rateBps, termMonths int
	var original int64
	if acc.InterestRateBps != nil {
		rateBps = *acc.InterestRateBps
	}
	if acc.TermMonths != nil {
		termMonths = *acc.TermMonths
	}
	if acc.OriginalAmountMinor != nil {
		original = *acc.OriginalAmountMinor
	}
	regular := a.debt.RegularPayment(original, rateBps, termMonths)
	months := a.debt.PayoffMonths(acc.BalanceMinor, rateBps, regular, extraMinor)
	return DebtPayoffDTO{AccountID: acc.ID, ExtraMinor: extraMinor, PayoffMonths: months}, nil
}

// ---------- Reports ----------

func (a *App) MonthlyTrend(months int) ([]MonthPointDTO, error) {
	points, err := a.reports.MonthlyTrend(a.ctx, time.Now(), months)
	if err != nil {
		return nil, err
	}
	out := make([]MonthPointDTO, len(points))
	for i, p := range points {
		out[i] = toMonthPointDTO(p)
	}
	return out, nil
}

func (a *App) CategoryTrend() ([]CategoryTrendDTO, error) {
	rows, err := a.reports.CategoryTrend(a.ctx, time.Now())
	if err != nil {
		return nil, err
	}
	out := make([]CategoryTrendDTO, len(rows))
	for i, r := range rows {
		out[i] = CategoryTrendDTO{
			Category:       r.Category,
			ThisMonthMinor: r.ThisMonthMinor,
			LastMonthMinor: r.LastMonthMinor,
		}
	}
	return out, nil
}

func (a *App) TopPayees(limit int) ([]PayeeSummaryDTO, error) {
	now := time.Now()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	txns, err := a.transactions.List(a.ctx, store.TransactionFilter{FromDate: from.Format("2006-01-02")})
	if err != nil {
		return nil, err
	}
	top := service.TopPayees(txns, limit)
	out := make([]PayeeSummaryDTO, len(top))
	for i, p := range top {
		out[i] = PayeeSummaryDTO{Payee: p.Payee, Count: p.Count, TotalMinor: p.TotalMinor}
	}
	return out, nil
}

func (a *App) Forecast() (MonthPointDTO, error) {
	points, err := a.reports.MonthlyTrend(a.ctx, time.Now(), 6)
	if err != nil {
		return MonthPointDTO{}, err
	}
	income, expense := service.Forecast(points)
	return MonthPointDTO{Label: "Forecast", IncomeMinor: income, ExpenseMinor: expense}, nil
}

func (a *App) RoundUpTotalMinor() (int64, error) {
	now := time.Now()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	txns, err := a.transactions.List(
		a.ctx,
		store.TransactionFilter{Kind: "expense", FromDate: from.Format("2006-01-02")},
	)
	if err != nil {
		return 0, err
	}
	return service.RoundUpTotalMinor(txns, 100), nil
}

// ---------- Settings ----------

func (a *App) GetSettings() (SettingsDTO, error) {
	s, err := a.settings.Get(a.ctx)
	if err != nil {
		return SettingsDTO{}, err
	}
	return toSettingsDTO(s), nil
}

func (a *App) UpdateSettings(req UpdateSettingsRequest) (SettingsDTO, error) {
	current, err := a.settings.Get(a.ctx)
	if err != nil {
		return SettingsDTO{}, err
	}
	current.Currency = req.Currency
	current.Locale = req.Locale
	current.Theme = domain.ThemeChoice(req.Theme)
	current.DateFormat = domain.DateFormat(req.DateFormat)
	current.HideAmounts = req.HideAmounts
	current.BudgetMethod = domain.BudgetMethod(req.BudgetMethod)
	current.DebtStrategy = domain.DebtStrategy(req.DebtStrategy)
	current.RoundUpSavings = req.RoundUpSavings
	if err := a.settings.Update(a.ctx, current); err != nil {
		return SettingsDTO{}, err
	}
	return toSettingsDTO(current), nil
}

// EraseAllData is the Settings -> Danger zone action. It deletes every row
// from every table (settings excepted, which is reset to defaults) inside
// one transaction, rather than deleting the SQLite file, so the already-open
// *sql.DB handle stays valid for the rest of the running session.
func (a *App) EraseAllData() error {
	tx, err := a.db.BeginTx(a.ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	tables := []string{"transactions", "bills", "goals", "budget_categories", "categorization_rules", "accounts"}
	for _, table := range tables {
		if _, err := tx.ExecContext(a.ctx, "DELETE FROM "+table); err != nil {
			return err
		}
	}
	resetSQL := `UPDATE settings SET currency='EUR',locale='en',theme='system',date_format='human',hide_amounts=0,budget_method='simple',debt_strategy='avalanche',round_up_savings=0,passcode_hash='' WHERE id=1`
	if _, err := tx.ExecContext(a.ctx, resetSQL); err != nil {
		return err
	}
	return tx.Commit()
}

// ---------- Categorization rules ----------

func (a *App) ListRules() ([]CategorizationRuleDTO, error) {
	rules, err := a.rules.List(a.ctx)
	if err != nil {
		return nil, err
	}
	out := make([]CategorizationRuleDTO, len(rules))
	for i, r := range rules {
		out[i] = toCategorizationRuleDTO(r)
	}
	return out, nil
}

func (a *App) CreateRule(matchText, category string) (CategorizationRuleDTO, error) {
	r, err := a.rules.Create(a.ctx, store.CategorizationRule{MatchText: matchText, Category: category})
	if err != nil {
		return CategorizationRuleDTO{}, err
	}
	return toCategorizationRuleDTO(r), nil
}

func (a *App) DeleteRule(id string) error { return a.rules.Delete(a.ctx, id) }
