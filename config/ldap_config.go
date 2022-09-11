package config

import (
	"errors"
	"fmt"
	"strings"
)

// LdapConfig is the configuration used to specify the properties of the LDAP queries
type LdapConfig struct {
	URL                  string                    `yaml:"server"`
	BindDN               string                    `yaml:"bind_dn"`
	BaseDnMappings       map[string]*BaseDnMapping `yaml:"base_dn_mappings"`
	Filter               string                    `yaml:"filter"`
	DefaultAttributes    []string                  `yaml:"default_attributes"`
	PasswordEnvVar       string                    `yaml:"password_env_var"`
	Authenticated        bool                      `yaml:"authenticated"`
	Unsecured            bool                      `yaml:"unsecured"`
	CacheDir             string                    `yaml:"cache_dir"`
	CacheTTL             int                       `yaml:"cache_ttl"`
	MaxReconnectAttempts int
}

type BaseDnMapping struct {
	BaseDnList   []string `yaml:"base_dn_list"`
	ExporterPort int      `yaml:"exporter_port"`
	Attributes   []string `yaml:"attributes"`
	Filter       string   `yaml:"filter"`
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
	if len(c.BaseDnMappings) == 0 {
		return errors.New("ldap_config.base_dn_mappings must be set")
	} else {
		for k, v := range c.BaseDnMappings {
			if len(v.BaseDnList) == 0 && v.Filter == "" {
				return fmt.Errorf("base_dn_list for %s must have at least one base DN or custom filter must be set", k)
			}
		}
	}

	if len(c.DefaultAttributes) == 0 {
		return errors.New("ldap_config.attributes must be set")
	}
	if c.Authenticated && strings.Trim(c.PasswordEnvVar, " ") == "" {
		return errors.New("The password_env_var value must be specified when authenticated=true")
	}
	return nil
}
