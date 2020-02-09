package utils

// LdapConfig is the configuration struct passed to the search method
type LdapConfig struct {
	Host       string
	Port       int
	BaseDN     string
	Filter     string
	Attributes []string
}
