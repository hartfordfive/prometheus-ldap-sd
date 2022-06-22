# Changelog


### v0.2.0
- Updating exposed labels prefix name from `_meta_<LABEL_NAME>` to `_meta_ldap_<LABEL_NAME>`
- The filter attribute has been removed from the configuration and set as a constant as it will always be `(&(objectClass=computer))`
- Attributes to fetch now always includes `name` and `dNSHostName` as they are required. 
- Updated configuration to allow for specification of exporter port and LDAP attributes for each group of base DNs specified 
- Updated attribute logic so that each object will be set with it's own attribute values.  In the previous version, the attribute value of the last object in the list was incorrectly applied to all resulting targets.

### v0.1.0
- Initial release.
