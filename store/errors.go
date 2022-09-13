package store

import "fmt"

// LdapStoreErrorInvalidTargetGroup is an error relating to an invalid target group name
type LdapStoreErrorInvalidTargetGroup struct {
	TargetGroup string
}

func (e *LdapStoreErrorInvalidTargetGroup) Error() string {
	return fmt.Sprintf("Invalid or empty target group specified: %s", e.TargetGroup)
}

// LdapStoreErrorInvalidQuery is an error relating to an invalide LDAP query
type LdapStoreErrorInvalidQuery struct{}

func (e *LdapStoreErrorInvalidQuery) Error() string {
	return "Invalid LDAP query"
}

// LdapStoreErrorMaxReconnects is an error relating to hitting the maximum reconneciton attempts
type LdapStoreErrorMaxReconnects struct{}

func (e *LdapStoreErrorMaxReconnects) Error() string {
	return "Maximum reconnection attempts reached"
}

// LdapStoreErrorCache is an error relating to a general caching error
type LdapStoreErrorCache struct{}

func (e *LdapStoreErrorCache) Error() string {
	return "A general cache error was encountered"
}

// LdapStoreErrorCacheUpdate is an error relating to an error updating the local cache
type LdapStoreErrorCacheUpdate struct{}

func (e *LdapStoreErrorCacheUpdate) Error() string {
	return "The cache update operation failed"
}

// LdapStoreErrorCacheFetch is an error relating to fetching an item from cache
type LdapStoreErrorCacheFetch struct{}

func (e *LdapStoreErrorCacheFetch) Error() string {
	return "The cache fetch operation failed"
}
