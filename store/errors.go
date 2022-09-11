package store

/*
	RESULT_INVALID_QUERY
	RESULT_MAX_RECONNECTS
	RESULT_CACHE_ERROR
	RESULT_CACHE_UPDATE_FAIL
	RESULT_CACHE_FETCH_FAIL
*/

type LdapStoreErrorInvalidQuery struct{}

func (e *LdapStoreErrorInvalidQuery) Error() string {
	return "Invalid LDAP query"
}

type LdapStoreErrorMaxReconnects struct{}

func (e *LdapStoreErrorMaxReconnects) Error() string {
	return "Maximum reconnection attempts reached"
}

type LdapStoreErrorCache struct{}

func (e *LdapStoreErrorCache) Error() string {
	return "A general cache error was encountered"
}

type LdapStoreErrorCacheUpdate struct{}

func (e *LdapStoreErrorCacheUpdate) Error() string {
	return "The cache update operation failed"
}

type LdapStoreErrorCacheFetch struct{}

func (e *LdapStoreErrorCacheFetch) Error() string {
	return "The cache fetch operation failed"
}
