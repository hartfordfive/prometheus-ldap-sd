package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/hartfordfive/prometheus-ldap-sd-server/config"
	"github.com/hartfordfive/prometheus-ldap-sd-server/handler"
	"github.com/hartfordfive/prometheus-ldap-sd-server/logger"
	"github.com/hartfordfive/prometheus-ldap-sd-server/store"
	"github.com/hartfordfive/prometheus-ldap-sd-server/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var dataStore store.DataStore

var (
	metricHttpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "ldap_sd_req_duration_seconds",
		Help: "Duration of HTTP requests.",
	}, []string{"path"})
)

var (
	flagConfPath       *string
	flagDebug          *bool
	flagVersion        *bool
	flagValidateConfig *bool
	log                *zap.Logger
	conf               *config.Config
	shutdownChan       chan bool
	interruptChan      chan os.Signal
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

	interruptChan = make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGTERM)

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
		route := mux.CurrentRoute(r)
		path, _ := route.GetPathTemplate()
		timer := prometheus.NewTimer(metricHttpDuration.WithLabelValues(path))
		next.ServeHTTP(w, r)
		timer.ObserveDuration()
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
	shutdownChan = make(chan bool)

	logger.Logger.Info("Starting server")

	config.GlobalConfig = conf

	logger.Logger.Info("Setting cache TTL",
		zap.Int("seconds", conf.LdapConfig.CacheTTL),
	)

	// Init datastore
	logger.Logger.Sugar().Infof("Starting LDAP datastore")
	store.StoreInstance, err = store.NewLdapStore(
		shutdownChan,
		conf.LdapConfig.URL,
		conf.LdapConfig.BindDN,
		conf.LdapConfig.BaseDnMapping,
		conf.LdapConfig.GroupExporterPortMapping,
		conf.LdapConfig.Filter,
		conf.LdapConfig.Attributes,
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

	// Init web server
	r := mux.NewRouter()
	r.HandleFunc("/targets", handler.ShowTargetsHandler).Methods("GET")
	r.HandleFunc("/config", handler.ShowConfigHandler).Methods("GET")
	r.Handle("/metrics", promhttp.Handler()).Methods("GET")
	r.HandleFunc("/health", handler.HealthHandler).Methods("GET")

	listenAddr := fmt.Sprintf("%s:%d", conf.Host, conf.Port)

	srv := &http.Server{
		Handler:      r,
		Addr:         listenAddr,
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}

	go func() {
		logger.Logger.Info("Running discovery server",
			zap.String("address", listenAddr),
		)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			logger.Logger.Fatal(fmt.Sprintf("Error starting admin server: %v", err))
		}
	}()

	killSig := <-interruptChan
	switch killSig {
	case os.Interrupt, syscall.SIGTERM:
		logger.Logger.Info("Received shutdown signal")
		close(shutdownChan)
	}

	store.StoreInstance.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err := srv.Shutdown(ctx); err != nil {
		panic(err)
	}

	logger.Logger.Info("Server shutdown complete")
}
