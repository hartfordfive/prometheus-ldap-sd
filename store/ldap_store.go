package store

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gadelkareem/cachita"
	ldap "github.com/go-ldap/ldap/v3"
	"github.com/hartfordfive/prometheus-ldap-sd-server/config"
	"github.com/hartfordfive/prometheus-ldap-sd-server/logger"
	"github.com/hartfordfive/prometheus-ldap-sd-server/metrics"
	"go.uber.org/zap"
)

var (
	searchPagingSize uint32 = 100
	matchFirstCap           = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap             = regexp.MustCompile("([a-z0-9])([A-Z])")
	baseAttributes          = []string{"name", "dNSHostName"}
)

const (
	ldapFilter           = "(&(objectClass=computer))"
	maxReconnectAttempts = 5
)

type LdapStore struct {
	Config            *config.LdapConfig
	conn              *ldap.Conn
	cache             cachita.Cache
	ReconnectAttempts int
	connLock          sync.Mutex
	cacheLock         sync.Mutex
	isReady           bool
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
	url string,
	bindDN string,
	baseDnMappings map[string]*config.BaseDnMapping,
	defaultAttributes []string,
	passEnvVar string,
	authenticated bool,
	unsecured bool,
	cacheDir string,
	cacheTTL int) (*LdapStore, error) {

	cache, err := cachita.NewFileCache(cacheDir, time.Duration(cacheTTL)*time.Second, 5*time.Minute)
	if err != nil {
		panic(err)
	}

	return &LdapStore{
		conn:              nil,
		ReconnectAttempts: 0,
		Config: &config.LdapConfig{
			URL:                  url,
			BindDN:               bindDN,
			BaseDnMappings:       baseDnMappings,
			DefaultAttributes:    defaultAttributes,
			PasswordEnvVar:       passEnvVar,
			Authenticated:        authenticated,
			Unsecured:            unsecured,
			CacheDir:             cacheDir,
			CacheTTL:             cacheTTL,
			MaxReconnectAttempts: maxReconnectAttempts,
		},
		cache:   cache,
		isReady: false,
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

func (s *LdapStore) reconnect() error {
	// Attempt to reconnect if the connection is no longer valid
	s.connLock.Lock()
	defer s.connLock.Unlock()

	for s.ReconnectAttempts < maxReconnectAttempts {
		s.ReconnectAttempts++
		metrics.MetricReconnect.Inc()
		ldapConn, err := getLdapConn(s.Config.URL, s.Config.BindDN, s.Config.Authenticated, s.Config.PasswordEnvVar, s.Config.Unsecured)
		if err != nil {
			logger.Logger.Warn("Reconnection attempt fialed",
				zap.Int("attempt", s.ReconnectAttempts),
				zap.String("error", err.Error()),
			)
			if s.ReconnectAttempts >= maxReconnectAttempts {
				s.isReady = false
				break
			}

			time.Sleep(5 * time.Second)
			continue
		} else {
			logger.Logger.Debug("LDAP connection re-established")
			s.conn = ldapConn
			return nil
		}
	}
	return &LdapStoreErrorMaxReconnects{}
}

func (s *LdapStore) getResults(targetGroup, baseDn, filter string, attributesList []string) ([]LdapObject, error) {
	var entries []LdapObject
	var obj LdapObject

	if _, err := s.conn.Search(&ldap.SearchRequest{}); err == ldap.ErrNilConnection {
		if err := s.reconnect(); err != nil {
			return entries, err
		}
	}

	search := ldap.NewSearchRequest(
		baseDn,
		ldap.ScopeWholeSubtree,
		0,
		0,
		0,
		false,
		filter, // the filter
		attributesList,
		[]ldap.Control{})

	logger.Logger.Debug("Running SearchWithPaging",
		zap.String("base_dn", baseDn),
		zap.String("filter", filter),
		zap.Any("attributesList", attributesList))

	results, err := s.conn.SearchWithPaging(search, searchPagingSize)

	if err != nil {
		logger.Logger.Error("Could not run search against LDAP",
			zap.String("base_dn", baseDn),
			zap.String("error", err.Error()),
		)
		metrics.MetricServerRequestsFailed.WithLabelValues(targetGroup).Inc()
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

		entries = append(entries, obj)
	}

	return entries, nil

}

func (s *LdapStore) updateCache(targetGroup string, entries []LdapObject, ttl time.Duration) error {
	s.cacheLock.Lock()
	defer s.cacheLock.Unlock()

	err := s.cache.Put(targetGroup, entries, ttl)
	if err != nil {
		logger.Logger.Error("Could not store result set in cache",
			zap.String("cache_key", targetGroup),
			zap.String("error", err.Error()),
		)
		metrics.MetricCacheUpdateFail.WithLabelValues(targetGroup).Inc()
		return &LdapStoreErrorCacheUpdate{}
	} else {
		metrics.MetricCacheUpdateSuccess.WithLabelValues(targetGroup).Inc()
	}
	return nil
}

func (s *LdapStore) runDiscovery(targetGroup string) ([]LdapObject, error) {
	var entries []LdapObject
	var attributesList []string
	var filter string = ldapFilter
	var resultsErr error

	if strings.TrimSpace(targetGroup) == "" {
		return entries, &LdapStoreErrorInvalidTargetGroup{}
	}

	// Fetch objects from cache if they are present and still valid
	err := s.cache.Get(targetGroup, &entries)
	if err != nil && err != cachita.ErrNotFound && err != cachita.ErrExpired {
		logger.Logger.Error("Could not fetch existing cached target group entries from cache",
			zap.Any("error", err.Error()),
		)
		return entries, &LdapStoreErrorCacheFetch{}
	} else if err == cachita.ErrExpired {
		logger.Logger.Debug("Target group cache entries expired. Fetching updated list.",
			zap.String("cache_key", targetGroup),
		)
	} else if err == nil {
		logger.Logger.Debug("Serving target group entries from disk cache",
			zap.String("cache_key", targetGroup),
		)
		metrics.MetricRequestsFromCache.WithLabelValues(targetGroup).Inc()
		return entries, nil
	}

	if _, err := s.conn.Search(&ldap.SearchRequest{}); err == ldap.ErrNilConnection {
		if err := s.reconnect(); err != nil {
			return entries, err
		}
	}

	select {
	default:

		logger.Logger.Debug("Refreshing object listing from LDAP", zap.String("group_name", targetGroup))

		baseDnMapping := s.Config.BaseDnMappings[targetGroup]
		attributesList = append(s.Config.DefaultAttributes, baseAttributes...)

		logger.Logger.Debug("Base DN mapping", zap.Any("base_dn_mapping", baseDnMapping))

		if len(baseDnMapping.Attributes) >= 1 {
			for _, attrib := range baseDnMapping.Attributes {
				attributesList = append(attributesList, attrib)
			}
		}

		if baseDnMapping == nil {
			return entries, &LdapStoreErrorInvalidQuery{}
		}
		if len(baseDnMapping.BaseDnList) == 0 && baseDnMapping.Filter == "" {
			logger.Logger.Error("Could not store result set in cache")
			return entries, &LdapStoreErrorCacheUpdate{}
		}

		if baseDnMapping.Filter == "(&(objectClass=computer))" || (baseDnMapping.Filter == "" && len(baseDnMapping.BaseDnList) == 0) {
			return entries, &LdapStoreErrorInvalidQuery{}
		}

		if baseDnMapping.Filter != "" {
			filter = baseDnMapping.Filter
		}

		if baseDnMapping.Filter != "" && len(baseDnMapping.BaseDnList) == 0 {
			entries, resultsErr = s.getResults(targetGroup, "", filter, attributesList)
			logger.Logger.Debug("Fetching LDAP objects corresponding to custom filter",
				zap.String("targetGroup", targetGroup),
				zap.String("filter", filter),
			)
		} else {
			for _, baseDn := range baseDnMapping.BaseDnList {
				logger.Logger.Debug("Fetching LDAP objects corresponding to base DN and filter",
					zap.String("base_dn", baseDn),
					zap.String("filter", filter),
				)
				entries, resultsErr = s.getResults(targetGroup, baseDn, filter, attributesList)
			}
		}

		if resultsErr != nil {
			metrics.MetricServerRequestsFailed.WithLabelValues(targetGroup).Inc()
		}

		metrics.MetricServerRequests.WithLabelValues(targetGroup).Inc()

	}

	return entries, s.updateCache(targetGroup, entries, time.Duration(s.Config.CacheTTL)*time.Second)

}

// Serialize returns the json representation of the discovered target groups
func (s *LdapStore) Serialize(targetGroup string) (string, error) {

	if _, ok := s.Config.BaseDnMappings[targetGroup]; !ok {
		return "", &LdapStoreErrorInvalidTargetGroup{targetGroup}
	}

	res, err := s.runDiscovery(targetGroup)
	if err != nil {
		return "", err
	}

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

// IsReady exposes if the discovery server is ready or not
func (s *LdapStore) IsReady() bool {
	return s.isReady
}

// Shutdown handles the shutdown procedure of the discovery server.
func (s *LdapStore) Shutdown() {
	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}
}
