package main

import (
	"context"
	"flag"
	"fmt"
	defaultLogger "log"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/gorilla/mux"
	"github.com/hartfordfive/prometheus-ldap-sd-server/config"
	"github.com/hartfordfive/prometheus-ldap-sd-server/logger"
	"github.com/hartfordfive/prometheus-ldap-sd-server/metrics"
	"github.com/hartfordfive/prometheus-ldap-sd-server/store"
	"github.com/hartfordfive/prometheus-ldap-sd-server/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var dataStore store.DataStore

var (
	flagConfPath       *string
	flagDebug          *bool
	flagVersion        *bool
	flagValidateConfig *bool
	log                *zap.Logger
	conf               *config.Config
)

func init() {

	flagConfPath = flag.String(
		"conf",
		"/etc/prometheus-ldap-sd-server/server.conf",
		"Path to the configuration file",
	)
	flagVersion = flag.Bool("version", false, "Show version and exit")
	flagDebug = flag.Bool("debug", false, "Enable debug mode")
	flagValidateConfig = flag.Bool("validate", false, "Validate config and exit")
	flag.Parse()

	prometheus.Register(metrics.MetricBuildInfo)
	prometheus.Register(metrics.MetricServerRequestsFailed)
	prometheus.Register(metrics.MetricServerRequests)
	prometheus.Register(metrics.MetricRequestsFromCache)
	prometheus.Register(metrics.MetricCacheUpdateSuccess)
	prometheus.Register(metrics.MetricCacheUpdateFail)
	prometheus.Register(metrics.MetricReconnect)
	prometheus.Register(metrics.MetricGroupNumObjects)

	var log *zap.Logger
	var loggerErr error

	atom := zap.NewAtomicLevel()

	log = zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.Lock(os.Stdout),
		atom,
	))
	defer log.Sync()

	//log, loggerErr = zap.NewProduction() // or NewExample, NewProduction, or NewDevelopment

	if *flagDebug {
		atom.SetLevel(zap.DebugLevel)
		// 	// If we're in debug mode, then create a dev logger instead
		// 	log, loggerErr = zap.NewDevelopment() // or NewExample, NewProduction, or NewDevelopment
	}

	if loggerErr != nil {
		fmt.Printf("Could not initialize logger: %s\n", loggerErr)
		os.Exit(1)
	}
	logger.Logger = log
	defer logger.Logger.Sync()

	if *flagVersion {
		fmt.Printf("prometheus-ldap-sd-server %s (Git hash: %s)\n", version.Version, version.CommitHash)
		os.Exit(0)
	}

	cnf, err := config.NewConfig(*flagConfPath)

	if err != nil {
		logger.Logger.Error(fmt.Sprintf("%s", err))
		os.Exit(1)
	}
	conf = cnf

}

// prometheusMiddleware implements mux.MiddlewareFunc.
func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := metrics.NewResponseWriter(w)
		next.ServeHTTP(rw, r)
	})
}

func validateConfig(cnf *config.Config) int {
	if cnf.Validate() != nil {
		return 1
	}
	return 0
}

