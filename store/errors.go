package store

import "fmt"

// LdapStore error codes
const (
	LdapStoreError                   = 0
	LdapStoreErrorInvalidTargetGroup = 1
	LdapStoreErrorInvalidQuery       = 2
	LdapStoreErrorMaxReconnects      = 3
	LdapStoreErrorCache              = 4
	LdapStoreErrorCacheUpdate        = 5
	LdapStoreErrorCacheFetch         = 6
)

// LDAPStoreErrorCodeMap contains string descriptions for LDAP error codes
var LDAPStoreErrorCodeMap = map[uint16]string{
	LdapStoreError:                   "Undefined store error",
	LdapStoreErrorInvalidTargetGroup: "Invalid or empty target group specified",
	LdapStoreErrorInvalidQuery:       "Invalid LDAP query",
	LdapStoreErrorMaxReconnects:      "Maximum reconnection attempts reached",
	LdapStoreErrorCache:              "A general cache error was encountered",
	LdapStoreErrorCacheUpdate:        "The cache update operation failed",
	LdapStoreErrorCacheFetch:         "The cache fetch operation failed",
}

// Error holds LdapStore error information
type Error struct {
	Properties map[string]string
	Code       uint16
}

func (e *Error) Error() string {
	err := fmt.Sprintf("LdapStore result code %d: %q", e.Code, LDAPStoreErrorCodeMap[e.Code])
	if len(e.Properties) >= 1 {
		err += " ("
		propIndex := 0
		for k, v := range e.Properties {
			err += fmt.Sprintf("%s=%s", k, v)
			if len(e.Properties) >= 2 && propIndex < len(e.Properties)-2 {
				err += ", "
			}
			propIndex++
		}
		err += ")"
	}
	return err
}
