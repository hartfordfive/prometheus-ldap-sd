package store

import (
	"fmt"
	"testing"
)

func genErrorMsg(expectedMsg string, errCode int) string {
	return fmt.Sprintf("%s %d: \"%s\"", "LdapStore result code", errCode, expectedMsg)
}

func TestErrors(t *testing.T) {

	errCode := 0
	errs := []error{
		&Error{},
		&Error{Code: LdapStoreError},
		&Error{Code: uint16(errCode)},
	}
	for _, err := range errs {
		if err.Error() != genErrorMsg("Undefined store error", 0) {
			t.Errorf("Expecting error: %q, wanted %q", err.Error(), LDAPStoreErrorCodeMap[LdapStoreError])
		}
	}

	errCode = 1
	errs = []error{
		&Error{Code: LdapStoreErrorInvalidTargetGroup},
		&Error{Code: uint16(errCode)},
	}
	for _, err := range errs {
		if err.Error() != genErrorMsg("Invalid or empty target group specified", errCode) {
			t.Errorf("Expecting error: %q, wanted %q", err.Error(), LDAPStoreErrorCodeMap[LdapStoreErrorInvalidTargetGroup])
		}
	}

	errCode = 2
	errs = []error{
		&Error{Code: LdapStoreErrorInvalidQuery},
		&Error{Code: uint16(errCode)},
	}
	for _, err := range errs {
		if err.Error() != genErrorMsg("Invalid LDAP query", errCode) {
			t.Errorf("Expecting error: %q, wanted %q", err.Error(), LDAPStoreErrorCodeMap[LdapStoreErrorInvalidQuery])
		}
	}

	errCode = 3
	errs = []error{
		&Error{Code: LdapStoreErrorMaxReconnects},
		&Error{Code: uint16(errCode)},
	}
	for _, err := range errs {
		if err.Error() != genErrorMsg("Maximum reconnection attempts reached", errCode) {
			t.Errorf("Expecting error: %q, wanted %q", err.Error(), LDAPStoreErrorCodeMap[LdapStoreErrorMaxReconnects])
		}
	}

	errCode = 4
	errs = []error{
		&Error{Code: LdapStoreErrorCache},
		&Error{Code: uint16(errCode)},
	}
	for _, err := range errs {
		if err.Error() != genErrorMsg("A general cache error was encountered", errCode) {
			t.Errorf("Expecting error: %q, wanted %q", err.Error(), LDAPStoreErrorCodeMap[LdapStoreErrorCache])
		}
	}

	errCode = 5
	errs = []error{
		&Error{Code: LdapStoreErrorCacheUpdate},
		&Error{Code: uint16(errCode)},
	}
	for _, err := range errs {
		if err.Error() != genErrorMsg("The cache update operation failed", errCode) {
			t.Errorf("Expecting error: %q, wanted %q", err.Error(), LDAPStoreErrorCodeMap[LdapStoreErrorCacheUpdate])
		}
	}

	errCode = 6
	errs = []error{
		&Error{Code: LdapStoreErrorCacheFetch},
		&Error{Code: uint16(errCode)},
	}
	for _, err := range errs {
		if err.Error() != genErrorMsg("The cache fetch operation failed", errCode) {
			t.Errorf("Expecting error: %q, wanted %q", err.Error(), LDAPStoreErrorCodeMap[LdapStoreErrorCacheFetch])
		}
	}

}
