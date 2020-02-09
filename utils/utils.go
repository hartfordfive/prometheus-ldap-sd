package utils

import (
	"fmt"
	"log"

	"github.com/go-ldap/ldap/v3"
)

// Search performs the search with the supplied parameters to return the matching host.
func LdapSearch(conf LdapConfig) {
	l, err := ldap.DialURL(fmt.Sprintf("ldap://%s:%d", conf.Host, conf.Port))
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	searchRequest := ldap.NewSearchRequest(
		conf.BaseDN, // The base dn to search
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		conf.Filter,     // The filter to apply
		conf.Attributes, // A list attributes to retrieve
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		log.Fatal(err)
	}

	for _, entry := range sr.Entries {
		fmt.Printf("%s: %v\n", entry.DN, entry.GetAttributeValue("cn"))
	}
}
