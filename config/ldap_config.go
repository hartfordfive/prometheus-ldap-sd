package config

import (
	"errors"
	"strings"
)

// LdapConfig is the configuration used to specify the properties of the LDAP queries
type LdapConfig struct {
	URL                      string              `yaml:"server"`
	BindDN                   string              `yaml:"bind_dn"`
	BaseDnMapping            map[string][]string `yaml:"base_dn_mappings"`
	GroupExporterPortMapping map[string]int      `yaml:"group_exporter_port_mapping"`
	Filter                   string              `yaml:"filter"`
	Attributes               []string            `yaml:"attributes"`
	PasswordEnvVar           string              `yaml:"password_env_var"`
	Authenticated            bool                `yaml:"authenticated"`
	Unsecured                bool                `yaml:"unsecured"`
	CacheDir                 string              `yaml:"cache_dir"`
	CacheTTL                 int                 `yaml:"cache_ttl"`
}

// Validate ensures that the current ldap configuration is valid
func (c *LdapConfig) Validate() error {
	if c.URL == "" {
		return errors.New("ldap_config.server configuration must be set to a valid address (format: <LDAP_HOST:<LDAP_PORT>)")
	}
	if c.CacheDir == "" {
		c.CacheDir = "./.cache"
	}
	if c.CacheTTL <= 0 {
		c.CacheTTL = 1 // Setting default to 1 second as 0 would mean no expiry
	}
	if c.BindDN == "" {
		return errors.New("ldap_config.bind_dn must be set")
	}
	if len(c.BaseDnMapping) == 0 {
		return errors.New("ldap_config.base_dn_mappings must be set")
	}
	if len(c.GroupExporterPortMapping) == 0 {
		return errors.New("ldap_config.group_exporter_port_mapping must be set")
	}
	if c.Filter == "" {
		c.Filter = "(&(objectClass=computer))"
	}
	if len(c.Attributes) == 0 {
		return errors.New("ldap_config.attributes must be set")
	}
	if c.Authenticated && strings.Trim(c.PasswordEnvVar, " ") == "" {
		return errors.New("The password_env_var value must be specified when authenticated=true")
	}
	return nil
}
