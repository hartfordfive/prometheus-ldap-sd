package utils

import (
	"fmt"
	"log"

	"github.com/go-ldap/ldap/v3"
)

// Search performs the search with the supplied parameters to return the matching host.
func Search(conf LdapConfig) {
	l, err := ldap.DialURL(fmt.Sprintf("ldap://%s:%d", conf.Host, conf.Port))
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	searchRequest := ldap.NewSearchRequest(
		"dc=example,dc=com", // The base dn to search
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		"(&(objectClass=organizationalPerson))", // The filter to apply
		[]string{"dn", "cn"},                    // A list attributes to retrieve
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
