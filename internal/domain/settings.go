package domain

type ThemeChoice string

const (
	ThemeSystem ThemeChoice = "system"
	ThemeLight  ThemeChoice = "light"
	ThemeDark   ThemeChoice = "dark"
)

type DateFormat string

const (
	DateHuman DateFormat = "human"
	DateDMY   DateFormat = "dmy"
	DateYMD   DateFormat = "ymd"
)

type DebtStrategy string

const (
	StrategyAvalanche DebtStrategy = "avalanche"
	StrategySnowball  DebtStrategy = "snowball"
	StrategyEqual     DebtStrategy = "equal"
	StrategyCustom    DebtStrategy = "custom"
)

// Settings is a singleton row (id=1 in SQLite) — there is exactly one
// settings record because duit is single-user and local-only.
type Settings struct {
	Currency       string
	Locale         string
	Theme          ThemeChoice
	DateFormat     DateFormat
	HideAmounts    bool
	BudgetMethod   BudgetMethod
	DebtStrategy   DebtStrategy
	RoundUpSavings bool
	PasscodeHash   string // empty string = passcode disabled
}
