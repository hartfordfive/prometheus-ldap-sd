package store

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gadelkareem/cachita"
	ldap "github.com/go-ldap/ldap/v3"
	"github.com/hartfordfive/prometheus-ldap-sd-server/config"
	"github.com/hartfordfive/prometheus-ldap-sd-server/logger"
	"go.uber.org/zap"
)

var (
	searchPagingSize uint32 = 100
	matchFirstCap           = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap             = regexp.MustCompile("([a-z0-9])([A-Z])")
)

type LdapStore struct {
	shutdownChan chan bool
	Config       *config.LdapConfig
	conn         *ldap.Conn
	cache        cachita.Cache
}

type LdapObject struct {
	Hostname   string
	Attributes map[string]string
}

type TargetGroup struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

func keyToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func NewLdapStore(
	shutdownChan chan bool,
	url string,
	bindDN string,
	baseDnMapping map[string][]string,
	groupExporterPortMapping map[string]int,
	filter string,
	attributes []string,
	passEnvVar string,
	authenticated bool,
	unsecured bool,
	cacheDir string,
	cacheTTL int) (*LdapStore, error) {

	conn, err := getLdapConn(url, bindDN, authenticated, passEnvVar, unsecured)
	if err != nil {
		return nil, err
	}
	cache, err := cachita.NewFileCache(cacheDir, time.Duration(cacheTTL)*time.Second, 5*time.Minute)
	if err != nil {
		panic(err)
	}

	return &LdapStore{
		shutdownChan: shutdownChan,
		conn:         conn,
		Config: &config.LdapConfig{
			URL:                      url,
			BindDN:                   bindDN,
			BaseDnMapping:            baseDnMapping,
			GroupExporterPortMapping: groupExporterPortMapping,
			Filter:                   filter,
			Attributes:               attributes,
			PasswordEnvVar:           passEnvVar,
			Authenticated:            authenticated,
			Unsecured:                unsecured,
			CacheDir:                 cacheDir,
			CacheTTL:                 cacheTTL,
		},
		cache: cache,
	}, nil

}

func getLdapConn(ldapURL, bindDN string, authenticated bool, passEnvVar string, unsecured bool) (*ldap.Conn, error) {
	logger.Logger.Debug("Dialing LDAP host", zap.String("host", ldapURL))
	l, err := ldap.DialURL(fmt.Sprintf("ldap://%s", ldapURL))
	if err != nil {
		return nil, fmt.Errorf("error connecting to %v", err)
	}
	l.SetTimeout(5 * time.Second)
	if unsecured {
		err = l.StartTLS(&tls.Config{InsecureSkipVerify: true})
		if err != nil {
			return nil, fmt.Errorf("error upgrading connection to TLS: %v", err)
		}
	}
	if !authenticated {
		err = l.UnauthenticatedBind(bindDN)
		if err != nil {
			return nil, fmt.Errorf("error performing unauthenticated bind: %v", err)
		}
	} else {
		err = l.Bind(bindDN, os.Getenv(passEnvVar))
		if err != nil {
			return nil, fmt.Errorf("error performing authenticated bind: %v", err)
		}
	}

	return l, nil
}

func (s *LdapStore) RunDiscovery(targetGroup string) ([]LdapObject, error) {
	var entries []LdapObject

	// Fetch objects from cache if they are present and still valid
	err := s.cache.Get(targetGroup, &entries)
	if err != nil && err != cachita.ErrNotFound && err != cachita.ErrExpired {
		logger.Logger.Error("Could not fetch existing cached target group entries from cache",
			zap.Any("error", err.Error()),
		)
	} else if len(entries) >= 1 {
		logger.Logger.Debug("Serving target group entries from disk cache",
			zap.String("cache_key", targetGroup),
		)
		return entries, nil
	}

	if err == cachita.ErrExpired {
		logger.Logger.Debug("Target group cache entries expired. Fetching updated list.",
			zap.String("cache_key", targetGroup),
		)
	}

	select {
	case <-s.shutdownChan:
		logger.Logger.Error("Discovery refresh cancelled")
	default:

		logger.Logger.Info("Refreshing object listing from LDAP", zap.String("group_name", targetGroup))

		for _, baseDn := range s.Config.BaseDnMapping[targetGroup] {
			logger.Logger.Debug("Getting LDAP objects corresponding to base DN",
				zap.String("base_dn", baseDn),
			)
			search := ldap.NewSearchRequest(
				baseDn,
				ldap.ScopeWholeSubtree,
				0,
				0,
				0,
				false,
				s.Config.Filter, // the filter
				s.Config.Attributes,
				[]ldap.Control{})

			logger.Logger.Debug("Runing SearchWithPaging")
			results, err := s.conn.SearchWithPaging(search, searchPagingSize)

			if err != nil {
				logger.Logger.Error("Could not run search against LDAP",
					zap.String("base_dn", baseDn),
					zap.String("error", err.Error()),
				)
				return []LdapObject{}, err
			}

			logger.Logger.Debug("Building results from discovered objects",
				zap.String("base_dn", baseDn),
				zap.Int("total_objects", len(results.Entries)),
			)
			var obj LdapObject
			for _, e := range results.Entries {
				if e.GetAttributeValue("dNSHostName") == "" {
					continue
				}
				obj = LdapObject{
					Hostname:   e.GetAttributeValue("name"),
					Attributes: map[string]string{},
				}
				if len(s.Config.Attributes) >= 1 {
					for _, attrib := range s.Config.Attributes {
						if attrib != "name" {
							obj.Attributes[attrib] = e.GetAttributeValue(attrib)
						}
					}
				}
				//logger.Logger.Debug("Printing object", zap.Any("object", obj))
				entries = append(entries, obj)
			}

		}
	}

	err = s.cache.Put(targetGroup, entries, time.Duration(s.Config.CacheTTL)*time.Second)
	if err != nil {
		logger.Logger.Error("Could not store result set in cache",
			zap.String("cache_key", targetGroup),
			zap.String("error", err.Error()),
		)
	}

	return entries, nil

}

func (s *LdapStore) Serialize(targetGroup string) (string, error) {

	res, _ := s.RunDiscovery(targetGroup)
	tg := TargetGroup{
		Targets: []string{},
		Labels:  map[string]string{},
	}

	for _, ldapObject := range res {
		tg.Targets = append(tg.Targets, strings.Join(
			[]string{
				ldapObject.Attributes["dNSHostName"],
				strconv.Itoa(s.Config.GroupExporterPortMapping[targetGroup]),
			}, ":"))
		for k, v := range ldapObject.Attributes {
			if k == "dNSHostName" {
				continue
			}
			if _, ok := tg.Labels[k]; !ok {
				tg.Labels[fmt.Sprintf("__meta_%s", keyToSnakeCase(k))] = v
			}
		}
	}

	output, _ := json.Marshal([]TargetGroup{tg})
	return string(output), nil

}

func (s *LdapStore) Shutdown() {
	s.conn.Close()
}
