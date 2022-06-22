package store

import (
	"crypto/tls"
	"encoding/json"
	"errors"
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
	baseAttributes          = []string{"name", "dNSHostName"}
)

const ldapFilter = "(&(objectClass=computer))"

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

func isBaseAttribute(name string, baseAttributes []string) bool {
	for _, v := range baseAttributes {
		if v == name {
			return true
		}
	}
	return false
}

func NewLdapStore(
	shutdownChan chan bool,
	url string,
	bindDN string,
	baseDnMappings map[string]*config.BaseDnMapping,
	defaultAttributes []string,
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
			URL:               url,
			BindDN:            bindDN,
			BaseDnMappings:    baseDnMappings,
			DefaultAttributes: defaultAttributes,
			PasswordEnvVar:    passEnvVar,
			Authenticated:     authenticated,
			Unsecured:         unsecured,
			CacheDir:          cacheDir,
			CacheTTL:          cacheTTL,
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
	var attributesList []string
	var obj LdapObject

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

		logger.Logger.Debug("Refreshing object listing from LDAP", zap.String("group_name", targetGroup))

		baseDnMapping := s.Config.BaseDnMappings[targetGroup]
		attributesList = append(s.Config.DefaultAttributes, baseAttributes...)

		if len(baseDnMapping.Attributes) >= 1 {
			for _, attrib := range baseDnMapping.Attributes {
				attributesList = append(attributesList, attrib)
			}
		}

		if baseDnMapping == nil {
			return entries, errors.New("No base DNs specified for group")
		}

		for _, baseDn := range baseDnMapping.BaseDnList {
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
				ldapFilter, // the filter
				attributesList,
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

			obj = LdapObject{}
			for _, e := range results.Entries {
				if e.GetAttributeValue("dNSHostName") == "" {
					logger.Logger.Warn("Skipping object as it's missing the dNSHostName attribute", zap.Any("name", e.GetAttributeValue("name")))
					continue
				}
				obj = LdapObject{
					Hostname:   e.GetAttributeValue("name"),
					Attributes: map[string]string{},
				}

				if len(attributesList) >= 1 {
					for _, attrib := range attributesList {
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

	tgList := []TargetGroup{}

	for _, ldapObject := range res {

		tg := TargetGroup{
			Targets: []string{},
			Labels:  map[string]string{},
		}

		tg.Targets = append(tg.Targets, strings.Join(
			[]string{
				ldapObject.Attributes["dNSHostName"],
				strconv.Itoa(s.Config.BaseDnMappings[targetGroup].ExporterPort),
			}, ":"))

		for k, v := range ldapObject.Attributes {
			if isBaseAttribute(k, baseAttributes) {
				continue
			}
			if _, ok := tg.Labels[k]; !ok {
				tg.Labels[fmt.Sprintf("__meta_ldap_%s", keyToSnakeCase(k))] = v
			}
		}

		tgList = append(tgList, tg)
	}

	output, _ := json.Marshal(tgList)
	return string(output), nil

}

func (s *LdapStore) Shutdown() {
	s.conn.Close()
}
