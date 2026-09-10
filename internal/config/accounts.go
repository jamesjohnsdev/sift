package config

// Account is one configured mail account. Kind is "gmail" or "outlook".
// ID becomes provider.AccountID and keys the OS keychain token and the
// storage database; there's no sensible default, so an empty Accounts
// list is the default.
type Account struct {
	ID       string
	Kind     string
	Email    string
	ClientID string
}
