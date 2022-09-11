package config

import (
	"errors"
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

var GlobalConfig *Config

// Config is the top level configuration used by the service discovery module
type Config struct {
	Host       string      `yaml:"server_host" json:"server_host"`
	Port       int         `yaml:"server_port" json:"server_port"`
	LdapConfig *LdapConfig `yaml:"ldap_config" json:"ldap_config"`
}

// NewConfig constructs a new Config instance
func NewConfig(configPath string) (*Config, error) {
	c := &Config{}
	b, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("Could not read config: %s", err)
	}

	err = yaml.Unmarshal([]byte(b), &c)
	if err != nil {
		return nil, fmt.Errorf("Could not unmarshal config: %v", err)
	}

	if err := c.Validate(); err != nil {
		return nil, err
	}

	return c, nil
}

// Serialize serializes the configuration so that it can be printed and viewed
func (c *Config) Serialize() (string, error) {
	if b, err := yaml.Marshal(c); err != nil {
		return "", err
	} else {
		return string(b), nil
	}
}

func (c *Config) Validate() error {
	if c.Host == "" {
		c.Host = "127.0.0.1" // default value
	}
	if c.Port == 0 {
		c.Port = 8889 // default value
	}
	if c.Port < 1 || c.Port > 65535 {
		return errors.New("Value 'port' must be between 1 and 65535")
	}
	if c.LdapConfig == nil {
		return errors.New("Missing 'ldap_config' configuraton block")
	}
	return c.LdapConfig.Validate()
}
