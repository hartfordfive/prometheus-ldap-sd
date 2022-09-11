# Changelog


### v0.3.0
- Exposed new prometheus metrics including: `ldap_sd_build_info`, `ldap_sd_req_failed_total`, `ldap_sd_req_success_total`, `ldap_sd_req_from_cache_total`, `ldap_sd_cache_update_success_total`, `ldap_sd_cache_update_fail_total`, `ldap_sd_reconnect`
- The `filter` and `attributes` properties are now configurable per target group.
- Created new `getResults` within the **store** package to allow to contain code relevant to fetching paginated result sets
- Enabled profiling via the  `/debug/profile` HTTP endpoint
- Renamed the `/health` endpoint to `/healthz` and implemented a basic verification to return the actual health status of the server.

### v0.2.0
- Updating exposed labels prefix name from `_meta_<LABEL_NAME>` to `_meta_ldap_<LABEL_NAME>`
- The filter attribute has been removed from the configuration and set as a constant as it will always be `(&(objectClass=computer))`
- Attributes to fetch now always includes `name` and `dNSHostName` as they are required. 
- Updated configuration to allow for specification of exporter port and LDAP attributes for each group of base DNs specified 
- Updated attribute logic so that each object will be set with it's own attribute values.  In the previous version, the attribute value of the last object in the list was incorrectly applied to all resulting targets.

### v0.1.0
- Initial release.
