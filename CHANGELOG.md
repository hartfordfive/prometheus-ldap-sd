# Changelog

### v0.4.2
- [bugfix] Update the `runDiscovery` to properly append all resulting objects from each defined OU as it was previously overwriting the slice and consquently only returning the results of the last specified OU.

### v0.4.1
- Updated `ldap_sd_build_info` metrics to include git hash via the `git_hash` label
- Updated connection and reconnection logic to use a single `connect()` function which uses a connection lock to ensure the operation is thread-safe.
- Improved structure of custom errors and added basic tests for them.
- Added the relevant target group and base DN to warning logs regarding an object missing the `dNSHostName` attribute.
- Updated `test` stage in make file.

### v0.4.1-alpha
- Updated `ldap_sd_build_info` metrics to include git hash via the `git_hash` label
- Updated connection and reconnection logic to use a single `connect()` function which uses a connection lock to ensure the operation is thread-safe.
- Improved structure of custom errors and added basic tests for them.
- Added the relevant target group and base DN to warning logs regarding an object missing the `dNSHostName` attribute.

### v0.4.0
- Moved the reconnection logic to a `reconnect`
- Added locks for both LDAP connection attempts and cache updates.   The LDAP store is a global struct being initialized in the main goroutine, thus there could be concurrent attempts being made by different incoming requests.
- Added proper comments to custom errors
- Removed the `s.conn.Close()` function call from the `runDiscovery` function so that the connection can remain active.  Disconnections are automatically dealt with by the `reconnect` function.
- Renamed `ldap_sd_req_from_cache_total` metric to `ldap_sd_cache_hit_total`
- Updated error checking in `/targets` handler function
- Added logic to ensure an existing targetGroup is specified.

### v0.3.0
- Exposed new prometheus metrics including: `ldap_sd_build_info`, `ldap_sd_req_failed_total`, `ldap_sd_req_success_total`, `ldap_sd_req_from_cache_total`, `ldap_sd_cache_update_success_total`, `ldap_sd_cache_update_fail_total`, `ldap_sd_reconnect`
- The `filter` and `attributes` properties are now configurable per target group.
- Created new `getResults` within the **store** package to allow to contain code relevant to fetching paginated result sets
- Enabled profiling via the  `/debug/profile` HTTP endpoint
- Renamed the `/health` endpoint to `/healthz` and implemented a basic verification to return the actual health status of the server.
- Removed handler package and move moved http handlers to main package
- Fixed documentation regarding the configuration options.
- Added custom errors types

### v0.2.0
- Updating exposed labels prefix name from `_meta_<LABEL_NAME>` to `_meta_ldap_<LABEL_NAME>`
- The filter attribute has been removed from the configuration and set as a constant as it will always be `(&(objectClass=computer))`
- Attributes to fetch now always includes `name` and `dNSHostName` as they are required. 
- Updated configuration to allow for specification of exporter port and LDAP attributes for each group of base DNs specified 
- Updated attribute logic so that each object will be set with it's own attribute values.  In the previous version, the attribute value of the last object in the list was incorrectly applied to all resulting targets.

### v0.1.0
- Initial release.