func main() {

	if *flagValidateConfig {
		logger.Logger.Info("Validating configuration", zap.String("path", *flagConfPath))
		res := validateConfig(conf)
		if res == 0 {
			logger.Logger.Info("Configuration OK")
		}
		os.Exit(res)
	}

	var err error

	logger.Logger.Info("Starting server")

	config.GlobalConfig = conf

	logger.Logger.Debug(fmt.Sprintf("Cache TTL set to %ds", conf.LdapConfig.CacheTTL))

	// Init datastore
	store.StoreInstance, err = store.NewLdapStore(
		conf.LdapConfig.URL,
		conf.LdapConfig.BindDN,
		conf.LdapConfig.BaseDnMappings,
		conf.LdapConfig.Filter,
		conf.LdapConfig.DefaultAttributes,
		conf.LdapConfig.PasswordEnvVar,
		conf.LdapConfig.Authenticated,
		conf.LdapConfig.Unsecured,
		conf.LdapConfig.CacheDir,
		conf.LdapConfig.CacheTTL,
	)
	if err != nil {
		logger.Logger.Error(err.Error())
		os.Exit(1)
	}

	// The ldap package Logger function can't use the zap.Logger struct, so we have to create a seperate one
	ldap.Logger(defaultLogger.New(os.Stdout, "", defaultLogger.LstdFlags))

	metrics.MetricBuildInfo.WithLabelValues(version.Version, version.CommitHash).Inc()

	for targetGroup, _ := range conf.LdapConfig.BaseDnMappings {
		metrics.MetricServerRequestsFailed.WithLabelValues(targetGroup)
		metrics.MetricServerRequests.WithLabelValues(targetGroup)
		metrics.MetricRequestsFromCache.WithLabelValues(targetGroup)
		metrics.MetricCacheUpdateSuccess.WithLabelValues(targetGroup)
		metrics.MetricCacheUpdateFail.WithLabelValues(targetGroup)
		metrics.MetricReconnect.Add(0)
		metrics.MetricGroupNumObjects.WithLabelValues(targetGroup).Add(0)
	}

	listenAddr := fmt.Sprintf("%s:%d", conf.Host, conf.Port)

	r := mux.NewRouter()
	srv := &http.Server{
		Handler:      r,
		Addr:         listenAddr,
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}

	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGTERM)

	r.HandleFunc("/targets", func(w http.ResponseWriter, req *http.Request) {
		logger.Logger.Debug("Target listing requested", zap.String("remote_addr", req.RemoteAddr))

		w.Header().Set("Content-Type", "application/json")
		targetGroup := req.URL.Query().Get("targetGroup")
		res, err := store.StoreInstance.Serialize(targetGroup)

		if err != nil {
			logger.Logger.Error(err.Error())
			storeError, isStoreErr := err.(*store.Error)
			if isStoreErr {
				if storeError.Code == store.LdapStoreErrorMaxReconnects {
					interruptChan <- syscall.SIGTERM
					return
				}
			}
			http.Error(w, "[]", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "%s\n", res)

	}).Methods("GET")

	r.HandleFunc("/config", func(w http.ResponseWriter, req *http.Request) {
		logger.Logger.Info("Debug config requested")
		w.Header().Set("Content-Type", "text/yaml")
		printCnf, err := config.GlobalConfig.Serialize()

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "%s\n", printCnf)
	}).Methods("GET")

	r.HandleFunc("/healthz", func(w http.ResponseWriter, req *http.Request) {
		dataStore := store.StoreInstance
		if dataStore.IsReady() {
			fmt.Fprint(w, "OK")
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "ERROR")
		}
	}).Methods("GET")

	r.Handle("/metrics", promhttp.Handler()).Methods("GET")
	r.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	r.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	r.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	r.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	r.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	r.Handle("/debug/pprof/{cmd}", http.HandlerFunc(pprof.Index)) // special handling for Gorilla mux

	go func() {
		logger.Logger.Info("Running discovery server",
			zap.String("address", listenAddr),
		)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			logger.Logger.Fatal(fmt.Sprintf("Error starting admin server: %v", err))
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		for {
			select {
			case killSig := <-interruptChan:
				if killSig == os.Interrupt || killSig == syscall.SIGTERM {
					logger.Logger.Info("Received shutdown notification")
					wg.Done()
					return
				}
			}
		}
	}()

	wg.Wait()

	logger.Logger.Info("Shutting down server")
	store.StoreInstance.Shutdown()

	ctx, cancelHTTPServer := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancelHTTPServer()
	}()

	if err := srv.Shutdown(ctx); err != nil {
		panic(err)
	}

	logger.Logger.Info("Server shutdown complete")
}
